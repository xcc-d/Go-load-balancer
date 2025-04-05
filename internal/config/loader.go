package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

// LoadConfig 从指定路径加载配置
func LoadConfig(configPath string) (*LBConfig, error) {
	// 设置viper配置
	v := viper.New()
	v.SetConfigType("yaml")

	// 如果指定了配置文件路径
	if configPath != "" {
		// 解析配置文件路径
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			return nil, fmt.Errorf("解析配置文件路径失败: %v", err)
		}

		// 设置配置文件路径
		v.SetConfigFile(absPath)
	} else {
		// 默认从当前目录查找config.yaml
		v.SetConfigName("config")
		v.AddConfigPath(".")
	}

	// 读取环境变量
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 打印原始配置
	fmt.Println("Raw config values:")
	for _, key := range v.AllKeys() {
		fmt.Printf("%s: %v\n", key, v.Get(key))
	}

	// 解析配置到结构体
	var cfg LBConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %v", err)
	}

	// 调试日志
	fmt.Printf("Loaded config: %+v\n", cfg)

	return &cfg, nil
}
