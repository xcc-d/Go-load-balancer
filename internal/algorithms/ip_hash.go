package algorithms

import (
	"go-load-balancer/internal/backend"
	"hash/fnv"
	"net/http"
	"strings"
	"sync"
)

// IPHash 实现IP哈希负载均衡算法
type IPHash struct {
	backends []*backend.Backend
	mu       sync.RWMutex
	request  *http.Request // 当前请求对象
}

// NewIPHash 创建新的IP哈希算法实例
func NewIPHash(backends []*backend.Backend) Algorithm {
	return &IPHash{
		backends: backends,
	}
}

// GetNextBackend 根据IP哈希获取后端服务器
func (ih *IPHash) GetNextBackend() *backend.Backend {
	ih.mu.RLock()
	defer ih.mu.RUnlock()

	// 如果没有可用后端
	if len(ih.backends) == 0 || ih.request == nil {
		return nil
	}

	// 筛选出活跃的后端
	var activeBackends []*backend.Backend
	for _, b := range ih.backends {
		if b.IsAlive() {
			activeBackends = append(activeBackends, b)
		}
	}

	// 如果没有活跃的后端，返回nil
	if len(activeBackends) == 0 {
		return nil
	}

	// 获取客户端IP
	clientIP := ih.getClientIP()

	// 计算哈希值
	h := fnv.New32a()
	h.Write([]byte(clientIP))
	hash := h.Sum32()

	// 确定索引
	index := int(hash % uint32(len(activeBackends)))

	return activeBackends[index]
}

// 获取客户端IP地址
func (ih *IPHash) getClientIP() string {
	// 如果请求为空，则返回默认IP
	if ih.request == nil {
		return "127.0.0.1"
	}

	// 尝试从X-Forwarded-For头获取
	ipSlice := ih.request.Header.Get("X-Forwarded-For")
	if ipSlice != "" {
		// X-Forwarded-For可能包含多个IP，取第一个
		ips := strings.Split(ipSlice, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 尝试从X-Real-IP头获取
	ip := ih.request.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// 从RemoteAddr获取
	ip = ih.request.RemoteAddr
	// 移除端口部分
	if i := strings.LastIndex(ip, ":"); i != -1 {
		ip = ip[:i]
	}

	return ip
}

// SetRequest 设置当前请求
func (ih *IPHash) SetRequest(req *http.Request) {
	ih.mu.Lock()
	defer ih.mu.Unlock()
	ih.request = req
}

// Name 返回算法名称
func (ih *IPHash) Name() string {
	return "ip_hash"
}
