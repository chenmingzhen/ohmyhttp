package demo

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/chenmingzhen/ohmyhttp"
	"github.com/chenmingzhen/ohmyhttp/errorx"
)

type ErrorExampleResponse struct {
	OK bool `json:"ok"`
}

func (*Handler) BusinessError(
	context.Context,
	*EmptyRequest,
) (*ErrorExampleResponse, error) {
	return nil, ohmyhttp.NewError(
		http.StatusConflict,
		"order_conflict",
		"order cannot be submitted twice",
	).WithDetails(map[string]string{
		"order_id": "order-1001",
	})
}

func (*Handler) StackError(
	context.Context,
	*EmptyRequest,
) (*ErrorExampleResponse, error) {
	databaseErr := errors.New("database connection refused")
	return nil, errorx.WithStack(fmt.Errorf("query order: %w", databaseErr))
}

func (*Handler) UnknownError(
	context.Context,
	*EmptyRequest,
) (*ErrorExampleResponse, error) {
	return nil, errors.New("unexpected dependency failure")
}

func (*Handler) Panic(
	context.Context,
	*EmptyRequest,
) (*ErrorExampleResponse, error) {
	panic("error example panic")
}

// AbortHandler 演示 net/http 约定的静默中止请求方式。框架的 Recovery
// 会继续抛出这个特殊值，最终由 net/http 关闭连接且不记录 panic 堆栈。
func (*Handler) AbortHandler(
	context.Context,
	*EmptyRequest,
) (*ErrorExampleResponse, error) {
	panic(http.ErrAbortHandler)
}
