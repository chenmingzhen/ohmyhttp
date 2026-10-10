# 运行环境

环境用来区分开发期诊断和生产期安静行为。当前会影响 `server` 包是否打印启动 Banner：`local`、`development`、`test` 会打印，`staging` 和 `production` 不打印。

默认环境是 `production`。

```go
app := ohmyhttp.New(
	ohmyhttp.WithEnvironment(ohmyhttp.EnvDevelopment),
)

app.Environment().IsDevelopment() // true
```

## 取值

| 常量 | 名称 |
| --- | --- |
| `EnvLocal` | `local` |
| `EnvDevelopment` | `development` |
| `EnvTest` | `test` |
| `EnvStaging` | `staging` |
| `EnvProduction` | `production` |

`local`、`development` 和 `test` 的 `IsDevelopment()` 为 true。

## 从环境变量读取

```go
app := ohmyhttp.New(ohmyhttp.WithEnvironmentFromEnv())
```

不传键名时，按顺序读取第一个非空值：`OHMYHTTP_ENV`、`APP_ENV`、`GO_ENV`、`ENV`。都为空则使用 `production`。环境变量的值使用上表中的名称，例如 `development`。

```go
ohmyhttp.WithEnvironmentFromEnv("SERVICE_ENV")
```

传入键名后只查这些键，不再使用默认列表。`server.Run` 在开发环境打印启动 Banner。
