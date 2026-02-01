package data

import (
	"context"
	"github.com/spf13/cast"
	core "lehu-video/api/videoCore/service/v1"
	"lehu-video/app/videoApi/service/internal/biz"
	"lehu-video/app/videoApi/service/internal/pkg/utils/respcheck"
)

// 实现CoreAdapter接口
func (r *CoreAdapterImpl) GetUserInfo(ctx context.Context, userId, accountId string) (*biz.UserInfo, error) {
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

func (r *CoreAdapterImpl) UpdateUserInfo(ctx context.Context, userId, name, avatar, backgroundImage, signature string) error {
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

func (r *CoreAdapterImpl) GetUserInfoByIdList(ctx context.Context, userIdList []string) ([]*biz.UserInfo, error) {
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
			Id:              cast.ToString(user.Id),
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
