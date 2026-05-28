package resp

import (
	"encoding/json"
	"google.golang.org/protobuf/encoding/protojson"
	videoapi "lehu-video/api/videoApi/service/v1"
	"lehu-video/pkg/apperror"
	"net/http"
	"time"

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

func requestIDFromHeader(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Header.Get("X-Request-ID")
}

// 转换错误为统一格式
func convertError(err error) *Response {
	appErr := apperror.From(err)
	return &Response{
		Code:      int(appErr.Code),
		Message:   appErr.Message,
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
	resp.RequestID = requestIDFromHeader(r)
	return json.NewEncoder(w).Encode(resp)
}

// 将 protobuf 消息转换为 map，保持数字类型
func convertProtoToMap(pm proto.Message) interface{} {
	if data, ok := convertVideoProtoToMap(pm); ok {
		return data
	}

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

func convertVideoProtoToMap(pm proto.Message) (interface{}, bool) {
	switch msg := pm.(type) {
	case *videoapi.FeedShortVideoResp:
		videos := make([]interface{}, 0, len(msg.Videos))
		for _, video := range msg.Videos {
			videos = append(videos, convertVideoToMap(video))
		}
		return map[string]interface{}{
			"videos":    videos,
			"next_time": msg.NextTime,
		}, true
	case *videoapi.GetVideoByIdResp:
		return map[string]interface{}{
			"video": convertVideoToMap(msg.Video),
		}, true
	case *videoapi.ListPublishedVideoResp:
		videos := make([]interface{}, 0, len(msg.VideoList))
		for _, video := range msg.VideoList {
			videos = append(videos, convertVideoToMap(video))
		}
		return map[string]interface{}{
			"video_list": videos,
			"page_stats": convertPageStatsToMap(msg.PageStats),
		}, true
	default:
		return nil, false
	}
}

func convertVideoToMap(video *videoapi.Video) map[string]interface{} {
	if video == nil {
		return nil
	}
	return map[string]interface{}{
		"id":             video.Id,
		"author":         convertVideoAuthorToMap(video.Author),
		"play_url":       video.PlayUrl,
		"cover_url":      video.CoverUrl,
		"favoriteCount":  video.FavoriteCount,
		"commentCount":   video.CommentCount,
		"view_count":     video.ViewCount,
		"isFavorite":     video.IsFavorite,
		"title":          video.Title,
		"description":    video.Description,
		"upload_time":    video.UploadTime,
		"isCollected":    video.IsCollected,
		"collectedCount": video.CollectedCount,
	}
}

func convertVideoAuthorToMap(author *videoapi.VideoAuthor) map[string]interface{} {
	if author == nil {
		return nil
	}
	return map[string]interface{}{
		"id":          author.Id,
		"name":        author.Name,
		"avatar":      author.Avatar,
		"isFollowing": author.IsFollowing,
	}
}

func convertPageStatsToMap(stats *videoapi.PageStatsResp) map[string]interface{} {
	if stats == nil {
		return nil
	}
	return map[string]interface{}{
		"total": stats.Total,
	}
}

// 递归修复数字类型
//func fixNumberTypes(data interface{}) interface{} {
//	switch v := data.(type) {
//	case map[string]interface{}:
//		for key, value := range v {
//			v[key] = fixNumberTypes(value)
//		}
//		return v
//	case []interface{}:
//		for i, item := range v {
//			v[i] = fixNumberTypes(item)
//		}
//		return v
//	case string:
//		// 尝试将字符串转换为数字
//		if num, err := strconv.ParseInt(v, 10, 64); err == nil {
//			return num
//		}
//		if num, err := strconv.ParseFloat(v, 64); err == nil {
//			return num
//		}
//		return v
//	default:
//		return v
//	}
//}

// 递归修复数字类型 - 改进版本
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

// 自定义错误编码器
func ErrorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	resp := convertError(err)
	resp.RequestID = requestIDFromHeader(r)

	w.WriteHeader(apperror.HTTPStatus(int32(resp.Code)))
	json.NewEncoder(w).Encode(resp)
}
