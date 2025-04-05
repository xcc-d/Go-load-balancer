package server

import (
	"Go-load-balancer/internal/backend"
	"Go-load-balancer/internal/config"
	"Go-load-balancer/internal/proxy"
	"log"
	"net/http"
	"time"
)

// HTTPServer HTTP服务器实现
type HTTPServer struct {
	cfg        *config.LBConfig
	proxy      *proxy.ReverseProxy
	health     *backend.HealthChecker
	httpServer *http.Server
}

// NewHTTPServer 创建新的HTTP服务器
func NewHTTPServer(cfg *config.LBConfig) *HTTPServer {
	// 创建后端池
	backends := make([]*backend.Backend, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		b, err := backend.NewBackend(s.URL, s.Weight)
		if err != nil {
			log.Fatalf("创建后端失败: %v", err)
		}
		backends = append(backends, b)
	}

	pool := backend.NewPool(backends)
	rp := proxy.NewReverseProxy(pool)

	// 创建健康检查
	timeout, _ := time.ParseDuration(cfg.HealthCheck.Timeout)
	checker := backend.NewHealthChecker(pool, timeout)

	return &HTTPServer{
		cfg:    cfg,
		proxy:  rp,
		health: checker,
	}
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start() error {
	// 启动健康检查
	interval, _ := time.ParseDuration(s.cfg.HealthCheck.Interval)
	s.health.Start(interval)

	// 设置HTTP服务器
	s.httpServer = &http.Server{
		Addr:    s.cfg.ListenAddr,
		Handler: s.proxy,
	}

	log.Printf("启动HTTP服务器，监听地址: %s", s.cfg.ListenAddr)
	return s.httpServer.ListenAndServe()
}

// Stop 停止HTTP服务器
func (s *HTTPServer) Stop() error {
	s.health.Stop()
	return s.httpServer.Close()
}
