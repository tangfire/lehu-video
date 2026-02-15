package biz

// LikeEvent 点赞/点踩事件（用于Kafka消费者聚合）
type LikeEvent struct {
	UserID       int64 `json:"user_id"`
	TargetID     int64 `json:"target_id"`
	TargetType   int32 `json:"target_type"`
	FavoriteType int32 `json:"favorite_type"`
	IsDeleted    bool  `json:"is_deleted"`
	Timestamp    int64 `json:"timestamp"`
}
