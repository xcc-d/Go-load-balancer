package config

// ServerConfig 定义后端服务器配置
type ServerConfig struct {
	URL             string `yaml:"url" json:"url" mapstructure:"url"`
	Weight          int    `yaml:"weight" json:"weight" mapstructure:"weight"`
	Health          string `yaml:"health" json:"health" mapstructure:"health"`
	HealthCheckPath string `yaml:"health_check_path" json:"health_check_path" mapstructure:"health_check_path"`
}

// LBConfig 负载均衡器配置
type LBConfig struct {
	ListenAddr  string         `yaml:"listen_addr" mapstructure:"listen_addr"`
	Algorithm   string         `yaml:"algorithm" mapstructure:"algorithm"`
	Servers     []ServerConfig `yaml:"servers" mapstructure:"servers"`
	HealthCheck struct {
		Interval      string `yaml:"interval" mapstructure:"interval"`
		Timeout       string `yaml:"timeout" mapstructure:"timeout"`
		Path          string `yaml:"path" mapstructure:"path"`
		RetryCount    int    `yaml:"retry_count" mapstructure:"retry_count"`
		RetryInterval string `yaml:"retry_interval" mapstructure:"retry_interval"`
		MaxFailures   int    `yaml:"max_failures" mapstructure:"max_failures"`
	} `yaml:"health_check" mapstructure:"health_check"`
}
