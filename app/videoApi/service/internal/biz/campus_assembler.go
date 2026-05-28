package biz

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

type CampusPostAssembler struct {
	repo CampusRepo
	core CoreAdapter
	log  *log.Helper
}

func NewCampusPostAssembler(repo CampusRepo, core CoreAdapter, logger log.Logger) *CampusPostAssembler {
	return &CampusPostAssembler{
		repo: repo,
		core: core,
		log:  log.NewHelper(logger),
	}
}

func (a *CampusPostAssembler) HydratePosts(ctx context.Context, posts []*CampusForumPost, currentUserID string) error {
	if len(posts) == 0 {
		return nil
	}
	userIDs := make([]string, 0, len(posts))
	postIDs := make([]int64, 0, len(posts))
	seen := map[string]struct{}{}
	for _, post := range posts {
		if post == nil {
			continue
		}
		postIDs = append(postIDs, post.ID)
		appendUniqueUserID(&userIDs, seen, post.AuthorID)
	}
	authors, err := a.LoadAuthors(ctx, userIDs)
	if err != nil {
		return err
	}
	likeStatus := map[int64]bool{}
	collectionStatus := map[int64]bool{}
	if currentUserID != "" && currentUserID != "0" {
		likeStatus, _ = a.repo.GetPostLikeStatus(ctx, currentUserID, postIDs)
		collectionStatus, _ = a.repo.GetPostCollectionStatus(ctx, currentUserID, postIDs)
	}
	for _, post := range posts {
		if post == nil {
			continue
		}
		post.Author = authors[post.AuthorID]
		post.IsLiked = likeStatus[post.ID]
		post.IsCollected = collectionStatus[post.ID]
	}
	return nil
}

func (a *CampusPostAssembler) HydrateComments(ctx context.Context, comments []*CampusForumComment, currentUserID string) error {
	if len(comments) == 0 {
		return nil
	}
	flat := flattenComments(comments)
	userIDs := make([]string, 0, len(flat)*2)
	seenUsers := map[string]struct{}{}
	commentIDs := make([]int64, 0, len(flat))
	for _, comment := range flat {
		if comment == nil {
			continue
		}
		commentIDs = append(commentIDs, comment.ID)
		appendUniqueUserID(&userIDs, seenUsers, comment.AuthorID)
		appendUniqueUserID(&userIDs, seenUsers, comment.ReplyToUserID)
	}
	authors, err := a.LoadAuthors(ctx, userIDs)
	if err != nil {
		return err
	}
	likeStatus := map[int64]bool{}
	if currentUserID != "" && currentUserID != "0" {
		likeStatus, _ = a.repo.GetCommentLikeStatus(ctx, currentUserID, commentIDs)
	}
	for _, comment := range flat {
		if comment == nil {
			continue
		}
		comment.Author = authors[comment.AuthorID]
		if comment.ReplyToUserID != "" && comment.ReplyToUserID != "0" {
			comment.ReplyToUser = authors[comment.ReplyToUserID]
		}
		comment.IsLiked = likeStatus[comment.ID]
	}
	return nil
}

func (a *CampusPostAssembler) FillPreviewReplies(ctx context.Context, comments []*CampusForumComment, currentUserID string) error {
	for _, comment := range comments {
		if comment == nil || comment.ID <= 0 || comment.ReplyCount <= 0 {
			continue
		}
		parentID := comment.ID
		replies, _, err := a.repo.ListComments(ctx, ListCampusCommentQuery{
			PostID:   comment.PostID,
			ParentID: &parentID,
			Statuses: []int32{CampusAuditStatusVisible},
			Offset:   0,
			Limit:    2,
		})
		if err != nil {
			return err
		}
		if err := a.HydrateComments(ctx, replies, currentUserID); err != nil {
			return err
		}
		comment.PreviewReplies = replies
	}
	return nil
}

func (a *CampusPostAssembler) HydrateFeedbackAuthors(ctx context.Context, feedbacks []*CampusFeedback) error {
	if len(feedbacks) == 0 {
		return nil
	}
	userIDs := make([]string, 0, len(feedbacks))
	seen := map[string]struct{}{}
	for _, feedback := range feedbacks {
		if feedback == nil {
			continue
		}
		appendUniqueUserID(&userIDs, seen, feedback.UserID)
	}
	authors, err := a.LoadAuthors(ctx, userIDs)
	if err != nil {
		return err
	}
	for _, feedback := range feedbacks {
		if feedback != nil {
			feedback.Author = authors[feedback.UserID]
		}
	}
	return nil
}

func (a *CampusPostAssembler) LoadAuthors(ctx context.Context, userIDs []string) (map[string]*CampusForumAuthor, error) {
	authors := make(map[string]*CampusForumAuthor, len(userIDs))
	if len(userIDs) == 0 || a.core == nil {
		return authors, nil
	}
	users, err := a.core.BatchGetUserBaseInfo(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		author := &CampusForumAuthor{
			UserID:   user.ID,
			Name:     firstNonEmpty(user.Nickname, user.Name, "同学"),
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		}
		if ok, profile, err := a.repo.GetProfileByUserID(ctx, user.ID); err == nil && ok {
			author.SchoolName = profile.SchoolName
			author.AuthStatus = profile.AuthStatus
		} else if err != nil {
			a.log.WithContext(ctx).Warnf("load campus author profile failed: user_id=%s err=%v", user.ID, err)
		}
		authors[user.ID] = author
	}
	return authors, nil
}

func appendUniqueUserID(userIDs *[]string, seen map[string]struct{}, userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == "0" {
		return
	}
	if _, ok := seen[userID]; ok {
		return
	}
	seen[userID] = struct{}{}
	*userIDs = append(*userIDs, userID)
}
