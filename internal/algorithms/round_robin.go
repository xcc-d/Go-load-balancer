package algorithms

import (
	"go-load-balancer/internal/backend"
	"net/http"
	"sync/atomic"
)

// RoundRobin 实现轮询负载均衡算法
type RoundRobin struct {
	backends []*backend.Backend
	current  uint64
}

// NewRoundRobin 创建新的轮询算法实例
func NewRoundRobin(backends []*backend.Backend) Algorithm {
	return &RoundRobin{
		backends: backends,
		current:  0,
	}
}

// GetNextBackend 获取下一个后端服务器
func (r *RoundRobin) GetNextBackend() *backend.Backend {
	// 如果没有可用后端
	if len(r.backends) == 0 {
		return nil
	}

	// 获取当前索引并原子递增
	next := atomic.AddUint64(&r.current, 1)

	// 计算实际索引
	index := int(next % uint64(len(r.backends)))

	// 返回对应的后端
	return r.backends[index]
}

// Name 返回算法名称
func (r *RoundRobin) Name() string {
	return "round_robin"
}

// SetRequest 设置当前请求(轮询算法不需要请求信息)
func (r *RoundRobin) SetRequest(req *http.Request) {
	// 轮询算法不依赖请求信息，此处为接口实现的空方法
}
