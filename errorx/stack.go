package errorx

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// StackTracer 由可以提供格式化调用堆栈的错误实现。
type StackTracer interface {
	StackTrace() string
}

// StackError 保存原始错误以及调用 WithStack 时的调用堆栈。
type StackError struct {
	cause error
	stack []uintptr
}

var (
	_ error       = (*StackError)(nil)
	_ StackTracer = (*StackError)(nil)
)

func (e *StackError) Error() string { return e.cause.Error() }

func (e *StackError) Unwrap() error { return e.cause }

func (e *StackError) StackTrace() string { return formatStack(e.stack) }

// WithStack 为 err 保存当前调用堆栈。nil 或已经包含堆栈的错误会原样返回。
func WithStack(err error) error {
	if err == nil || HasStack(err) {
		return err
	}

	pcs := make([]uintptr, 64)
	n := runtime.Callers(2, pcs)
	return &StackError{
		cause: err,
		stack: pcs[:n],
	}
}

// HasStack 判断 err 的错误链中是否包含调用堆栈。
func HasStack(err error) bool {
	var target StackTracer
	return errors.As(err, &target)
}

// StackTrace 返回 err 错误链中的第一个调用堆栈；没有堆栈时返回空字符串。
func StackTrace(err error) string {
	var target StackTracer
	if !errors.As(err, &target) {
		return ""
	}
	return target.StackTrace()
}

func formatStack(pcs []uintptr) string {
	frames := runtime.CallersFrames(pcs)
	var builder strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(
			&builder,
			"%s\n\t%s:%d\n",
			frame.Function,
			frame.File,
			frame.Line,
		)
		if !more {
			return builder.String()
		}
	}
}
