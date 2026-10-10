# HTTP 服务器

`server` 包负责监听和退出。应用逻辑留在 `App` 上，`App` 作为 `http.Handler` 传进去。

```go
if err := httpserver.Run(app); err != nil {
	log.Fatal(err)
}
```

默认监听 `127.0.0.1:8080`。

## 超时

| 选项 | 默认 | 含义 |
| --- | --- | --- |
| `WithAddress` | `127.0.0.1:8080` | 监听地址 |
| `WithReadHeaderTimeout` | 10s | 读取请求头的最长时间。零表示不限制 |
| `WithReadTimeout` | 不限制 | 读取完整请求 |
| `WithWriteTimeout` | 不限制 | 写响应 |
| `WithIdleTimeout` | 60s | keep-alive 空闲连接 |
| `WithShutdownTimeout` | 10s | 收到退出信号后等待在途请求的最长时间 |

```go
httpserver.Run(app,
	httpserver.WithAddress("127.0.0.1:8081"),
	httpserver.WithReadHeaderTimeout(5*time.Second),
	httpserver.WithShutdownTimeout(15*time.Second),
)
```

## HTTPS

```go
err := httpserver.RunTLS(app, "cert.pem", "key.pem",
	httpserver.WithAddress(":443"),
)
```

证书在监听前加载。超时、Banner 和停机流程与 `Run` 相同。

## 优雅停机

`Run` 监听 `SIGINT` 和 `SIGTERM`。第一次信号到达后：

1. 停止继续接管信号。第二次 Ctrl+C 由进程默认行为处理，不再等待在途请求
2. `http.Server.Shutdown` 停止接收新连接，等待已接受的请求结束
3. 超过 `WithShutdownTimeout` 后 `Shutdown` 返回错误，服务器改为 `Close` 强制断开

`http.ErrServerClosed` 视为正常结束，`Run` 返回 nil。监听失败和真正的服务错误会包上 `ohmyhttp/server:` 前缀返回。

## 开发 Banner

`App` 处于 `local`、`development` 或 `test` 时，启动成功会向标准输出打一行彩色 Banner，包含环境名、HTTP 或 HTTPS，以及监听地址。`staging` 和 `production` 保持安静。环境的设置方式见 [运行环境](/guide/environment)。

Banner 通过可选接口 `Environment()` 识别应用。传入的 `http.Handler` 若没有这个方法，则不打印。

## 不用这个包

`App` 已经实现 `http.Handler`。需要自定义 `BaseContext`、连接状态回调或 HTTP/2 参数时，可以直接：

```go
httpServer := &http.Server{
	Addr:    "127.0.0.1:8080",
	Handler: app,
}
log.Fatal(httpServer.ListenAndServe())
```

这样就不会带上 `server` 包的 Banner 和信号处理，退出逻辑由你自己的服务器负责。
