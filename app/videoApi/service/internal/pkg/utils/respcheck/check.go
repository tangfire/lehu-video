package respcheck

import "lehu-video/pkg/apperror"

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
		return apperror.New(apperror.CodeDependencyUnavailable, apperror.ReasonDependencyUnavailable, "下游服务响应为空")
	}

	code := meta.GetCode()
	if code != 0 {
		reason := apperror.ReasonInternal
		if reasons := meta.GetReason(); len(reasons) > 0 && reasons[0] != "" {
			reason = reasons[0]
		}
		message := meta.GetMessage()
		if message == "" {
			message = "下游服务处理失败"
		}
		return apperror.New(code, reason, message)
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
		return "service error"
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
	return meta.GetMessage()
}
