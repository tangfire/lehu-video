package data

import (
	"context"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

// 实现CoreAdapter接口
func (r *CoreAdapterImpl) GetUserBaseInfo(ctx context.Context, userID, accountID string) (*biz.UserBaseInfo, error) {
	resp, err := r.user.GetUserBaseInfo(ctx, &core.GetUserBaseInfoReq{
		UserId:    userID,
		AccountId: accountID,
	})
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	return convertToBizUserBaseInfo(resp.User), nil
}

// 转换函数
func convertToBizUserBaseInfo(user *core.UserBaseInfo) *biz.UserBaseInfo {
	if user == nil {
		return nil
	}

	return &biz.UserBaseInfo{
		ID:              user.Id,
		Name:            user.Name,
		Nickname:        user.Nickname,
		Avatar:          user.Avatar,
		BackgroundImage: user.BackgroundImage,
		Signature:       user.Signature,
		Mobile:          user.Mobile,
		Email:           user.Email,
		Gender:          user.Gender,
		FollowCount:     user.FollowCount,
		FollowerCount:   user.FollowerCount,
		TotalFavorited:  user.TotalFavorited,
		WorkCount:       user.WorkCount,
		FavoriteCount:   user.FavoriteCount,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}

// UpdateUserInfo 修复方法签名
func (r *CoreAdapterImpl) UpdateUserInfo(ctx context.Context, userID, name, nickName, avatar, backgroundImage, signature string, gender int32) error {
	req := &core.UpdateUserInfoReq{
		UserId:          userID,
		Name:            name,
		Nickname:        nickName,
		Avatar:          avatar,
		BackgroundImage: backgroundImage,
		Signature:       signature,
		Gender:          gender,
	}

	resp, err := r.user.UpdateUserInfo(ctx, req)
	if err != nil {
		return err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return err
	}
	return nil
}

// GetUserInfoByIdList 修复返回类型
func (r *CoreAdapterImpl) GetUserInfoByIdList(ctx context.Context, userIdList []string) ([]*biz.UserInfo, error) {
	resp, err := r.user.BatchGetUserBaseInfo(ctx, &core.BatchGetUserBaseInfoReq{
		UserIds: userIdList,
	})
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	// 注意：这里返回的是 []*biz.UserBaseInfo，不是 []*biz.UserInfo
	// 我们需要转换成 []*biz.UserInfo
	var retUserInfos []*biz.UserInfo
	for _, user := range resp.Users {
		retUserInfos = append(retUserInfos, &biz.UserInfo{
			ID:              user.Id,
			Name:            user.Name,
			Nickname:        user.Nickname,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			Mobile:          user.Mobile,
			Email:           user.Email,
			Gender:          user.Gender,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorited:  user.TotalFavorited,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
			CreatedAt:       user.CreatedAt,
			// OnlineStatus 和 LastOnlineTime 需要从chat服务获取
		})
	}
	return retUserInfos, nil
}

func (r *CoreAdapterImpl) CreateUser(ctx context.Context, mobile, email, accountId string) (string, error) {
	resp, err := r.user.CreateUser(ctx, &core.CreateUserReq{
		Mobile:    mobile,
		Email:     email,
		AccountId: accountId,
	})
	if err != nil {
		return "0", err
	}
	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return "0", err
	}
	return resp.UserId, nil
}

// SearchUsers 实现接口方法
func (r *CoreAdapterImpl) SearchUsers(ctx context.Context, keyword string, page, pageSize int32) (int64, []*biz.UserBaseInfo, error) {
	resp, err := r.user.SearchUsers(ctx, &core.SearchUsersReq{
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return 0, nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return 0, nil, err
	}

	users := make([]*biz.UserBaseInfo, 0, len(resp.Users))
	for _, user := range resp.Users {
		users = append(users, &biz.UserBaseInfo{
			ID:              user.Id,
			Name:            user.Name,
			Nickname:        user.Nickname,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			Mobile:          user.Mobile,
			Email:           user.Email,
			Gender:          user.Gender,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorited:  user.TotalFavorited,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
			CreatedAt:       user.CreatedAt,
			UpdatedAt:       user.UpdatedAt,
		})
	}

	return int64(resp.Total), users, nil
}

// 添加缺失的方法实现
func (r *CoreAdapterImpl) BatchGetUserBaseInfo(ctx context.Context, userIDs []string) ([]*biz.UserBaseInfo, error) {
	resp, err := r.user.BatchGetUserBaseInfo(ctx, &core.BatchGetUserBaseInfoReq{
		UserIds: userIDs,
	})
	if err != nil {
		return nil, err
	}

	err = respcheck.ValidateResponseMeta(resp.Meta)
	if err != nil {
		return nil, err
	}

	users := make([]*biz.UserBaseInfo, 0, len(resp.Users))
	for _, user := range resp.Users {
		users = append(users, &biz.UserBaseInfo{
			ID:              user.Id,
			Name:            user.Name,
			Nickname:        user.Nickname,
			Avatar:          user.Avatar,
			BackgroundImage: user.BackgroundImage,
			Signature:       user.Signature,
			Mobile:          user.Mobile,
			Email:           user.Email,
			Gender:          user.Gender,
			FollowCount:     user.FollowCount,
			FollowerCount:   user.FollowerCount,
			TotalFavorited:  user.TotalFavorited,
			WorkCount:       user.WorkCount,
			FavoriteCount:   user.FavoriteCount,
			CreatedAt:       user.CreatedAt,
			UpdatedAt:       user.UpdatedAt,
		})
	}

	return users, nil
}
