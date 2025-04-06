package stats

import (
	"encoding/json"
	"go-load-balancer/internal/backend"
	"net/http"
	"sync"
	"time"
)

// StatusReport 代表负载均衡器的状态报告
type StatusReport struct {
	Timestamp      time.Time                 `json:"timestamp"`
	TotalRequests  int64                     `json:"total_requests"`
	ActiveRequests int                       `json:"active_requests"`
	BackendStatus  map[string]BackendMetrics `json:"backend_status"`
	Uptime         time.Duration             `json:"uptime"`
	StartTime      time.Time                 `json:"start_time"`
}

// BackendMetrics 包含后端服务器的指标
type BackendMetrics struct {
	URL               string        `json:"url"`
	Status            string        `json:"status"`
	ActiveConnections int64         `json:"active_connections"`
	TotalRequests     int64         `json:"total_requests"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	LastChecked       time.Time     `json:"last_checked"`
}

// Reporter 接口定义了报告生成功能
type Reporter interface {
	// GenerateReport 生成当前状态报告
	GenerateReport() *StatusReport

	// ServeHTTP 实现HTTP处理器接口，提供状态API
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// UpdateBackends 更新后端列表
	UpdateBackends(backends []*backend.Backend)

	// IncrementRequests 增加请求计数
	IncrementRequests()

	// DecrementRequests 减少请求计数
	DecrementRequests()
}

// DefaultReporter 默认报告生成器实现
type DefaultReporter struct {
	mu             sync.RWMutex
	backends       []*backend.Backend
	startTime      time.Time
	totalRequests  int64
	activeRequests int
	backendMetrics map[string]*BackendMetrics
}

// NewDefaultReporter 创建新的默认报告生成器
func NewDefaultReporter() *DefaultReporter {
	return &DefaultReporter{
		startTime:      time.Now(),
		backendMetrics: make(map[string]*BackendMetrics),
	}
}

// GenerateReport 生成当前状态报告
func (r *DefaultReporter) GenerateReport() *StatusReport {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 创建当前后端状态的副本
	backendStatus := make(map[string]BackendMetrics)
	for k, v := range r.backendMetrics {
		backendStatus[k] = *v
	}

	return &StatusReport{
		Timestamp:      time.Now(),
		TotalRequests:  r.totalRequests,
		ActiveRequests: r.activeRequests,
		BackendStatus:  backendStatus,
		Uptime:         time.Since(r.startTime),
		StartTime:      r.startTime,
	}
}

// ServeHTTP 处理HTTP请求，返回JSON格式的状态报告
func (r *DefaultReporter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	report := r.GenerateReport()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// 格式化输出
	if req.URL.Query().Get("pretty") == "true" {
		json.NewEncoder(w).Encode(report)
	} else {
		// 紧凑输出
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		encoder.Encode(report)
	}
}

// UpdateBackends 更新后端列表
func (r *DefaultReporter) UpdateBackends(backends []*backend.Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.backends = backends

	// 更新后端指标
	for _, b := range backends {
		addr := b.Addr()
		if _, exists := r.backendMetrics[addr]; !exists {
			r.backendMetrics[addr] = &BackendMetrics{
				URL:    b.URL.String(),
				Status: "unknown",
			}
		}

		// 更新状态
		if b.IsAlive() {
			r.backendMetrics[addr].Status = "healthy"
		} else {
			r.backendMetrics[addr].Status = "failed"
		}

		r.backendMetrics[addr].ActiveConnections = b.GetConnections()
		r.backendMetrics[addr].LastChecked = time.Now()
	}
}

// IncrementRequests 增加请求计数
func (r *DefaultReporter) IncrementRequests() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.totalRequests++
	r.activeRequests++
}

// DecrementRequests 减少请求计数
func (r *DefaultReporter) DecrementRequests() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activeRequests > 0 {
		r.activeRequests--
	}
}
