package server

import (
	"Go-load-balancer/internal/config"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ServerManager 管理多个服务器实例
type ServerManager struct {
	servers []*HTTPServer
	wg      sync.WaitGroup
}

// NewServerManager 创建新的服务管理器
func NewServerManager() *ServerManager {
	return &ServerManager{}
}

// AddServer 添加服务器到管理器
func (m *ServerManager) AddServer(server *HTTPServer) {
	m.servers = append(m.servers, server)
}

// StartAll 启动所有服务器
func (m *ServerManager) StartAll() {
	// 设置信号监听
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动所有服务器
	for _, s := range m.servers {
		m.wg.Add(1)
		go func(s *HTTPServer) {
			defer m.wg.Done()
			if err := s.Start(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("服务器启动失败: %v", err)
			}
		}(s)
	}

	// 等待终止信号
	<-stopCh
	log.Println("接收到终止信号，正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, s := range m.servers {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("服务器关闭失败: %v", err)
		}
	}

	m.wg.Wait()
	log.Println("所有服务器已关闭")
}

// CreateFromConfig 从配置创建服务器
func (m *ServerManager) CreateFromConfig(cfg *config.LBConfig) {
	server := NewHTTPServer(cfg)
	m.AddServer(server)
}
