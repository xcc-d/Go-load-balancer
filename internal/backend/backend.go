package backend

import (
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Backend 表示一个后端服务器
type Backend struct {
	URL         *url.URL
	Alive       bool
	Weight      int
	mux         sync.RWMutex
	connections int64
}

// NewBackend 创建一个新的后端服务器实例
func NewBackend(rawURL string, weight int) (*Backend, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, err
	}

	return &Backend{
		URL:    parsedURL,
		Alive:  true,
		Weight: weight,
	}, nil
}

// IsAlive 检查后端是否存活
func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.Alive
}

// SetAlive 设置后端存活状态
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Alive = alive
}

// GetConnections 获取当前连接数
func (b *Backend) GetConnections() int64 {
	return atomic.LoadInt64(&b.connections)
}

// IncrementConnections 增加连接数
func (b *Backend) IncrementConnections() {
	atomic.AddInt64(&b.connections, 1)
}

// DecrementConnections 减少连接数
func (b *Backend) DecrementConnections() {
	atomic.AddInt64(&b.connections, -1)
}

// HealthCheck 执行健康检查
func (b *Backend) HealthCheck(timeout time.Duration) bool {
	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(b.URL.String() + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		b.SetAlive(false)
		return false
	}

	b.SetAlive(true)
	return true
}
