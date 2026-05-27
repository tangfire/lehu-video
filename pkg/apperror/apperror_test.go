package apperror

import "testing"

func TestFromPreservesAppError(t *testing.T) {
	err := InvalidArgument("参数错误")
	got := From(err)
	if got.Code != CodeInvalidArgument || got.Reason != ReasonInvalidArgument || got.Message != "参数错误" {
		t.Fatalf("From() = %#v", got)
	}
}

func TestHTTPStatus(t *testing.T) {
	cases := map[int32]int{
		CodeInvalidArgument: 400,
		CodeUnauthorized:    401,
		CodeForbidden:       403,
		CodeNotFound:        404,
		CodeConflict:        409,
		CodeTooManyRequests: 429,
		CodeInternal:        500,
	}
	for code, want := range cases {
		if got := HTTPStatus(code); got != want {
			t.Fatalf("HTTPStatus(%d) = %d, want %d", code, got, want)
		}
	}
}
