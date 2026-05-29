package resp

import (
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"lehu-video/pkg/apperror"
)

type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

func success(data interface{}) *Response {
	return &Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

func requestIDFromHeader(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Header.Get("X-Request-ID")
}

func convertError(err error) *Response {
	appErr := apperror.From(err)
	return &Response{
		Code:      int(appErr.Code),
		Message:   appErr.Message,
		Data:      nil,
		Timestamp: time.Now().Unix(),
	}
}

func ResponseEncoder(w http.ResponseWriter, r *http.Request, v interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if resp, ok := v.(*Response); ok {
		return json.NewEncoder(w).Encode(resp)
	}

	var data interface{}
	if pm, ok := v.(proto.Message); ok {
		data = convertProtoToMap(pm)
	} else {
		data = v
	}

	resp := success(data)
	resp.RequestID = requestIDFromHeader(r)
	return json.NewEncoder(w).Encode(resp)
}

func ErrorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	resp := convertError(err)
	resp.RequestID = requestIDFromHeader(r)
	w.WriteHeader(apperror.HTTPStatus(int32(resp.Code)))
	_ = json.NewEncoder(w).Encode(resp)
}

func convertProtoToMap(pm proto.Message) interface{} {
	marshalOptions := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
		UseEnumNumbers:  true,
	}

	protoData, err := marshalOptions.Marshal(pm)
	if err != nil {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(protoData, &result); err != nil {
		return nil
	}

	return fixNumberTypes(result)
}

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
	default:
		return v
	}
}
