package algorithms

import (
	"go-load-balancer/internal/backend"
	"net/http"
	"sync"
)

// WeightedRoundRobin 实现加权轮询负载均衡算法
type WeightedRoundRobin struct {
	backends       []*backend.Backend
	currentWeights []int
	mu             sync.Mutex
}

// NewWeightedRoundRobin 创建新的加权轮询算法实例
func NewWeightedRoundRobin(backends []*backend.Backend) Algorithm {
	weights := make([]int, len(backends))
	return &WeightedRoundRobin{
		backends:       backends,
		currentWeights: weights,
	}
}

// GetNextBackend 根据加权轮询算法获取下一个后端服务器
func (wrr *WeightedRoundRobin) GetNextBackend() *backend.Backend {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	// 如果没有可用后端
	if len(wrr.backends) == 0 {
		return nil
	}

	// 筛选出活跃的后端及其权重
	var activeBackends []*backend.Backend
	var activeWeights []int
	var totalWeight int

	for _, b := range wrr.backends {
		if b.IsAlive() {
			activeBackends = append(activeBackends, b)
			activeWeights = append(activeWeights, b.Weight)
			totalWeight += b.Weight
		}
	}

	// 如果没有活跃的后端，返回nil
	if len(activeBackends) == 0 {
		return nil
	}

	// 如果只有一个活跃的后端，直接返回
	if len(activeBackends) == 1 {
		return activeBackends[0]
	}

	// 用于记录权重最大的后端索引
	var maxIndex int
	var maxWeight int = 0

	// 实现加权轮询
	for i := range activeBackends {
		// 更新当前权重
		if i < len(wrr.currentWeights) {
			wrr.currentWeights[i] += activeWeights[i]
		}

		// 查找最大权重
		if wrr.currentWeights[i] > maxWeight {
			maxWeight = wrr.currentWeights[i]
			maxIndex = i
		}
	}

	// 选中后端后，减去总权重
	wrr.currentWeights[maxIndex] -= totalWeight

	return activeBackends[maxIndex]
}

// Name 返回算法名称
func (wrr *WeightedRoundRobin) Name() string {
	return "weighted_rr"
}

// SetRequest 设置当前请求(加权轮询算法不需要请求信息)
func (wrr *WeightedRoundRobin) SetRequest(req *http.Request) {
	// 加权轮询算法不依赖请求信息，此处为接口实现的空方法
}
