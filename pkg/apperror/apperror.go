package apperror

import (
	"errors"
	"fmt"
	"net/http"

	kerrors "github.com/go-kratos/kratos/v2/errors"
)

const (
	CodeOK                    int32 = 0
	CodeInvalidArgument       int32 = 40001
	CodeUnauthorized          int32 = 40100
	CodeForbidden             int32 = 40300
	CodeNotFound              int32 = 40400
	CodeConflict              int32 = 40900
	CodeTooManyRequests       int32 = 42900
	CodeDependencyUnavailable int32 = 50300
	CodeInternal              int32 = 50000
)

const (
	ReasonInvalidArgument       = "INVALID_ARGUMENT"
	ReasonUnauthorized          = "UNAUTHORIZED"
	ReasonForbidden             = "FORBIDDEN"
	ReasonNotFound              = "NOT_FOUND"
	ReasonConflict              = "CONFLICT"
	ReasonTooManyRequests       = "TOO_MANY_REQUESTS"
	ReasonDependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	ReasonInternal              = "INTERNAL"
)

type Error struct {
	Code    int32
	Reason  string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(code int32, reason, message string) *Error {
	return &Error{Code: code, Reason: reason, Message: message}
}

func Wrap(err error, code int32, reason, message string) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Reason: reason, Message: message, Cause: err}
}

func InvalidArgument(message string) *Error {
	return New(CodeInvalidArgument, ReasonInvalidArgument, message)
}

func Unauthorized(message string) *Error {
	return New(CodeUnauthorized, ReasonUnauthorized, message)
}

func Forbidden(message string) *Error {
	return New(CodeForbidden, ReasonForbidden, message)
}

func NotFound(message string) *Error {
	return New(CodeNotFound, ReasonNotFound, message)
}

func Conflict(message string) *Error {
	return New(CodeConflict, ReasonConflict, message)
}

func TooManyRequests(message string) *Error {
	return New(CodeTooManyRequests, ReasonTooManyRequests, message)
}

func DependencyUnavailable(err error, message string) error {
	return Wrap(err, CodeDependencyUnavailable, ReasonDependencyUnavailable, message)
}

func Internal(err error, message string) error {
	return Wrap(err, CodeInternal, ReasonInternal, message)
}

func From(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	var kratosErr *kerrors.Error
	if errors.As(err, &kratosErr) {
		return &Error{
			Code:    kratosErr.Code,
			Reason:  kratosErr.Reason,
			Message: kratosErr.Message,
			Cause:   err,
		}
	}
	return &Error{
		Code:    CodeInternal,
		Reason:  ReasonInternal,
		Message: "系统开小差了，请稍后再试",
		Cause:   err,
	}
}

func HTTPStatus(code int32) int {
	switch {
	case code == CodeOK:
		return http.StatusOK
	case code >= 40000 && code < 40100:
		return http.StatusBadRequest
	case code >= 40100 && code < 40200:
		return http.StatusUnauthorized
	case code >= 40300 && code < 40400:
		return http.StatusForbidden
	case code >= 40400 && code < 40500:
		return http.StatusNotFound
	case code >= 40900 && code < 41000:
		return http.StatusConflict
	case code >= 42900 && code < 43000:
		return http.StatusTooManyRequests
	case code >= 50300 && code < 50400:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
