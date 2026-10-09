// Package demo 包含完整的 ohmyhttp 示例应用。
package demo

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/chenmingzhen/ohmyhttp"
	"github.com/chenmingzhen/ohmyhttp/meta"
	"github.com/chenmingzhen/ohmyhttp/openapi"
)

// New 构建完整应用；底层路由器由 App 管理。
func New() *ohmyhttp.App {
	logger := log.Default()
	accessLog := NewAccessLog(
		logger,
		"access_token",
		"refresh_token",
		"authorization",
		"token",
		"secret",
	)
	accessFilter := NewAccessFilterRule()
	accessFilter.Handle("/tenants/*/login", RedactRequestFields("password"))
	accessFilter.Handle("/oauth/token", RedactRequestFields("password"))
	accessFilter.Handle("/sessions", RedactRequestFields("password"))
	accessFilter.Handle(
		"/users",
		RedactRequestFields("password", "confirm_password"),
	)
	accessFilter.Exclude("/healthz")
	app := ohmyhttp.New(
		ohmyhttp.WithRecovery(),
		ohmyhttp.WithEnvironment(ohmyhttp.EnvDevelopment),
		ohmyhttp.WithRequestObserver(accessLog.ObserveRequest),
		ohmyhttp.WithErrorObserver(LogErrors(logger)),
		ohmyhttp.WithResponseObserver(accessLog.ObserveResponse),
		ohmyhttp.WithDescribedBy("/openapi.json"),
		ohmyhttp.WithDiscardUnknownFields(true),
		ohmyhttp.WithMultipartMemory(16<<20),
		ohmyhttp.WithValidationRule("username", ValidateUsername),
	)

	// 访问日志包裹 Recovery，因此 panic 转换为响应后也会记录最终状态。
	app.Use(
		RequestID(),
		accessLog.Middleware(accessFilter),
		CORS(CORSConfig{
			AllowOrigin:      "*",
			AllowHeaders:     "Authorization, Content-Type, X-Request-ID, X-User-ID",
			AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
			AllowCredentials: false,
			ExposeHeaders: "X-Request-ID, " + refreshAccessTokenHeader +
				", " + refreshAccessExpiresAtHeader,
			MaxAge: 12 * time.Hour,
		}),
		JWTAuthentication(JWTConfig{
			Secret: []byte(demoJWTSecret),
			ExcludeRoutes: []string{
				"/",
				"/api/*",
				"/tenants/*",
				"/oauth/*",
				"/sessions",
				"/users",
				"/users/*",
				"/reports/*",
				"/events",
				"/errors/*",
				"/ws/*",
				"/healthz",
				"/openapi.json",
				"/openapi.yaml",
				"/schemas",
				"/docs",
				"/assets/*",
				"/browse/*",
			},
			Authorize: func(_ context.Context, user User) error {
				if user.Name == "disabled" {
					return errors.New("account is disabled")
				}
				return nil
			},
			RefreshBefore: 15 * time.Minute,
			RefreshExpire: 2 * time.Hour,
		}),
	)
	// EndpointMiddleware 在请求绑定和校验后执行；这里只包裹强类型 endpoint。
	app.UseEndpointMiddleware(EndpointExecutionLog(logger))

	handler := &Handler{}
	app.Get(
		"/{$}",
		handler.Home,
		ohmyhttp.Describe(
			meta.Name("home"),
			meta.Summary("Demo 首页"),
			meta.Tags("system"),
			meta.ReturnsContent(http.StatusOK, "HTML 首页", "text/html; charset=utf-8", ""),
		),
	)
	api := app.Group("/api")
	v1 := api.Group("/v1")

	v1.Get(
		"/ping",
		handler.Ping,
		ohmyhttp.Describe(
			meta.Name("ping"),
			meta.Summary("健康检查"),
			meta.Tags("system"),
			meta.ReturnsContent(http.StatusOK, "健康检查响应", "text/plain; charset=utf-8", ""),
		),
	)
	errorExamples := app.Group("/errors")
	errorExamples.Get(
		"/business",
		handler.BusinessError,
		ohmyhttp.Describe(
			meta.Name("business_error"),
			meta.Summary("返回可公开的业务错误"),
			meta.Tags("errors"),
			meta.ReturnsError(http.StatusConflict, "订单状态冲突"),
		),
	)
	errorExamples.Get(
		"/stack",
		handler.StackError,
		ohmyhttp.Describe(
			meta.Name("stack_error"),
			meta.Summary("返回包含产生位置堆栈的内部错误"),
			meta.Tags("errors"),
			meta.ReturnsError(http.StatusInternalServerError, "系统错误"),
		),
	)
	errorExamples.Get(
		"/unknown",
		handler.UnknownError,
		ohmyhttp.Describe(
			meta.Name("unknown_error"),
			meta.Summary("返回没有堆栈的普通未知错误"),
			meta.Tags("errors"),
			meta.ReturnsError(http.StatusInternalServerError, "系统错误"),
		),
	)
	errorExamples.Get(
		"/panic",
		handler.Panic,
		ohmyhttp.Describe(
			meta.Name("panic_error"),
			meta.Summary("触发 Recovery 并记录 panic 堆栈"),
			meta.Tags("errors"),
			meta.ReturnsError(http.StatusInternalServerError, "系统错误"),
		),
	)
	errorExamples.Get(
		"/abort",
		handler.AbortHandler,
		ohmyhttp.Describe(
			meta.Name("abort_handler"),
			meta.Summary("使用 http.ErrAbortHandler 中止请求"),
			meta.Description("连接会被关闭，不生成 HTTP 响应，也不会记录普通 panic 堆栈。"),
			meta.Tags("errors"),
		),
	)
	v1.Get(
		"/hello.html",
		handler.HelloHTML,
		ohmyhttp.Describe(
			meta.Name("hello_html"),
			meta.Summary("HTML 响应示例"),
			meta.Tags("system"),
			meta.ReturnsContent(http.StatusOK, "HTML 页面", "text/html; charset=utf-8", ""),
		),
	)
	v1.Get(
		"/bytes",
		handler.RawBytes,
		ohmyhttp.Describe(
			meta.Name("raw_bytes"),
			meta.Summary("原始字节响应示例"),
			meta.Tags("system"),
			meta.ReturnsContent(http.StatusOK, "原始字节", "application/octet-stream", []byte{}),
		),
	)
	app.Post(
		"/tenants/{tenant_id}/login",
		handler.Login,
		ohmyhttp.Describe(
			meta.Name("login"),
			meta.Summary("租户登录"),
			meta.Tags("auth"),
			meta.Returns(http.StatusOK, "登录成功", (*LoginResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
			meta.ReturnsError(http.StatusUnauthorized, "账号或密码错误"),
		),
	)
	app.Post(
		"/oauth/token",
		handler.IssueOAuthToken,
		ohmyhttp.Describe(
			meta.Name("issue_oauth_token"),
			meta.Summary("签发 OAuth 风格 token"),
			meta.Description("演示 application/x-www-form-urlencoded 表单绑定和请求级跨字段校验。"),
			meta.Tags("auth"),
			meta.Returns(http.StatusOK, "签发成功", (*OAuthTokenResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
			meta.ReturnsError(http.StatusUnauthorized, "账号或刷新令牌错误"),
		),
	)
	app.Post(
		"/sessions",
		handler.CreateSession,
		ohmyhttp.MaxBodyBytes(512),
		ohmyhttp.Describe(
			meta.Name("create_session"),
			meta.Summary("创建登录会话"),
			meta.Description("演示显式响应头、Set-Cookie 和单路由请求体大小限制。"),
			meta.Tags("auth"),
			meta.Extension("x-demo", map[string]string{"feature": "explicit-response"}),
			meta.Returns(http.StatusCreated, "创建成功", (*CreateSessionResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
			meta.ReturnsError(http.StatusUnauthorized, "账号或密码错误"),
			meta.ReturnsError(http.StatusRequestEntityTooLarge, "请求体过大"),
		),
	)
	app.Patch(
		"/profile",
		handler.UpdateProfile,
		ohmyhttp.Describe(
			meta.Name("update_profile"),
			meta.Summary("更新当前用户资料"),
			meta.Description("请求体按 JSON 结构绑定到 Request.Profile，X-User-ID 从 header 绑定。"),
			meta.Tags("user"),
			meta.Security(ohmyhttp.SecurityRequirement{"bearerAuth": nil}),
			meta.Returns(http.StatusOK, "更新成功", (*UpdateProfileResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
			meta.ReturnsError(http.StatusUnauthorized, "未登录或登录状态失效"),
		),
	)
	app.Post(
		"/tenants/{tenant_id}/reports",
		handler.CreateReport,
		ohmyhttp.Status(http.StatusCreated),
		ohmyhttp.Describe(
			meta.Name("create_report"),
			meta.Summary("创建报表"),
			meta.Description("演示 path、header 和 JSON body 同时绑定。"),
			meta.Tags("report"),
			meta.Returns(http.StatusCreated, "创建成功", (*CreateReportResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
		),
	)
	app.Post(
		"/users",
		handler.CreateUser,
		ohmyhttp.Status(http.StatusCreated),
		ohmyhttp.Describe(
			meta.Name("create_user"),
			meta.Summary("创建用户"),
			meta.Tags("user"),
			meta.Value("audit", true),
			meta.Returns(http.StatusCreated, "创建成功", (*CreateUserResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
		),
	)
	app.Post(
		"/users/{user_id}/avatar",
		handler.UploadAvatar,
		ohmyhttp.Status(http.StatusCreated),
		ohmyhttp.MaxBodyBytes(32<<20),
		ohmyhttp.Describe(
			meta.Name("upload_avatar"),
			meta.Summary("上传用户头像"),
			meta.Description("演示 multipart/form-data 表单字段和文件字段绑定。"),
			meta.Tags("user"),
			meta.Returns(http.StatusCreated, "上传成功", (*UploadAvatarResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
			meta.ReturnsError(http.StatusRequestEntityTooLarge, "文件过大"),
		),
	)
	app.Get(
		"/reports/search",
		handler.SearchReports,
		ohmyhttp.Describe(
			meta.Name("search_reports"),
			meta.Summary("搜索报表"),
			meta.Description("演示可重复 query 参数绑定到 slice，以及 default、enum、范围校验如何进入 OpenAPI。"),
			meta.Tags("report"),
			meta.Returns(http.StatusOK, "搜索结果", (*SearchReportsResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
		),
	)
	app.Get(
		"/profile",
		handler.Profile,
		ohmyhttp.Describe(
			meta.Name("profile"),
			meta.Summary("查询当前用户资料"),
			meta.Tags("user"),
			meta.Security(ohmyhttp.SecurityRequirement{"bearerAuth": nil}),
			meta.Returns(http.StatusOK, "用户资料", (*ProfileResponse)(nil)),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
			meta.ReturnsError(http.StatusUnauthorized, "未登录或登录状态失效"),
		),
	)
	app.Get(
		"/admin",
		handler.Admin,
		ohmyhttp.Describe(
			meta.Name("admin"),
			meta.Summary("管理员接口"),
			meta.Tags("admin"),
			meta.Security(ohmyhttp.SecurityRequirement{"bearerAuth": nil}),
			meta.Returns(http.StatusOK, "管理员响应", (*AdminResponse)(nil)),
			meta.ReturnsError(http.StatusUnauthorized, "未登录或登录状态失效"),
			meta.ReturnsError(http.StatusForbidden, "权限不足"),
		),
	)
	app.Get(
		"/reports/{report_id}/metadata",
		handler.ReportMetadata,
		ohmyhttp.Describe(
			meta.Name("report_metadata"),
			meta.Summary("查询报表元数据"),
			meta.Description("演示直接返回 ohmyhttp.Response，并通过 WithHeader 写出 ETag 和 Cache-Control。"),
			meta.Tags("report"),
			meta.Returns(http.StatusOK, "报表元数据", (*ReportMetadataResponse)(nil)),
			meta.ReturnsError(http.StatusForbidden, "权限不足"),
			meta.ReturnsError(http.StatusNotFound, "报表不存在"),
		),
	)
	app.Get(
		"/reports/{report_id}/download",
		handler.DownloadReport,
		ohmyhttp.Describe(
			meta.Name("download_report"),
			meta.Summary("下载报表文件"),
			meta.Tags("report"),
			meta.ReturnsFile(http.StatusOK, "text/plain; charset=utf-8", "报表文件"),
			meta.ReturnsError(http.StatusForbidden, "权限不足"),
			meta.ReturnsError(http.StatusNotFound, "报表不存在"),
		),
	)
	app.Get(
		"/reports/{report_id}/redirect",
		handler.RedirectReport,
		ohmyhttp.Describe(
			meta.Name("redirect_report"),
			meta.Summary("跳转到报表下载地址"),
			meta.Tags("report"),
			meta.Returns(http.StatusFound, "重定向响应", nil),
			meta.ReturnsError(http.StatusForbidden, "权限不足"),
			meta.ReturnsError(http.StatusNotFound, "报表不存在"),
		),
	)
	app.Get(
		"/reports/{report_id}/export",
		handler.ExportReportCSV,
		ohmyhttp.Describe(
			meta.Name("export_report"),
			meta.Summary("流式导出 CSV"),
			meta.Tags("report"),
			meta.ReturnsStream(http.StatusOK, "text/csv; charset=utf-8", "CSV 流"),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
			meta.ReturnsError(http.StatusForbidden, "权限不足"),
			meta.ReturnsError(http.StatusNotFound, "报表不存在"),
		),
	)
	app.Get(
		"/events",
		handler.Events,
		ohmyhttp.Describe(
			meta.Name("events"),
			meta.Summary("SSE 事件流"),
			meta.Tags("system"),
			meta.ReturnsStream(http.StatusOK, "text/event-stream", "SSE 事件流"),
			meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
		),
	)
	app.HandleHTTP(
		http.MethodGet,
		"/ws/echo",
		http.HandlerFunc(handler.WebSocketEcho),
	)
	app.Delete(
		"/reports/{report_id}",
		handler.DeleteReport,
		ohmyhttp.Describe(
			meta.Name("delete_report"),
			meta.Summary("删除报表"),
			meta.Tags("report"),
			meta.Returns(http.StatusNoContent, "删除成功", nil),
			meta.ReturnsError(http.StatusForbidden, "权限不足"),
			meta.ReturnsError(http.StatusNotFound, "报表不存在"),
		),
	)

	options := OpenAPIOptions()
	app.HandleResponse(http.MethodGet, "/healthz", func(*http.Request) (ohmyhttp.Response, error) {
		return ohmyhttp.NoContent(), nil
	})
	app.HandleResponse(http.MethodGet, "/openapi.json", openapi.JSONHandler(app, options...))
	app.HandleResponse(http.MethodGet, "/openapi.yaml", openapi.YAMLHandler(app, options...))
	app.HandleResponse(http.MethodGet, "/schemas", openapi.SchemasHandler(app, options...))
	app.HandleResponse(http.MethodGet, "/docs", openapi.DocsHandler(
		"/openapi.json",
		openapi.WithDocsTitle(OpenAPITitle),
	))
	app.StaticDir(
		"/assets",
		"examples/demo/public",
		ohmyhttp.StaticDirectoryListing(false),
		ohmyhttp.StaticCacheControl("public, max-age=3600"),
	)
	// files 目录没有 index.html；访问 /browse/ 时会生成静态目录列表。
	app.StaticDir(
		"/browse",
		"examples/demo/files",
		ohmyhttp.StaticDirectoryListing(true),
	)

	return app
}
