package tests

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// 错误类型常量
const (
	ErrorTypeBusiness = "business" // 业务错误
	ErrorTypeSystem   = "system"   // 系统错误
	ErrorTypeExternal = "external" // 外部依赖错误
)

// Error 自定义错误结构体
type Error struct {
	Code    int         // 错误码 (e.g., 404, 500)
	Type    string      // 错误类型 (business/system/external)
	Message string      // 用户可读的错误信息
	Detail  interface{} // 错误详情 (可以是结构体、字符串等)
	Stack   []string    // 调用堆栈 (可选)
	Time    time.Time   // 错误发生时间
	Err     error       // 原始错误 (实现错误链)
}

// 实现 error 接口
func (e *Error) Error() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[%s] Code=%d, Message=%s", e.Type, e.Code, e.Message))

	if e.Detail != nil {
		builder.WriteString(fmt.Sprintf(", Detail=%v", e.Detail))
	}

	if e.Err != nil {
		builder.WriteString(fmt.Sprintf("\nCaused by: %s", e.Err.Error()))
	}

	if len(e.Stack) > 0 {
		builder.WriteString("\nStack:\n")
		builder.WriteString(strings.Join(e.Stack, "\n"))
	}

	return builder.String()
}

// Unwrap 实现错误链 (Go 1.13+)
func (e *Error) Unwrap() error {
	return e.Err
}

// Is 实现错误类型匹配
func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return e.Code == t.Code && e.Type == t.Type
	}
	return false
}

// As 实现错误类型转换
func (e *Error) As(target interface{}) bool {
	if t, ok := target.(**Error); ok {
		*t = e
		return true
	}
	return false
}

// --------------------------
// 构造函数和工具方法
// --------------------------

// NewBaseError 创建基础错误 (自动捕获堆栈)
func NewBaseError(code int, errType string, message string, detail interface{}) *Error {
	return &Error{
		Code:    code,
		Type:    errType,
		Message: message,
		Detail:  detail,
		Time:    time.Now(),
		Stack:   captureStack(2), // 跳过当前函数和调用者
	}
}

// WrapError 包装现有错误 (实现错误链)
func WrapError(err error, code int, errType string, message string, detail interface{}) *Error {
	return &Error{
		Code:    code,
		Type:    errType,
		Message: message,
		Detail:  detail,
		Time:    time.Now(),
		Err:     err,
		Stack:   captureStack(2),
	}
}

// 捕获堆栈信息 (可控制深度)
func captureStack(skip int) []string {
	pc := make([]uintptr, 32)        // 最多记录32层堆栈
	n := runtime.Callers(skip+2, pc) // skip captureStack和New/Wrap函数
	if n == 0 {
		return nil
	}

	pc = pc[:n]
	frames := runtime.CallersFrames(pc)

	var stack []string
	for {
		frame, more := frames.Next()
		stack = append(stack, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	return stack
}

// --------------------------
// 快捷方法
// --------------------------

// BusinessError 快速创建业务错误
func BusinessError(code int, message string, detail interface{}) *Error {
	return NewBaseError(code, ErrorTypeBusiness, message, detail)
}

// SystemError 快速创建系统错误
func SystemError(code int, message string, detail interface{}) *Error {
	return NewBaseError(code, ErrorTypeSystem, message, detail)
}

// WrapBusinessError 包装错误为业务错误
func WrapBusinessError(err error, code int, message string, detail interface{}) *Error {
	return WrapError(err, code, ErrorTypeBusiness, message, detail)
}

// --------------------------
// 辅助方法
// --------------------------

// GetCode 获取错误码 (支持错误链查找)
func GetCode(err error) int {
	if e := getError(err); e != nil {
		return e.Code
	}
	return 0
}

// GetType 获取错误类型 (支持错误链查找)
func GetType(err error) string {
	if e := getError(err); e != nil {
		return e.Type
	}
	return ""
}

// 递归查找自定义错误
func getError(err error) *Error {
	for err != nil {
		if e, ok := err.(*Error); ok {
			return e
		}
		err = errors.Unwrap(err)
	}
	return nil
}

func TestError(t *testing.T) {
	// 示例1: 创建基础业务错误
	err1 := BusinessError(1001, "Invalid parameters", map[string]interface{}{
		"field": "username",
		"rule":  "length > 6",
	})
	fmt.Println("Error1:\n", err1)

	// 示例2: 包装原生错误
	_, err := os.Open("non-existent-file")
	if err != nil {
		fmt.Println("\nError2:\n", err)
	}

	// 示例3: 错误链处理
	if err := processRequest(); err != nil {
		fmt.Println("\nError3:\n", err)
		fmt.Println("Root Code:", GetCode(err))
	}
}

func processRequest() error {
	if err := validateInput(); err != nil {
		return WrapBusinessError(
			err,
			2001,
			"Request validation failed",
			nil,
		)
	}
	return nil
}

func validateInput() error {
	return BusinessError(1002, "Empty required field", "email")
}
