package config

// ServerConfig 定义后端服务器配置
type ServerConfig struct {
	URL    string `yaml:"url" json:"url"`
	Weight int    `yaml:"weight" json:"weight"`
	Health string `yaml:"health" json:"health"`
}

// LBConfig 负载均衡器配置
type LBConfig struct {
	ListenAddr  string         `yaml:"listen_addr" mapstructure:"listen_addr"`
	Algorithm   string         `yaml:"algorithm" mapstructure:"algorithm"`
	Servers     []ServerConfig `yaml:"servers" mapstructure:"servers"`
	HealthCheck struct {
		Interval string `yaml:"interval" mapstructure:"interval"`
		Timeout  string `yaml:"timeout" mapstructure:"timeout"`
		Path     string `yaml:"path" mapstructure:"path"`
	} `yaml:"health_check" mapstructure:"health_check"`
}
