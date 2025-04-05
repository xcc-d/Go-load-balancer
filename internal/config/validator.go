package config

import (
	"fmt"
	"net/url"
	"strings"
)

// Validate 验证配置是否有效
func (c *LBConfig) Validate() error {
	// 验证监听地址
	if c.ListenAddr == "" {
		return fmt.Errorf("监听地址不能为空")
	}

	// 验证算法类型
	supportedAlgorithms := map[string]bool{
		"round_robin": true,
		"least_conn":  true,
		"weighted_rr": true,
		"ip_hash":     true,
	}
	if !supportedAlgorithms[strings.ToLower(c.Algorithm)] {
		return fmt.Errorf("不支持的负载均衡算法: %s", c.Algorithm)
	}

	// 验证后端服务器
	if len(c.Servers) == 0 {
		return fmt.Errorf("至少需要一个后端服务器")
	}

	for _, server := range c.Servers {
		if _, err := url.ParseRequestURI(server.URL); err != nil {
			return fmt.Errorf("无效的后端服务器URL: %s", server.URL)
		}
	}

	return nil
}
