package stats

import (
	"go-load-balancer/internal/backend"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// 单例模式类
	promOnce sync.Once

	// 单例实例
	promInstance *PrometheusCollector
)

// MetricNamespace 定义指标命名空间
const MetricNamespace = "go_lb"

// PrometheusCollector 用于收集和导出Prometheus指标
type PrometheusCollector struct {
	// 请求计数器
	requestCounter *prometheus.CounterVec

	// 响应时间直方图
	responseTime *prometheus.HistogramVec

	// 当前连接计数器
	activeConnections *prometheus.GaugeVec

	// 后端状态计数器
	backendStatus *prometheus.GaugeVec

	// 请求失败计数器
	requestErrors *prometheus.CounterVec
}

// GetPrometheusCollector 获取PrometheusCollector单例
func GetPrometheusCollector() *PrometheusCollector {
	promOnce.Do(func() {
		promInstance = newPrometheusCollector()
	})
	return promInstance
}

// newPrometheusCollector 创建新的Prometheus收集器
func newPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{
		// 请求计数器
		requestCounter: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricNamespace,
				Name:      "request_total",
				Help:      "请求总数",
			},
			[]string{"backend", "status_code", "method"},
		),

		// 响应时间直方图
		responseTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: MetricNamespace,
				Name:      "response_time_seconds",
				Help:      "请求响应时间",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"backend", "status_code", "method"},
		),

		// 当前连接计数器
		activeConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: MetricNamespace,
				Name:      "active_connections",
				Help:      "当前活动连接数",
			},
			[]string{"backend"},
		),

		// 后端状态计数器
		backendStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: MetricNamespace,
				Name:      "backend_status",
				Help:      "后端状态(1=活跃, 0=故障)",
			},
			[]string{"backend", "url"},
		),

		// 请求失败计数器
		requestErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricNamespace,
				Name:      "request_errors_total",
				Help:      "请求错误总数",
			},
			[]string{"backend", "error_type"},
		),
	}
}

// RecordRequest 记录请求信息
func (pc *PrometheusCollector) RecordRequest(backend string, statusCode int, method string, duration time.Duration) {
	pc.requestCounter.WithLabelValues(backend, http.StatusText(statusCode), method).Inc()
	pc.responseTime.WithLabelValues(backend, http.StatusText(statusCode), method).Observe(duration.Seconds())
}

// RecordError 记录错误信息
func (pc *PrometheusCollector) RecordError(backend string, errorType string) {
	pc.requestErrors.WithLabelValues(backend, errorType).Inc()
}

// UpdateBackendStatus 更新后端状态
func (pc *PrometheusCollector) UpdateBackendStatus(backends []*backend.Backend) {
	for _, b := range backends {
		isAlive := 0.0
		if b.IsAlive() {
			isAlive = 1.0
		}
		pc.backendStatus.WithLabelValues(b.Addr(), b.URL.String()).Set(isAlive)
		pc.activeConnections.WithLabelValues(b.Addr()).Set(float64(b.GetConnections()))
	}
}

// GetPrometheusHandler 获取Prometheus HTTP处理器
func GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}
