package server

import (
	"context"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/videoApi/service/internal/biz"
)

type CampusTaskServer struct {
	uc     *biz.CampusUsecase
	log    *log.Helper
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
}

func NewCampusTaskServer(uc *biz.CampusUsecase, logger log.Logger) *CampusTaskServer {
	return &CampusTaskServer{
		uc:   uc,
		log:  log.NewHelper(logger),
		done: make(chan struct{}),
	}
}

func (s *CampusTaskServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return nil
	}
	taskCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	go s.run(taskCtx)
	s.log.Info("启动校园后台任务")
	return nil
}

func (s *CampusTaskServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	select {
	case <-s.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := s.uc.StopCampusBatches(ctx); err != nil {
		s.log.Warnf("停止校园批处理失败: %v", err)
	}
	s.log.Info("停止校园后台任务")
	return nil
}

func (s *CampusTaskServer) run(ctx context.Context) {
	defer close(s.done)
	s.safeRefreshRecommendPool(ctx)
	s.safeProcessNotificationOutbox(ctx)
	recommendTicker := time.NewTicker(5 * time.Minute)
	reconcileTicker := time.NewTicker(1 * time.Hour)
	flushTicker := time.NewTicker(10 * time.Second)
	notificationTicker := time.NewTicker(2 * time.Second)
	defer recommendTicker.Stop()
	defer reconcileTicker.Stop()
	defer flushTicker.Stop()
	defer notificationTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-recommendTicker.C:
			s.safeRefreshRecommendPool(ctx)
		case <-reconcileTicker.C:
			s.safeReconcile(ctx)
		case <-flushTicker.C:
			if err := s.uc.FlushCampusBatches(ctx); err != nil {
				s.log.Warnf("flush campus batches failed: %v", err)
			}
		case <-notificationTicker.C:
			s.safeProcessNotificationOutbox(ctx)
		}
	}
}

func (s *CampusTaskServer) safeRefreshRecommendPool(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.uc.RefreshCampusRecommendPool(taskCtx); err != nil {
		s.log.Warnf("刷新校园推荐池失败: %v", err)
	}
}

func (s *CampusTaskServer) safeReconcile(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result, err := s.uc.RunCampusStatsReconcile(taskCtx)
	if err != nil {
		s.log.Warnf("校园计数对账失败: %v", err)
		return
	}
	s.log.Infof("校园计数对账完成: updated_posts=%d updated_comments=%d", result.UpdatedPosts, result.UpdatedComments)
}

func (s *CampusTaskServer) safeProcessNotificationOutbox(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.uc.ProcessPendingNotificationOutbox(taskCtx, 100); err != nil {
		s.log.Warnf("处理校园通知任务失败: %v", err)
	}
}
