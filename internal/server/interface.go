package server

import "go-load-balancer/internal/config"

// Server 定义服务器接口
type Server interface {
	Start() error
	Stop() error
}

// NewServerFunc 创建服务器的函数类型
type NewServerFunc func(*config.LBConfig) Server
