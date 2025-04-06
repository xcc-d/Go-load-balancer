package backend

import (
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Backend 表示一个后端服务器
type Backend struct {
	URL             *url.URL
	Status          string // "active", "retrying", "failed"
	Weight          int
	HealthCheckPath string `yaml:"health_check_path" mapstructure:"health_check_path"`
	FailureCount    int    // 连续失败次数
	mux             sync.RWMutex
	connections     int64
	retryCh         chan struct{}
}

// NewBackend 创建一个新的后端服务器实例
func NewBackend(rawURL string, weight int) (*Backend, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, err
	}

	return &Backend{
		URL:     parsedURL,
		Status:  StatusActive,
		Weight:  weight,
		retryCh: make(chan struct{}),
	}, nil
}

// IsAlive 检查后端是否存活
func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.Status == StatusActive
}

// SetStatus 设置后端状态
func (b *Backend) SetStatus(status string) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Status = status
	if status == StatusActive {
		b.FailureCount = 0
	}
}

// SetAlive 兼容旧接口
func (b *Backend) SetAlive(alive bool) {
	if alive {
		b.SetStatus(StatusActive)
	} else {
		b.SetStatus(StatusRetrying)
	}
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

// Addr 获取后端服务器地址(host:port)
func (b *Backend) Addr() string {
	return b.URL.Host
}

// HealthCheck 执行健康检查(兼容旧接口)
func (b *Backend) HealthCheck(timeout time.Duration) bool {
	checker := &HealthChecker{
		timeout:    timeout,
		retryCount: 1,
	}

	path := b.HealthCheckPath
	if path == "" {
		path = "/health" // 默认路径
	}
	return checker.performCheckWithRetry(func() bool {
		return checker.checkHTTP(b.URL.String() + path)
	})
}
