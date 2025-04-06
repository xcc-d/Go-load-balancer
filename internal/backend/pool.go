package backend

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Pool 管理后端服务器池
type Pool struct {
	activeBackends []*Backend // 正常服务器
	retryBackends  []*Backend // 重试服务器
	current        uint64
	mux            sync.RWMutex
}

// NewPool 创建新的后端服务器池
func NewPool(backends []*Backend) *Pool {
	p := &Pool{}
	for _, b := range backends {
		if b.IsAlive() {
			p.activeBackends = append(p.activeBackends, b)
		} else {
			p.retryBackends = append(p.retryBackends, b)
		}
	}
	return p
}

// GetNextPeer 使用改进的无锁轮询算法获取下一个可用后端
func (p *Pool) GetNextPeer() (*Backend, error) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	if len(p.activeBackends) == 0 {
		return nil, errors.New("没有可用的后端服务器")
	}

	start := int(atomic.AddUint64(&p.current, 1))
	end := start + len(p.activeBackends)

	for i := start; i < end; i++ {
		idx := i % len(p.activeBackends)
		if p.activeBackends[idx].IsAlive() {
			return p.activeBackends[idx], nil
		}
	}
	return nil, errors.New("所有后端服务器不可用")
}

// AddBackend 添加新的后端到池中
func (p *Pool) AddBackend(backend *Backend) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if backend.IsAlive() {
		p.activeBackends = append(p.activeBackends, backend)
	} else {
		p.retryBackends = append(p.retryBackends, backend)
	}
}

// GetBackends 获取所有后端（包括活跃和重试）
func (p *Pool) GetBackends() []*Backend {
	p.mux.RLock()
	defer p.mux.RUnlock()

	all := make([]*Backend, 0, len(p.activeBackends)+len(p.retryBackends))
	all = append(all, p.activeBackends...)
	all = append(all, p.retryBackends...)
	return all
}

// GetAll 获取所有后端服务器，供外部调用
func (p *Pool) GetAll() []*Backend {
	return p.GetBackends()
}

// MaxFailures 定义健康检查最大失败次数
var MaxFailures = 3 // 默认值，会被配置文件覆盖

// HealthCheck 对所有后端执行健康检查
func (p *Pool) HealthCheck(checker *HealthChecker) {
	// 获取当前所有后端的快照，减少锁持有时间
	p.mux.RLock()
	activeBackendsCopy := make([]*Backend, len(p.activeBackends))
	retryBackendsCopy := make([]*Backend, len(p.retryBackends))
	copy(activeBackendsCopy, p.activeBackends)
	copy(retryBackendsCopy, p.retryBackends)
	p.mux.RUnlock()

	// 用于收集检查结果
	type checkResult struct {
		backend *Backend
		isAlive bool
	}

	// 创建工作组进行并行健康检查
	var wg sync.WaitGroup
	resultChan := make(chan checkResult, len(activeBackendsCopy)+len(retryBackendsCopy))

	// 检查活跃池中的服务器(并行)
	for _, b := range activeBackendsCopy {
		wg.Add(1)
		go func(backend *Backend) {
			defer wg.Done()
			isAlive := p.checkBackend(backend, checker)
			resultChan <- checkResult{backend: backend, isAlive: isAlive}
		}(b)
	}

	// 检查重试池中的服务器(并行)
	for _, b := range retryBackendsCopy {
		wg.Add(1)
		go func(backend *Backend) {
			defer wg.Done()
			isAlive := p.checkBackend(backend, checker)
			resultChan <- checkResult{backend: backend, isAlive: isAlive}
		}(b)
	}

	// 等待所有健康检查完成并关闭结果通道
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集所有结果
	activeResults := make([]checkResult, 0)
	retryResults := make([]checkResult, 0)

	for result := range resultChan {
		// 根据backend当前所在池分类结果
		isRetry := false
		for _, b := range retryBackendsCopy {
			if b == result.backend {
				isRetry = true
				break
			}
		}

		if isRetry {
			retryResults = append(retryResults, result)
		} else {
			activeResults = append(activeResults, result)
		}
	}

	// 一次性更新池状态，减少锁争用
	p.mux.Lock()
	defer p.mux.Unlock()

	// 处理活跃池结果
	for _, result := range activeResults {
		if !result.isAlive {
			// 从活跃池移除
			for i, b := range p.activeBackends {
				if b == result.backend {
					p.activeBackends = append(p.activeBackends[:i], p.activeBackends[i+1:]...)
					break
				}
			}
			// 加入重试池
			result.backend.SetStatus(StatusRetrying)
			p.retryBackends = append(p.retryBackends, result.backend)
			log.Printf("服务 %s 移入重试池", result.backend.URL.Host)
		}
	}

	// 处理重试池结果
	for _, result := range retryResults {
		if result.isAlive {
			// 从重试池移除
			for i, b := range p.retryBackends {
				if b == result.backend {
					p.retryBackends = append(p.retryBackends[:i], p.retryBackends[i+1:]...)
					break
				}
			}
			// 加入活跃池
			result.backend.SetStatus(StatusActive)
			p.activeBackends = append(p.activeBackends, result.backend)
			log.Printf("服务 %s 恢复并移入活跃池", result.backend.URL.Host)
		} else if result.backend.FailureCount >= MaxFailures { // 修改为直接使用MaxFailures而不是*2
			// 彻底移除
			for i, b := range p.retryBackends {
				if b == result.backend {
					p.retryBackends = append(p.retryBackends[:i], p.retryBackends[i+1:]...)
					break
				}
			}
			log.Printf("服务 %s 连续失败 %d 次，已从池中移除", result.backend.URL.Host, result.backend.FailureCount)
		}
	}
}

// checkBackend 执行健康检查并更新状态
func (p *Pool) checkBackend(b *Backend, checker *HealthChecker) bool {
	// 设置单个后端的健康检查超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 使用通道来实现超时控制
	resultChan := make(chan bool, 1)

	go func() {
		var isAlive bool

		// 根据配置选择检查方式
		path := b.HealthCheckPath
		if path != "" {
			// HTTP检查
			url := b.URL.String() + path
			log.Printf("健康检查 - URL: %s", url)
			isAlive = checker.checkHTTP(url)
		} else {
			// TCP检查
			addr := b.URL.Host
			log.Printf("健康检查 - 地址: %s", addr)
			isAlive = checker.checkTCP(addr)
		}

		resultChan <- isAlive
	}()

	// 等待结果或超时
	select {
	case isAlive := <-resultChan:
		if isAlive {
			b.SetStatus(StatusActive)
			b.FailureCount = 0
		} else {
			b.FailureCount++
		}
		return isAlive
	case <-ctx.Done():
		// 健康检查超时，视为失败
		log.Printf("健康检查超时: %s", b.URL.Host)
		b.FailureCount++
		return false
	}
}
