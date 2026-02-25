package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version   string          `yaml:"version"`
	Listen    string          `yaml:"listen"`
	Routes    []Route         `yaml:"routes"`
	Transport TransportConfig `yaml:"transport"`
	CORS      CORSConfig      `yaml:"cors"`
	Proxy     ProxyConfig     `yaml:"proxy"`
}

type Route struct {
	Path   string `yaml:"path"`
	Target string `yaml:"target"`
}

type TransportConfig struct {
	DialTimeout           int `yaml:"dial_timeout"`
	DialKeepAlive         int `yaml:"dial_keep_alive"`
	MaxIdleConns          int `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost   int `yaml:"max_idle_conns_per_host"`
	MaxConnsPerHost       int `yaml:"max_conns_per_host"`
	IdleConnTimeout       int `yaml:"idle_conn_timeout"`
	ResponseHeaderTimeout int `yaml:"response_header_timeout"`
	ReadBufferSize        int `yaml:"read_buffer_size"`
	WriteBufferSize       int `yaml:"write_buffer_size"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type ProxyConfig struct {
	MaxConcurrentRequests int `yaml:"max_concurrent_requests"`
}

// LoadConfig loads the configuration from a YAML file using gopkg.in/yaml.v3.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	cfg := &Config{
		Version: "1.0.0",
		Listen:  ":80",
		Transport: TransportConfig{
			DialTimeout:           10,
			DialKeepAlive:         30,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   2,
			MaxConnsPerHost:       20,
			IdleConnTimeout:       30,
			ResponseHeaderTimeout: 30,
			ReadBufferSize:        4096,
			WriteBufferSize:       4096,
		},
		Proxy: ProxyConfig{
			MaxConcurrentRequests: 100,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", path, err)
	}

	if len(cfg.Routes) == 0 {
		return nil, fmt.Errorf("no routes defined in %s", path)
	}

	for _, r := range cfg.Routes {
		if r.Path == "/health" {
			return nil, fmt.Errorf("route path cannot be '/health' as it is reserved for the health check")
		}
	}

	return cfg, nil
}
