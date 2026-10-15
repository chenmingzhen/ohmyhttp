package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chenmingzhen/ohmyhttp"
	"github.com/chenmingzhen/ohmyhttp/errorx"
)

type accessLogStateKey struct{}

type RequestInfo struct {
	Method string  `json:"method"`
	Path   string  `json:"path"`
	RID    string  `json:"rid"`
	Req    string  `json:"req"`
	Resp   string  `json:"resp"`
	Code   int     `json:"code"`
	Error  string  `json:"error"`
	Cost   float64 `json:"cost"`
}

type accessLogState struct {
	request  string
	response string
	err      error
}

type accessLogger struct {
	logger          *log.Logger
	sensitiveFields map[string]struct{}
}

func NewAccessLog(logger *log.Logger, sensitiveFields ...string) *accessLogger {
	return &accessLogger{
		logger:          logger,
		sensitiveFields: makeSensitiveFields(sensitiveFields),
	}
}

// Middleware 记录完整 HTTP 生命周期；请求和响应的逻辑值由两个 Observer
// 写入同一个请求级状态，最终只输出一条访问日志。
func (access *accessLogger) Middleware(
	filterRule *AccessFilterRule,
) ohmyhttp.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Method == http.MethodOptions {
				next.ServeHTTP(writer, request)
				return
			}

			startedAt := time.Now()
			state := &accessLogState{}
			ctx := context.WithValue(request.Context(), accessLogStateKey{}, state)
			request = request.WithContext(ctx)

			next.ServeHTTP(writer, request)

			response, _ := ohmyhttp.ResponseInfoFromWriter(writer)
			if state.request == "" {
				state.request = nilLogValue
			}
			if state.response == "" {
				state.response = nilLogValue
			}
			errorMessage := ""
			if state.err != nil {
				errorMessage = state.err.Error()
			}
			info := RequestInfo{
				Method: request.Method,
				Path:   access.requestPath(request),
				RID:    RequestIDFromContext(request.Context()),
				Req:    state.request,
				Resp:   state.response,
				Code:   response.StatusCode,
				Error:  errorMessage,
				Cost:   time.Since(startedAt).Seconds(),
			}
			if filterRule != nil {
				filterRule.Apply(request.URL.Path, &info)
				if filterRule.Excluded(request.URL.Path) {
					return
				}
			}
			access.log(info)
		})
	}
}

func (access *accessLogger) log(info RequestInfo) {
	access.logger.Printf(
		"level=%s msg=http_request method=%s path=%q rid=%s req=%q resp=%q code=%d error=%q cost=%f",
		accessLogLevel(info),
		info.Method,
		info.Path,
		info.RID,
		info.Req,
		info.Resp,
		info.Code,
		info.Error,
		info.Cost,
	)
}

func (access *accessLogger) ObserveRequest(
	ctx context.Context,
	observation ohmyhttp.RequestObservation,
) {
	state, ok := accessLogStateFromContext(ctx)
	if !ok {
		return
	}
	state.request = access.requestLogValue(observation.Input)
	state.err = observation.Err
}

func (access *accessLogger) ObserveResponse(
	ctx context.Context,
	_ ohmyhttp.RouteInfo,
	response ohmyhttp.Response,
	err error,
) {
	state, ok := accessLogStateFromContext(ctx)
	if !ok {
		return
	}
	state.response = access.responseLogValue(response)
	state.err = err
}

func accessLogStateFromContext(ctx context.Context) (*accessLogState, bool) {
	state, ok := ctx.Value(accessLogStateKey{}).(*accessLogState)
	return state, ok
}

func (access *accessLogger) requestLogValue(input any) string {
	return access.structuredLogValue(input)
}

func (access *accessLogger) structuredLogValue(input any) string {
	if input == nil {
		return nilLogValue
	}
	data, err := json.Marshal(input)
	if err != nil {
		return err.Error()
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err.Error()
	}
	redactLogValue(value, access.sensitiveFields)
	return logJSON(value)
}

