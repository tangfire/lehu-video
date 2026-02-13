// api/videoApi/service/internal/biz/feed_item.go
package biz

type FeedItem struct {
	VideoID   string
	AuthorID  string
	Timestamp int64
	Score     float64
}
