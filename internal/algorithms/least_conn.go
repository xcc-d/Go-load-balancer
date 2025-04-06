package algorithms

import (
	"go-load-balancer/internal/backend"
	"net/http"
	"sync"
)

// LeastConn 实现最少连接负载均衡算法
type LeastConn struct {
	backends []*backend.Backend
	mu       sync.Mutex
}

// NewLeastConn 创建新的最少连接算法实例
func NewLeastConn(backends []*backend.Backend) Algorithm {
	return &LeastConn{
		backends: backends,
	}
}

// GetNextBackend 获取活动连接数最少的后端服务器
func (lc *LeastConn) GetNextBackend() *backend.Backend {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// 如果没有可用后端
	if len(lc.backends) == 0 {
		return nil
	}

	// 筛选出活跃的后端
	var activeBackends []*backend.Backend
	for _, b := range lc.backends {
		if b.IsAlive() {
			activeBackends = append(activeBackends, b)
		}
	}

	// 如果没有活跃的后端，返回nil
	if len(activeBackends) == 0 {
		return nil
	}

	// 初始选择第一个后端
	minConnBackend := activeBackends[0]
	minConn := minConnBackend.GetConnections()

	// 查找具有最少活动连接数的后端
	for _, b := range activeBackends[1:] {
		currConn := b.GetConnections()
		if currConn < minConn {
			minConn = currConn
			minConnBackend = b
		}
	}

	return minConnBackend
}

// Name 返回算法名称
func (lc *LeastConn) Name() string {
	return "least_conn"
}

// SetRequest 设置当前请求(最少连接算法不需要请求信息)
func (lc *LeastConn) SetRequest(req *http.Request) {
	// 最少连接算法不依赖请求信息，此处为接口实现的空方法
}
