package algorithms

import (
	"Go-load-balancer/internal/backend"
	"sync/atomic"
)

type RoundRobin struct {
	backends []*backend.Backend
	current  uint64
}

func NewRoundRobin(backends []*backend.Backend) *RoundRobin {
	return &RoundRobin{
		backends: backends,
		current:  0,
	}
}

func (r *RoundRobin) GetNextBackend() *backend.Backend {
	// 获取当前索引并原子递增
	next := atomic.AddUint64(&r.current, 1)

	// 计算实际索引
	index := int(next % uint64(len(r.backends)))

	// 返回对应的后端
	return r.backends[index]
}

func (r *RoundRobin) Name() string {
	return "round_robin"
}
