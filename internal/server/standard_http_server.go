package server

import (
	"context"
	"go-load-balancer/internal/algorithms"
	"go-load-balancer/internal/backend"
	"go-load-balancer/internal/config"
	"go-load-balancer/internal/proxy"
	"go-load-balancer/internal/stats"
	"log"
	"net/http"
	"time"
)

// StandardHTTPServer 是标准HTTP服务器实现
type StandardHTTPServer struct {
	cfg            *config.LBConfig
	proxy          *proxy.ReverseProxy
	health         *backend.HealthChecker
	httpServer     *http.Server
	algorithm      algorithms.Algorithm
	backendPool    *backend.Pool
	statsCollector stats.StatsCollector
	reporter       stats.Reporter
}

// NewStandardHTTPServer 创建新的标准HTTP服务器
func NewStandardHTTPServer(cfg *config.LBConfig) Server {
	// 创建后端池
	backends := make([]*backend.Backend, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		b, err := backend.NewBackend(s.URL, s.Weight)
		if err != nil {
			log.Fatalf("创建后端失败: %v", err)
		}
		// 设置健康检查路径
		b.HealthCheckPath = s.HealthCheckPath
		backends = append(backends, b)
	}

	pool := backend.NewPool(backends)

	// 创建负载均衡算法
	alg, err := algorithms.CreateAlgorithm(cfg.Algorithm, backends)
	if err != nil {
		log.Fatalf("创建负载均衡算法失败: %v", err)
	}

	// 创建统计收集器
	collector := stats.NewDefaultCollector()

	// 创建状态报告器
	reporter := stats.NewDefaultReporter()
	reporter.UpdateBackends(backends)

	// 创建反向代理
	rp := proxy.NewReverseProxy(pool, alg, collector)

	// 创建健康检查
	timeout, _ := time.ParseDuration(cfg.HealthCheck.Timeout)
	checker := backend.NewHealthChecker(pool, timeout, cfg.HealthCheck.RetryCount)

	return &StandardHTTPServer{
		cfg:            cfg,
		proxy:          rp,
		health:         checker,
		backendPool:    pool,
		algorithm:      alg,
		statsCollector: collector,
		reporter:       reporter,
	}
}

// Start 启动HTTP服务器
func (s *StandardHTTPServer) Start() error {
	// 启动健康检查
	interval, _ := time.ParseDuration(s.cfg.HealthCheck.Interval)
	s.health.Start(interval)

	// 创建HTTP服务器
	mux := http.NewServeMux()

	// 设置主处理器为反向代理
	mux.Handle("/", s.proxy)

	// 添加监控端点
	mux.Handle("/metrics", stats.GetPrometheusHandler())

	// 添加状态报告端点
	mux.Handle("/status", s.reporter)

	// 添加健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建HTTP服务器
	s.httpServer = &http.Server{
		Addr:    s.cfg.ListenAddr,
		Handler: mux,
	}

	// 定期更新统计信息
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.statsCollector.UpdateBackendStatus(s.backendPool.GetAll())
				s.reporter.UpdateBackends(s.backendPool.GetAll())
			}
		}
	}()

	log.Printf("HTTP服务器已启动，监听地址: %s\n", s.cfg.ListenAddr)
	return s.httpServer.ListenAndServe()
}

// Stop 停止HTTP服务器
func (s *StandardHTTPServer) Stop() error {
	// 先停止健康检查
	s.health.Stop()

	// 优雅关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}
