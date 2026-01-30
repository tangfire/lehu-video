package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

type CoreAdapterImpl struct {
	user       core.UserServiceClient
	video      core.VideoServiceClient
	collection core.CollectionServiceClient
	comment    core.CommentServiceClient
	favorite   core.FavoriteServiceClient
	follow     core.FollowServiceClient
}

func NewCoreAdapter(
	user core.UserServiceClient,
	video core.VideoServiceClient,
	collection core.CollectionServiceClient,
	comment core.CommentServiceClient,
	favorite core.FavoriteServiceClient,
	follow core.FollowServiceClient) biz.CoreAdapter {
	return &CoreAdapterImpl{
		user:       user,
		video:      video,
		collection: collection,
		comment:    comment,
		favorite:   favorite,
		follow:     follow,
	}
}

func NewUserServiceClient(r registry.Discovery) core.UserServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewUserServiceClient(conn)
}

func NewVideoServiceClient(r registry.Discovery) core.VideoServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewVideoServiceClient(conn)
}

func NewCollectionServiceClient(r registry.Discovery) core.CollectionServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewCollectionServiceClient(conn)
}

func NewCommentServiceClient(r registry.Discovery) core.CommentServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewCommentServiceClient(conn)
}

func NewFavoriteServiceClient(r registry.Discovery) core.FavoriteServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewFavoriteServiceClient(conn)
}

func NewFollowServiceClient(r registry.Discovery) core.FollowServiceClient {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///lehu-video.core.service"),
		grpc.WithDiscovery(r),
		grpc.WithMiddleware(
			recovery.Recovery(),
		),
	)
	if err != nil {
		panic(err)
	}
	return core.NewFollowServiceClient(conn)
}

// 实现CoreAdapter接口
func (r *CoreAdapterImpl) GetUserInfo(ctx context.Context, userId, accountId int64) (*biz.UserInfo, error) {
	resp, err := r.user.GetUserInfo(ctx, &core.GetUserInfoReq{
		UserId:    userId,
		AccountId: accountId,
	})
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	user := resp.User
	return &biz.UserInfo{
		Id:              user.Id,
		Name:            user.Name,
		Nickname:        user.Nickname,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
		Mobile:          user.Mobile,
		Email:           user.Email,
		Gender:          user.Gender,
	}, nil
}

