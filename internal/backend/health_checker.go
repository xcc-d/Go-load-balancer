package backend

import (
	"log"
	"time"
)

// HealthChecker 定期检查后端服务器健康状态
type HealthChecker struct {
	pool    *Pool
	timeout time.Duration
	stopCh  chan struct{}
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(pool *Pool, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		pool:    pool,
		timeout: timeout,
		stopCh:  make(chan struct{}),
	}
}

// Start 启动健康检查
func (hc *HealthChecker) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				hc.pool.HealthCheck(hc.timeout)
				log.Println("执行健康检查完成")
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
