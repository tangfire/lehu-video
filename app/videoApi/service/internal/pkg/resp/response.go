package resp

import (
	"encoding/json"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/protobuf/proto"
)

// 统一响应格式
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// 成功响应
func success(data interface{}) *Response {
	return &Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// 转换错误为统一格式
func convertError(err error) *Response {
	// 如果是 Kratos 错误
	if kratosErr := errors.FromError(err); kratosErr != nil {
		return &Response{
			Code:      int(kratosErr.Code),
			Message:   kratosErr.Message,
			Data:      nil,
			Timestamp: time.Now().Unix(),
		}
	}

	// 其他错误
	return &Response{
		Code:      500,
		Message:   err.Error(),
		Data:      nil,
		Timestamp: time.Now().Unix(),
	}
}

// 自定义 HTTP 响应编码器
func ResponseEncoder(w http.ResponseWriter, r *http.Request, v interface{}) error {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// 如果已经是 Response 类型，直接编码
	if resp, ok := v.(*Response); ok {
		return json.NewEncoder(w).Encode(resp)
	}

	var data interface{}

	// 检查是否为 protobuf 消息
	if pm, ok := v.(proto.Message); ok {
		// 对于 protobuf 消息，使用自定义的转换函数
		data = convertProtoToMap(pm)
	} else {
		// 其他类型直接使用
		data = v
	}

	// 创建成功响应
	resp := success(data)
	return json.NewEncoder(w).Encode(resp)
}

// 将 protobuf 消息转换为 map，保持数字类型
func convertProtoToMap(pm proto.Message) interface{} {
	marshalOptions := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
		UseEnumNumbers:  true,
	}

	// 序列化为 JSON
	protoData, err := marshalOptions.Marshal(pm)
	if err != nil {
		return nil
	}

	// 反序列化为 map
	var result map[string]interface{}
	if err := json.Unmarshal(protoData, &result); err != nil {
		return nil
	}

	// 递归处理数字类型
	return fixNumberTypes(result)
}

// 递归修复数字类型
func fixNumberTypes(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			v[key] = fixNumberTypes(value)
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = fixNumberTypes(item)
		}
		return v
	case string:
		// 尝试将字符串转换为数字
		if num, err := strconv.ParseInt(v, 10, 64); err == nil {
			return num
		}
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num
		}
		return v
	default:
		return v
	}
}

// 自定义错误编码器
func ErrorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	resp := convertError(err)

	// 设置 HTTP 状态码
	var statusCode int
	switch resp.Code {
	case 0:
		statusCode = http.StatusOK
	case 400:
		statusCode = http.StatusBadRequest
	case 401:
		statusCode = http.StatusUnauthorized
	case 403:
		statusCode = http.StatusForbidden
	case 404:
		statusCode = http.StatusNotFound
	default:
		statusCode = http.StatusInternalServerError
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