func (r *CoreAdapterImpl) UpdateUserInfo(ctx context.Context, userId int64, name, avatar, backgroundImage, signature string) error {
	req := &core.UpdateUserInfoReq{
		UserId:          userId,
		Name:            name,
		Avatar:          avatar,
		BackgroundImage: backgroundImage,
		Signature:       signature,
	}

	resp, err := r.user.UpdateUser(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) GetUserInfoByIdList(ctx context.Context, userIdList []int64) ([]*biz.UserInfo, error) {
	resp, err := r.user.GetUserByIdList(ctx, &core.GetUserByIdListReq{
		UserIdList: userIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	var retUserInfos []*biz.UserInfo
	for _, user := range resp.UserList {
		retUserInfos = append(retUserInfos, &biz.UserInfo{
			Id:              user.Id,
			Name:            user.Name,
			Nickname:        user.Nickname,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			Mobile:          user.Mobile,
			Email:           user.Email,
			Gender:          user.Gender,
		})
	}
	return retUserInfos, nil
}

func (r *CoreAdapterImpl) CountFollow4User(ctx context.Context, userId int64) ([]int64, error) {
	resp, err := r.follow.CountFollow(ctx, &core.CountFollowReq{UserId: userId})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	return []int64{resp.FollowingCount, resp.FollowerCount}, nil
}

func (r *CoreAdapterImpl) CountBeFavoriteNumber4User(ctx context.Context, userId int64) (int64, error) {
	resp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		Id:            []int64{userId},
		AggregateType: core.FavoriteAggregateType_BY_USER,
		FavoriteType:  core.FavoriteType_FAVORITE,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.Items[0].Count, nil
}

func (r *CoreAdapterImpl) CreateUser(ctx context.Context, mobile, email string, accountId int64) (int64, error) {
	resp, err := r.user.CreateUser(ctx, &core.CreateUserReq{
		Mobile:    mobile,
		Email:     email,
		AccountId: accountId,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.UserId, nil
}

func (r *CoreAdapterImpl) SaveVideoInfo(ctx context.Context, title, videoUrl, coverUrl, desc string, userId int64) (int64, error) {
	resp, err := r.video.PublishVideo(ctx, &core.PublishVideoReq{
		Title:       title,
		CoverUrl:    coverUrl,
		PlayUrl:     videoUrl,
		Description: desc,
		UserId:      userId,
	})
	if err != nil {
		return 0, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, err
	}
	return resp.VideoId, nil
}

func (r *CoreAdapterImpl) GetVideoById(ctx context.Context, videoId int64) (*biz.Video, error) {
	resp, err := r.video.GetVideoById(ctx, &core.GetVideoByIdReq{
		VideoId: videoId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	video := resp.Video
	author := video.Author
	retVideo := &biz.Video{
		ID: video.Id,
		Author: &biz.VideoAuthor{
			ID:          author.Id,
			Name:        author.Name,
			Avatar:      author.Avatar,
			IsFollowing: author.IsFollowing != 0,
		},
		PlayURL:       video.PlayUrl,
		CoverURL:      video.CoverUrl,
		FavoriteCount: video.FavoriteCount,
		CommentCount:  video.CommentCount,
		IsFavorite:    video.IsFavorite != 0,
		Title:         video.Title,
	}
	return retVideo, nil
}

func (r *CoreAdapterImpl) ListPublishedVideo(ctx context.Context, userId int64, pageStats *biz.PageStats) (int64, []*biz.Video, error) {
	resp, err := r.video.ListPublishedVideo(ctx, &core.ListPublishedVideoReq{
		UserId: userId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	var retVideos []*biz.Video
	for _, video := range resp.Videos {
		retVideos = append(retVideos, &biz.Video{
			ID: video.Id,
			Author: &biz.VideoAuthor{
				ID:          video.Author.Id,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing != 0,
			},
			PlayURL:       video.PlayUrl,
			CoverURL:      video.CoverUrl,
			FavoriteCount: video.IsFavorite,
			CommentCount:  video.CommentCount,
			IsFavorite:    video.IsFavorite != 0,
			Title:         video.Title,
		})
	}
	return int64(resp.PageStats.Total), retVideos, nil
}

func (r *CoreAdapterImpl) IsUserFavoriteVideo(ctx context.Context, userId int64, videoIdList []int64) (map[int64]bool, error) {

	var items []*core.IsFavoriteReqItem
	for _, id := range videoIdList {
		items = append(items, &core.IsFavoriteReqItem{
			BizId:  id,
			UserId: userId,
		})
	}

	resp, err := r.favorite.IsFavorite(ctx, &core.IsFavoriteReq{
		Items: items,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]bool)
	if len(resp.Items) == 0 {
		return result, nil
	}

	for _, item := range resp.Items {
		result[item.BizId] = item.IsFavorite
	}
	return result, nil
}

func (r *CoreAdapterImpl) IsFollowing(ctx context.Context, userId int64, targetUserIdList []int64) (map[int64]bool, error) {
	resp, err := r.follow.IsFollowing(ctx, &core.IsFollowingReq{
		UserId:           userId,
		TargetUserIdList: targetUserIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]bool)
	if len(resp.FollowingList) == 0 {
		return result, nil
	}

	for _, item := range resp.FollowingList {
		result[item] = true
	}
	return result, nil
}

func (r *CoreAdapterImpl) IsCollected(ctx context.Context, userId int64, videoIdList []int64) (map[int64]bool, error) {
	resp, err := r.collection.IsCollected(ctx, &core.IsCollectedReq{
		UserId:      userId,
		VideoIdList: videoIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]bool)
	if len(resp.VideoIdList) == 0 {
		return result, nil
	}

	for _, item := range resp.VideoIdList {
		result[item] = true
	}

	return result, nil
}

func (r *CoreAdapterImpl) CountComments4Video(ctx context.Context, videoIdList []int64) (map[int64]int64, error) {
	resp, err := r.comment.CountComment4Video(ctx, &core.CountComment4VideoReq{
		VideoId: videoIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int64)
	for _, item := range resp.Results {
		result[item.Id] = item.Count
	}

	return result, nil
}

func (r *CoreAdapterImpl) CountFavorite4Video(ctx context.Context, videoIdList []int64) (map[int64]int64, error) {
	resp, err := r.favorite.CountFavorite(ctx, &core.CountFavoriteReq{
		AggregateType: core.FavoriteAggregateType_BY_VIDEO,
		Id:            videoIdList,
		FavoriteType:  core.FavoriteType_FAVORITE,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int64)
	for _, item := range resp.Items {
		result[item.BizId] = item.Count
	}
	return result, nil
}

func (r *CoreAdapterImpl) CountCollected4Video(ctx context.Context, videoIdList []int64) (map[int64]int64, error) {
	resp, err := r.collection.CountCollect4Video(ctx, &core.CountCollect4VideoReq{
		VideoIdList: videoIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int64)
	for _, item := range resp.CountResult {
		result[item.Id] = item.Count
	}

	return result, nil
}

func (r *CoreAdapterImpl) Feed(ctx context.Context, userId int64, num int64, latestTime int64) ([]*biz.Video, error) {
	req := &core.FeedShortVideoReq{
		LatestTime: latestTime,
		UserId:     userId,
		FeedNum:    num,
	}

	resp, err := r.video.FeedShortVideo(ctx, req)
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	var videos []*biz.Video
	for _, video := range resp.Videos {
		videos = append(videos, &biz.Video{
			ID: video.Id,
			Author: &biz.VideoAuthor{
				ID:          video.Author.Id,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing != 0,
			},
			PlayURL:       video.PlayUrl,
			CoverURL:      video.CoverUrl,
			FavoriteCount: video.FavoriteCount,
			CommentCount:  video.CommentCount,
			IsFavorite:    video.IsFavorite != 0,
			Title:         video.Title,
		})
	}
	return videos, nil
}

func (r *CoreAdapterImpl) CreateComment(ctx context.Context, userId int64, content string, videoId int64, parentId int64, replyUserId int64) (*biz.Comment, error) {
	resp, err := r.comment.CreateComment(ctx, &core.CreateCommentReq{
		VideoId:     videoId,
		UserId:      userId,
		Content:     content,
		ParentId:    parentId,
		ReplyUserId: replyUserId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	comment := resp.Comment
	childComments := comment.Comments
	var retChildComments []*biz.Comment
	for _, childComment := range childComments {
		retChildComments = append(retChildComments, &biz.Comment{
			Id:         childComment.Id,
			VideoId:    childComment.VideoId,
			ParentId:   childComment.ParentId,
			User:       nil,
			ReplyUser:  nil,
			Content:    childComment.Content,
			Date:       childComment.Date,
			LikeCount:  childComment.LikeCount,
			ReplyCount: childComment.ReplyCount,
			Comments:   nil,
		})
	}
	retComment := &biz.Comment{
		Id:       comment.Id,
		VideoId:  comment.VideoId,
		ParentId: comment.ParentId,
		User: &biz.CommentUser{
			Id:          comment.UserId,
			Name:        "",
			Avatar:      "",
			IsFollowing: false,
		},
		ReplyUser: &biz.CommentUser{
			Id:          comment.ReplyUserId,
			Name:        "",
			Avatar:      "",
			IsFollowing: false,
		},
		Content:    comment.Content,
		Date:       comment.Date,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		Comments:   retChildComments,
	}
	return retComment, nil
}

func (r *CoreAdapterImpl) GetCommentById(ctx context.Context, commentId int64) (*biz.Comment, error) {
	resp, err := r.comment.GetCommentById(ctx, &core.GetCommentByIdReq{
		CommentId: commentId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	comment := resp.Comment
	retComment := &biz.Comment{
		Id:         comment.Id,
		VideoId:    comment.VideoId,
		ParentId:   comment.ParentId,
		Content:    comment.Content,
		Date:       comment.Date,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		Comments:   nil,
		User: &biz.CommentUser{
			Id: comment.UserId,
		},
	}
	return retComment, nil
}

func (r *CoreAdapterImpl) RemoveComment(ctx context.Context, commentId, userId int64) error {
	resp, err := r.comment.RemoveComment(ctx, &core.RemoveCommentReq{
		CommentId: commentId,
		UserId:    userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) ListChildComment(ctx context.Context, commentId int64, pageStats *biz.PageStats) (int64, []*biz.Comment, error) {
	resp, err := r.comment.ListChildComment4Comment(ctx, &core.ListChildComment4CommentReq{
		CommentId: commentId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	var childComments []*biz.Comment
	for _, comment := range resp.CommentList {
		childComments = append(childComments, &biz.Comment{
			Id:         comment.Id,
			VideoId:    comment.VideoId,
			ParentId:   comment.ParentId,
			User:       &biz.CommentUser{Id: comment.UserId},
			ReplyUser:  &biz.CommentUser{Id: comment.ReplyUserId},
			Content:    comment.Content,
			Date:       comment.Date,
			LikeCount:  comment.LikeCount,
			ReplyCount: comment.ReplyCount,
			Comments:   nil,
		})
	}
	return int64(resp.PageStats.Total), childComments, nil
}

func (r *CoreAdapterImpl) ListComment4Video(ctx context.Context, videoId int64, pageStats *biz.PageStats) (int64, []*biz.Comment, error) {
	resp, err := r.comment.ListComment4Video(ctx, &core.ListComment4VideoReq{
		VideoId: videoId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	var Comments []*biz.Comment
	for _, comment := range resp.CommentList {
		childComments := comment.Comments
		var retChildComments []*biz.Comment
		for _, childComment := range childComments {
			retChildComments = append(retChildComments, &biz.Comment{
				Id:         childComment.Id,
				VideoId:    childComment.VideoId,
				ParentId:   childComment.ParentId,
				User:       &biz.CommentUser{Id: childComment.Id},
				ReplyUser:  &biz.CommentUser{Id: childComment.ReplyUserId},
				Content:    childComment.Content,
				Date:       childComment.Date,
				LikeCount:  childComment.LikeCount,
				ReplyCount: childComment.ReplyCount,
				Comments:   nil,
			})
		}
		Comments = append(Comments, &biz.Comment{
			Id:         comment.Id,
			VideoId:    comment.VideoId,
			ParentId:   comment.ParentId,
			User:       &biz.CommentUser{Id: comment.UserId},
			ReplyUser:  &biz.CommentUser{Id: comment.ReplyUserId},
			Content:    comment.Content,
			Date:       comment.Date,
			LikeCount:  comment.LikeCount,
			ReplyCount: comment.ReplyCount,
			Comments:   retChildComments,
		})
	}
	return int64(resp.PageStats.Total), Comments, nil
}

func (r *CoreAdapterImpl) AddFavorite(ctx context.Context, id, userId int64, target *biz.FavoriteTarget, _type *biz.FavoriteType) error {

	resp, err := r.favorite.AddFavorite(ctx, &core.AddFavoriteReq{
		Target: core.FavoriteTarget(*target), // 类型转换
		Type:   core.FavoriteType(*_type),    // 类型转换
		Id:     id,
		UserId: userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) RemoveFavorite(ctx context.Context, id, userId int64, target *biz.FavoriteTarget, _type *biz.FavoriteType) error {
	resp, err := r.favorite.RemoveFavorite(ctx, &core.RemoveFavoriteReq{
		Target: core.FavoriteTarget(*target), // 类型转换
		Type:   core.FavoriteType(*_type),    // 类型转换
		Id:     id,
		UserId: userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) ListUserFavoriteVideo(ctx context.Context, userId int64, pageStats *biz.PageStats) (int64, []int64, error) {
	resp, err := r.favorite.ListFavorite(ctx, &core.ListFavoriteReq{
		Id:            userId,
		AggregateType: core.FavoriteAggregateType_BY_USER,
		FavoriteType:  core.FavoriteType_FAVORITE,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	return int64(resp.PageStats.Total), resp.Ids, nil
}

func (r *CoreAdapterImpl) GetVideoByIdList(ctx context.Context, videoIdList []int64) ([]*biz.Video, error) {
	resp, err := r.video.GetVideoByIdList(ctx, &core.GetVideoByIdListReq{
		VideoIdList: videoIdList,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	videos := resp.Videos
	var retVideos []*biz.Video
	for _, video := range videos {
		retVideos = append(retVideos, &biz.Video{
			ID: video.Id,
			Author: &biz.VideoAuthor{
				ID:          video.Author.Id,
				Name:        video.Author.Name,
				Avatar:      video.Author.Avatar,
				IsFollowing: video.Author.IsFollowing != 0,
			},
			PlayURL:       video.PlayUrl,
			CoverURL:      video.CoverUrl,
			FavoriteCount: video.FavoriteCount,
			CommentCount:  video.CommentCount,
			IsFavorite:    video.IsFavorite != 0,
			Title:         video.Title,
		})
	}
	return retVideos, nil
}

func (r *CoreAdapterImpl) AddFollow(ctx context.Context, userId, targetUserId int64) error {
	resp, err := r.follow.AddFollow(ctx, &core.AddFollowReq{
		UserId:       userId,
		TargetUserId: targetUserId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}
func (r *CoreAdapterImpl) RemoveFollow(ctx context.Context, userId, targetUserId int64) error {
	resp, err := r.follow.RemoveFollow(ctx, &core.RemoveFollowReq{
		UserId:       userId,
		TargetUserId: targetUserId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) ListFollow(ctx context.Context, userId int64, _type *biz.FollowType, pageStats *biz.PageStats) (int64, []int64, error) {
	followType := core.FollowType(*_type)
	resp, err := r.follow.ListFollowing(ctx, &core.ListFollowingReq{
		UserId:     userId,
		FollowType: followType,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	return int64(resp.PageStats.Total), resp.UserIdList, nil
}

func (r *CoreAdapterImpl) GetCollectionById(ctx context.Context, collectionId int64) (*biz.Collection, error) {
	resp, err := r.collection.GetCollectionById(ctx, &core.GetCollectionByIdReq{
		Id: collectionId,
	})
	if err != nil {
		return nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}
	collection := resp.Collection
	retCollection := &biz.Collection{
		Id:          collection.Id,
		UserId:      collection.UserId,
		Name:        collection.Name,
		Description: collection.Description,
	}
	return retCollection, nil
}
func (r *CoreAdapterImpl) AddVideo2Collection(ctx context.Context, userId, collectionId, videoId int64) error {
	resp, err := r.collection.AddVideo2Collection(ctx, &core.AddVideo2CollectionReq{
		CollectionId: collectionId,
		VideoId:      videoId,
		UserId:       userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}
func (r *CoreAdapterImpl) AddCollection(ctx context.Context, collection *biz.Collection) error {
	resp, err := r.collection.CreateCollection(ctx, &core.CreateCollectionReq{
		Name:        collection.Name,
		Description: collection.Description,
		UserId:      collection.UserId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}
func (r *CoreAdapterImpl) ListCollection(ctx context.Context, userId int64, pageStats *biz.PageStats) (int64, []*biz.Collection, error) {
	resp, err := r.collection.ListCollection(ctx, &core.ListCollectionReq{
		UserId: userId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	collections := resp.Collections
	var retCollections []*biz.Collection
	for _, collection := range collections {
		retCollections = append(retCollections, &biz.Collection{
			Id:          collection.Id,
			UserId:      collection.UserId,
			Name:        collection.Name,
			Description: collection.Description,
		})
	}
	return int64(resp.PageStats.Total), retCollections, nil
}

func (r *CoreAdapterImpl) ListVideo4Collection(ctx context.Context, collectionId int64, pageStats *biz.PageStats) (int64, []int64, error) {
	resp, err := r.collection.ListVideo4Collection(ctx, &core.ListVideo4CollectionReq{
		CollectionId: collectionId,
		PageStats: &core.PageStatsReq{
			Page: int32(pageStats.Page),
			Size: int32(pageStats.PageSize),
		},
	})
	if err != nil {
		return 0, nil, err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}
	return int64(resp.PageStats.Total), resp.VideoIdList, nil
}

func (r *CoreAdapterImpl) RemoveCollection(ctx context.Context, userId, collectionId int64) error {
	resp, err := r.collection.RemoveCollection(ctx, &core.RemoveCollectionReq{
		Id:     collectionId,
		UserId: userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) RemoveVideo4Collection(ctx context.Context, userId, collectionId int64, videoId int64) error {
	resp, err := r.collection.RemoveVideoFromCollection(ctx, &core.RemoveVideoFromCollectionReq{
		CollectionId: collectionId,
		VideoId:      videoId,
		UserId:       userId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

func (r *CoreAdapterImpl) UpdateCollection(ctx context.Context, collection *biz.Collection) error {
	resp, err := r.collection.UpdateCollection(ctx, &core.UpdateCollectionReq{
		Id:          collection.Id,
		Name:        collection.Name,
		Description: collection.Description,
		UserId:      collection.UserId,
	})
	if err != nil {
		return err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}
