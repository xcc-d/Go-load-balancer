package algorithms

import (
	"fmt"
	"go-load-balancer/internal/backend"
	"net/http"
	"strings"
)

// Algorithm 定义负载均衡算法接口
type Algorithm interface {
	// GetNextBackend 根据负载均衡算法获取下一个后端服务
	GetNextBackend() *backend.Backend

	// Name 返回算法名称
	Name() string

	// SetRequest 设置当前请求(对于基于请求的算法，如IP哈希)
	SetRequest(req *http.Request)
}

// CreateAlgorithm 根据算法名称和后端服务列表创建对应的负载均衡算法
func CreateAlgorithm(name string, backends []*backend.Backend) (Algorithm, error) {
	// 转换为小写以支持大小写不敏感的配置
	algorithmName := strings.ToLower(name)

	switch algorithmName {
	case "round_robin":
		return NewRoundRobin(backends), nil
	case "least_conn":
		return NewLeastConn(backends), nil
	case "weighted_rr":
		return NewWeightedRoundRobin(backends), nil
	case "ip_hash":
		return NewIPHash(backends), nil
	default:
		return nil, fmt.Errorf("不支持的负载均衡算法: %s", name)
	}
}
