package errorx

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// StackError 保存原始错误，以及调用 WithStack 时的调用堆栈。
// 堆栈只给日志使用。Error 返回内层错误的文本，Unwrap 返回内层错误。
type StackError struct {
	cause error
	stack []uintptr
}

var (
	_ error = (*StackError)(nil)
)

// Error 返回内层错误的文本，不包含堆栈。
func (e *StackError) Error() string { return e.cause.Error() }

// Unwrap 返回内层错误，供 errors.Is 和 errors.As 继续匹配。
func (e *StackError) Unwrap() error { return e.cause }

// StackTrace 返回 WithStack 调用处格式化后的调用堆栈。
func (e *StackError) StackTrace() string { return formatStack(e.stack) }

// WithStack 为 err 保存当前调用堆栈。堆栈从调用 WithStack 的位置开始。
// err 为 nil，或错误链上已经有 StackError 时，原样返回，避免重复包装。
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	var stackErr *StackError
	if errors.As(err, &stackErr) {
		return err
	}

	pcs := make([]uintptr, 64)
	n := runtime.Callers(2, pcs)
	return &StackError{
		cause: err,
		stack: pcs[:n],
	}
}

// StackTrace 返回 err 错误链中的第一段调用堆栈。
// 没有 StackError 的错误返回空字符串。
func StackTrace(err error) string {
	var stackErr *StackError
	if !errors.As(err, &stackErr) {
		return ""
	}
	return stackErr.StackTrace()
}

// formatStack 把程序计数器格式化成「函数名、文件、行号」。
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