func (access *accessLogger) requestPath(request *http.Request) string {
	query := request.URL.Query()
	redacted := false
	for key := range query {
		if _, sensitive := access.sensitiveFields[normalizeLogField(key)]; sensitive {
			query.Set(key, "***")
			redacted = true
		}
	}
	if !redacted {
		return request.URL.RequestURI()
	}
	requestURL := *request.URL
	requestURL.RawQuery = query.Encode()
	return requestURL.RequestURI()
}

func makeSensitiveFields(fields []string) map[string]struct{} {
	result := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field = normalizeLogField(field); field != "" {
			result[field] = struct{}{}
		}
	}
	return result
}

func redactLogValue(value any, sensitiveFields map[string]struct{}) {
	switch value := value.(type) {
	case map[string]any:
		for key, item := range value {
			if _, sensitive := sensitiveFields[normalizeLogField(key)]; sensitive {
				value[key] = "***"
				continue
			}
			redactLogValue(item, sensitiveFields)
		}
	case []any:
		for _, item := range value {
			redactLogValue(item, sensitiveFields)
		}
	}
}

func normalizeLogField(field string) string {
	field = strings.ToLower(strings.TrimSpace(field))
	return strings.NewReplacer("_", "", "-", "", ".", "").Replace(field)
}

const nilLogValue = "<nil>"

func accessLogLevel(info RequestInfo) string {
	switch {
	case info.Code >= 500:
		return "ERROR"
	case info.Code >= 400 || info.Error != "":
		return "WARN"
	default:
		return "INFO"
	}
}

func setAccessResponse(ctx context.Context, response any, err error) {
	state, ok := accessLogStateFromContext(ctx)
	if !ok {
		return
	}
	state.response = logJSON(response)
	state.err = err
}

func logJSON(value any) string {
	if value == nil {
		return nilLogValue
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// LogErrors 演示错误观察器：它只记录错误，不改变最终响应。
func LogErrors(logger *log.Logger) ohmyhttp.ErrorObserver {
	return func(ctx context.Context, err error, response ohmyhttp.Response) {
		if response.Status < 500 {
			return
		}

		meta, _ := ohmyhttp.MetaFromContext(ctx)
		message := fmt.Sprintf(
			"level=error rid=%s method=%s route=%s status=%d err=%v",
			RequestIDFromContext(ctx),
			meta.Route.Method,
			meta.Route.Pattern,
			response.Status,
			err,
		)
		if stack := errorx.StackTrace(err); stack != "" {
			message += "\n" + stack
		}
		logger.Print(message)
	}
}

func (access *accessLogger) responseLogValue(response ohmyhttp.Response) string {
	switch body := response.Body.(type) {
	case ohmyhttp.FileBody:
		if body.DownloadName() != "" {
			return "attachment " + body.DownloadName()
		}
		return "file " + body.Name()
	case ohmyhttp.RedirectBody:
		return "redirect to " + body.Location()
	case ohmyhttp.RawBody:
		switch {
		case strings.HasPrefix(body.ContentType(), "text/plain"):
			return string(body.Data())
		case strings.HasPrefix(body.ContentType(), "text/html"):
			return "html response"
		default:
			return fmt.Sprintf("%d bytes", len(body.Data()))
		}
	case ohmyhttp.StreamingBody:
		return "streaming response"
	default:
		return access.structuredLogValue(response.Body)
	}
}

type ReportNotFoundError struct {
	ReportID string
}

var (
	_ error              = (*ReportNotFoundError)(nil)
	_ ohmyhttp.HTTPError = (*ReportNotFoundError)(nil)
)

func (e *ReportNotFoundError) Error() string {
	return fmt.Sprintf("report %q was not found", e.ReportID)
}

func (e *ReportNotFoundError) HTTPStatus() int { return http.StatusNotFound }

func (e *ReportNotFoundError) ErrorCode() string { return "report_not_found" }

func (e *ReportNotFoundError) ErrorMessage() string { return "report does not exist" }

func (e *ReportNotFoundError) ErrorDetails() any {
	return map[string]string{"report_id": e.ReportID}
}

func newReportNotFound(reportID string) error {
	return &ReportNotFoundError{ReportID: reportID}
}
