package respcheck

import (
	"fmt"
	"strings"
)

// MetaInfo 定义了所有Metadata共有的接口
type MetaInfo interface {
	GetCode() int32
	GetMessage() string
	GetData() string
	GetReason() []string
}

// ValidateResponseMeta 通用的Metadata校验函数
func ValidateResponseMeta(meta MetaInfo) error {
	if meta == nil {
		return fmt.Errorf("response metadata is nil")
	}

	code := meta.GetCode()
	if code != 0 {
		// 收集错误信息
		var errorDetails []string
		errorDetails = append(errorDetails, fmt.Sprintf("code: %d", code))

		if msg := meta.GetMessage(); msg != "" {
			errorDetails = append(errorDetails, fmt.Sprintf("message: %s", msg))
		}

		if reasons := meta.GetReason(); len(reasons) > 0 {
			errorDetails = append(errorDetails, fmt.Sprintf("reasons: [%s]", strings.Join(reasons, ", ")))
		}

		return fmt.Errorf("service error: %s", strings.Join(errorDetails, ", "))
	}

	return nil
}

// GetErrorCode 获取错误码
func GetErrorCode(meta MetaInfo) int32 {
	if meta == nil {
		return -1
	}
	return meta.GetCode()
}

// GetErrorMessage 获取错误信息
func GetErrorMessage(meta MetaInfo) string {
	if meta == nil {
		return "response metadata is nil"
	}

	code := meta.GetCode()
	if code == 0 {
		return "success"
	}

	msg := meta.GetMessage()
	if msg == "" {
		return fmt.Sprintf("error with code: %d", code)
	}
	return msg
}

// IsSuccess 判断是否成功
func IsSuccess(meta MetaInfo) bool {
	if meta == nil {
		return false
	}
	return meta.GetCode() == 0
}

// GetReasons 获取原因列表
func GetReasons(meta MetaInfo) []string {
	if meta == nil {
		return []string{"metadata is nil"}
	}
	return meta.GetReason()
}

// FormatError 格式化错误信息
func FormatError(meta MetaInfo) string {
	if meta == nil {
		return "Response metadata is nil"
	}

	code := meta.GetCode()
	if code == 0 {
		return "Success"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Error[%d]: %s", code, meta.GetMessage()))

	if reasons := meta.GetReason(); len(reasons) > 0 {
		builder.WriteString("\nReasons:")
		for i, reason := range reasons {
			builder.WriteString(fmt.Sprintf("\n  %d. %s", i+1, reason))
		}
	}

	return builder.String()
}
