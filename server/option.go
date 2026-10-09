package server

import "time"

// Option 配置 Run 创建的 HTTP Server。
type Option func(*config)

// WithAddress 设置监听地址。
func WithAddress(address string) Option {
	return func(cfg *config) {
		cfg.address = address
	}
}

// WithReadHeaderTimeout 设置读取请求头的超时时间；零表示不限制。
func WithReadHeaderTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.readHeaderTimeout = timeout
	}
}

// WithReadTimeout 设置读取完整请求的超时时间；零表示不限制。
func WithReadTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.readTimeout = timeout
	}
}

// WithWriteTimeout 设置写响应的超时时间；零表示不限制。
func WithWriteTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.writeTimeout = timeout
	}
}

// WithIdleTimeout 设置 keep-alive 空闲连接的超时时间；零表示不限制。
func WithIdleTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.idleTimeout = timeout
	}
}

// WithShutdownTimeout 设置收到退出信号后等待请求完成的最长时间。
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.shutdownTimeout = timeout
	}
}

const (
	defaultAddress           = "127.0.0.1:8080"
	defaultReadHeaderTimeout = 10 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

type config struct {
	address           string
	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	shutdownTimeout   time.Duration
}

func defaultConfig() config {
	return config{
		address:           defaultAddress,
		readHeaderTimeout: defaultReadHeaderTimeout,
		idleTimeout:       defaultIdleTimeout,
		shutdownTimeout:   defaultShutdownTimeout,
	}
}

func buildConfig(options []Option) config {
	config := defaultConfig()
	for _, option := range options {
		option(&config)
	}
	return config
}
