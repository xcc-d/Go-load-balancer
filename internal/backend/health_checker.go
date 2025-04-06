package backend

import (
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	DefaultRetryCount    = 3
	DefaultRetryInterval = 5 * time.Second
)

// HealthChecker 定期检查后端服务器健康状态
type HealthChecker struct {
	pool            *Pool
	timeout         time.Duration
	stopCh          chan struct{}
	retryCount      int
	checkInProgress atomic.Bool // 防止健康检查并发执行
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(pool *Pool, timeout time.Duration, retryCount int) *HealthChecker {
	if retryCount <= 0 {
		retryCount = DefaultRetryCount
	}
	return &HealthChecker{
		pool:       pool,
		timeout:    timeout,
		stopCh:     make(chan struct{}),
		retryCount: retryCount,
	}
}

// checkTCP 执行TCP健康检查
func (hc *HealthChecker) checkTCP(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, hc.timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkHTTP 执行HTTP健康检查
func (hc *HealthChecker) checkHTTP(url string) bool {
	client := http.Client{Timeout: hc.timeout}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// performCheckWithRetry 带重试的健康检查(非阻塞版)
func (hc *HealthChecker) performCheckWithRetry(checkFunc func() bool) bool {
	resultChan := make(chan bool, 1)
	timeoutChan := time.After(hc.timeout * time.Duration(hc.retryCount)) // 减少总超时时间

	go func() {
		for i := 0; i < hc.retryCount; i++ {
			// 设置单次检查超时
			checkDone := make(chan bool, 1)

			go func() {
				checkDone <- checkFunc()
			}()

			var success bool

			// 等待单次检查结果或超时
			select {
			case result := <-checkDone:
				success = result
			case <-time.After(hc.timeout):
				// 单次检查超时
				log.Printf("单次健康检查超时(尝试 %d/%d)", i+1, hc.retryCount)
				success = false
			}

			if success {
				log.Printf("健康检查成功(尝试 %d/%d)", i+1, hc.retryCount)
				resultChan <- true
				return
			}

			if i < hc.retryCount-1 {
				log.Printf("健康检查失败(尝试 %d/%d)，%s后重试",
					i+1, hc.retryCount, DefaultRetryInterval)
				select {
				case <-time.After(DefaultRetryInterval):
					continue
				case <-hc.stopCh:
					resultChan <- false
					return
				}
			}
		}
		log.Printf("健康检查失败(已达最大重试次数 %d)", hc.retryCount)
		resultChan <- false
	}()

	select {
	case result := <-resultChan:
		return result
	case <-timeoutChan:
		log.Printf("健康检查总超时(超过 %v)", hc.timeout*time.Duration(hc.retryCount))
		return false
	case <-hc.stopCh:
		return false
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("健康检查goroutine panic: %v", r)
			}
		}()

		for {
			select {
			case <-ticker.C:
				// 如果上一次健康检查还在执行，跳过此次检查
				if hc.checkInProgress.Load() {
					log.Println("上一次健康检查仍在进行中，跳过本次检查")
					continue
				}

				hc.checkInProgress.Store(true)
				done := make(chan struct{})

				go func() {
					defer close(done)
					defer hc.checkInProgress.Store(false)
					hc.pool.HealthCheck(hc)
				}()

				// 设置更严格的超时
				select {
				case <-done:
					log.Println("执行健康检查完成")
				case <-time.After(5 * time.Second): // 更短的超时时间
					log.Println("健康检查监视超时，可能阻塞")
					// 尽管超时，我们不中止健康检查，让它在后台继续，但下一次检查会跳过
				}
			case <-hc.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop 停止健康检查
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}
