package demo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/chenmingzhen/ohmyhttp"
)

type User struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

type CORSConfig struct {
	AllowOrigin      string
	AllowHeaders     string
	AllowMethods     string
	AllowCredentials bool
	ExposeHeaders    string
	MaxAge           time.Duration
}

func CORS(config CORSConfig) ohmyhttp.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Access-Control-Allow-Origin", config.AllowOrigin)
			writer.Header().Set("Access-Control-Allow-Headers", config.AllowHeaders)
			writer.Header().Set("Access-Control-Allow-Methods", config.AllowMethods)
			writer.Header().Set(
				"Access-Control-Allow-Credentials",
				strconv.FormatBool(config.AllowCredentials),
			)
			writer.Header().Set("Access-Control-Expose-Headers", config.ExposeHeaders)
			writer.Header().Set(
				"Access-Control-Max-Age",
				strconv.FormatInt(int64(config.MaxAge/time.Second), 10),
			)

			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func RequestID() ohmyhttp.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestID := request.Header.Get("X-Request-ID")
			if requestID == "" {
				var value [12]byte
				if _, err := rand.Read(value[:]); err == nil {
					requestID = hex.EncodeToString(value[:])
				}
			}
			writer.Header().Set("X-Request-ID", requestID)

			ctx := context.WithValue(request.Context(), requestIDKey{}, requestID)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) User {
	user, _ := ctx.Value(userKey{}).(User)
	return user
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// EndpointExecutionLog 演示应用级端点中间件的用法。
//
// 它只会包裹强类型 endpoint，并且执行时请求已经完成绑定和校验，因此可以
// 通过 InputAs 读取绑定后的请求值。ResponseFunc、静态文件和普通 HTTP 路由
// 不会经过这里。
func EndpointExecutionLog(logger *log.Logger) ohmyhttp.EndpointMiddleware {
	return func(ctx *ohmyhttp.EndpointContext, next ohmyhttp.EndpointNext) (ohmyhttp.Response, error) {
		startedAt := time.Now()
		input, _ := ohmyhttp.InputAs[any](ctx)
		logger.Printf(
			"endpoint start method=%s path=%s request_id=%s input=%T",
			ctx.Request.Method,
			ctx.Route.Pattern,
			RequestIDFromContext(ctx.Context()),
			input,
		)

		response, err := next(ctx)
		logger.Printf(
			"endpoint finish method=%s path=%s status=%d cost=%s error=%v",
			ctx.Request.Method,
			ctx.Route.Pattern,
			response.Status,
			time.Since(startedAt),
			err,
		)
		return response, err
	}
}

type userKey struct{}
type requestIDKey struct{}
