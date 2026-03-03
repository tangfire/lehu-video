package biz

import (
	"github.com/google/wire"
	"lehu-video/app/videoCore/service/internal/deprecated"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewVideoUsecase,
	NewUserUsecase,
	NewFavoriteUsecase,
	NewCommentUsecase,
	NewFollowUsecase,
	NewCollectionUsecase,
	NewFeedUsecase,
	NewKafkaConsumer,
	NewKafkaProducer,
	deprecated.NewHotVideoDetector,
	NewRecentViewedManager,
)

type PageStats struct {
	Page     int32
	PageSize int32
}
