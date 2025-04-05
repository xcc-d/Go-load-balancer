package main

import (
	"Go-load-balancer/internal/config"
	"Go-load-balancer/internal/server"
	"flag"
	"log"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	// 创建服务管理器
	mgr := server.NewServerManager()
	mgr.CreateFromConfig(cfg)

	// 启动服务
	mgr.StartAll()
}
