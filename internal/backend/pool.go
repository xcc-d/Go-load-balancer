package backend

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Pool 管理后端服务器池
type Pool struct {
	backends []*Backend
	current  uint64
	mux      sync.RWMutex
}

// NewPool 创建新的后端服务器池
func NewPool(backends []*Backend) *Pool {
	return &Pool{
		backends: backends,
	}
}

// GetNextPeer 使用轮询算法获取下一个可用后端
func (p *Pool) GetNextPeer() (*Backend, error) {
	p.mux.Lock()
	defer p.mux.Unlock()

	// 遍历所有后端，找到第一个存活的
	for i := 0; i < len(p.backends); i++ {
		next := int(atomic.AddUint64(&p.current, 1)) % len(p.backends)
		if p.backends[next].IsAlive() {
			return p.backends[next], nil
		}
	}

	return nil, errors.New("没有可用的后端服务器")
}

// AddBackend 添加新的后端到池中
func (p *Pool) AddBackend(backend *Backend) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.backends = append(p.backends, backend)
}

// GetBackends 获取所有后端
func (p *Pool) GetBackends() []*Backend {
	p.mux.RLock()
	defer p.mux.RUnlock()
	return p.backends
}

// HealthCheck 对所有后端执行健康检查
func (p *Pool) HealthCheck(timeout time.Duration) {
	p.mux.Lock()
	defer p.mux.Unlock()

	for _, b := range p.backends {
		b.HealthCheck(timeout)
	}
}
