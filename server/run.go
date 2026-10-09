// Package server 启动 http.Handler，并在进程收到退出信号时执行优雅停机。
package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// Run 监听 HTTP 端口并阻塞运行。App 处于 local、development 或 test 环境时会
// 输出启动 Banner。收到 SIGINT 或 SIGTERM 后，Run 会停止接收新连接，等待正在
// 处理的请求完成，并在超过配置的停机时间后强制关闭连接。
func Run(handler http.Handler, options ...Option) error {
	return run(handler, "HTTP", nil, options...)
}

// RunTLS 使用证书和私钥监听 HTTPS 端口并阻塞运行。它与 Run 使用相同的超时、
// Banner 和优雅停机配置。
func RunTLS(
	handler http.Handler,
	certFile string,
	keyFile string,
	options ...Option,
) error {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("ohmyhttp/server: load TLS certificate: %w", err)
	}
	return run(handler, "HTTPS", &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}, options...)
}

func run(
	handler http.Handler,
	protocol string,
	tlsConfig *tls.Config,
	options ...Option,
) error {
	config := buildConfig(options)

	httpServer := &http.Server{
		Addr:              config.address,
		Handler:           handler,
		ReadHeaderTimeout: config.readHeaderTimeout,
		ReadTimeout:       config.readTimeout,
		WriteTimeout:      config.writeTimeout,
		IdleTimeout:       config.idleTimeout,
		TLSConfig:         tlsConfig,
	}
	listener, err := net.Listen("tcp", config.address)
	if err != nil {
		return fmt.Errorf("ohmyhttp/server: %w", err)
	}
	defer listener.Close()

	ctx, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	printDevelopmentBanner(handler, protocol, config.address)

	serveResult := make(chan error, 1)
	go func() {
		// serveResult<-httpServer.ListenAndServe() 上面已经Listen，直接使用下面的Serve

		if tlsConfig == nil {
			serveResult <- httpServer.Serve(listener)
			return
		}
		serveResult <- httpServer.ServeTLS(listener, "", "")
	}()

	var serveErr error
	select {
	case serveErr = <-serveResult:
	case <-ctx.Done():
		// 第一次 Ctrl+C 已收到。
		// 从现在开始，第二次 Ctrl+C 将直接强制退出，Go 不再接管后续的 Ctrl+C
		stopSignals()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			config.shutdownTimeout,
		)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			_ = httpServer.Close()
			return fmt.Errorf("ohmyhttp/server: shutdown: %w", err)
		}

		serveErr = <-serveResult
	}

	if serveErr == nil || errors.Is(serveErr, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("ohmyhttp/server: serve: %w", serveErr)
}
