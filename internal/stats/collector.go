package stats

import (
	"go-load-balancer/internal/backend"
	"net/http"
	"time"
)

// StatsCollector 定义统计数据收集接口
type StatsCollector interface {
	// RecordRequest 记录请求信息
	RecordRequest(backend string, statusCode int, method string, duration time.Duration)

	// RecordError 记录错误信息
	RecordError(backend string, errorType string)

	// UpdateBackendStatus 更新后端状态
	UpdateBackendStatus(backends []*backend.Backend)
}

// DefaultCollector 默认统计收集器
type DefaultCollector struct {
	// 内部实现的集合
	collectors []StatsCollector
}

// NewDefaultCollector 创建新的默认收集器
func NewDefaultCollector() *DefaultCollector {
	// 创建普罗米修斯收集器
	promCollector := GetPrometheusCollector()

	return &DefaultCollector{
		collectors: []StatsCollector{
			promCollector,
		},
	}
}

// RecordRequest 记录请求信息
func (dc *DefaultCollector) RecordRequest(backend string, statusCode int, method string, duration time.Duration) {
	for _, collector := range dc.collectors {
		collector.RecordRequest(backend, statusCode, method, duration)
	}
}

// RecordError 记录错误信息
func (dc *DefaultCollector) RecordError(backend string, errorType string) {
	for _, collector := range dc.collectors {
		collector.RecordError(backend, errorType)
	}
}

// UpdateBackendStatus 更新后端状态
func (dc *DefaultCollector) UpdateBackendStatus(backends []*backend.Backend) {
	for _, collector := range dc.collectors {
		collector.UpdateBackendStatus(backends)
	}
}

// StatsMiddleware 创建统计中间件
func StatsMiddleware(collector StatsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			// 创建自定义的响应写入器以捕获状态码
			rw := NewResponseWriter(w)

			// 调用下一个处理器
			next.ServeHTTP(rw, r)

			// 记录请求
			duration := time.Since(startTime)
			statusCode := rw.StatusCode()
			backend := "unknown" // 稍后由反向代理填充

			// 获取请求方法
			method := r.Method

			// 记录请求统计
			collector.RecordRequest(backend, statusCode, method, duration)
		})
	}
}

// ResponseWriter 是一个http.ResponseWriter包装器，用于捕获状态码
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewResponseWriter 创建一个新的响应写入器
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // 默认200
	}
}

// WriteHeader 捕获状态码
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// StatusCode 返回状态码
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}
