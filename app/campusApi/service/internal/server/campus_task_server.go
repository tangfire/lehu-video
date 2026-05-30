package server

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"lehu-video/app/campusApi/service/internal/biz"
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
	s.safeProcessAIReplyTasks(ctx)
	s.safeProcessAIContentAuditTasks(ctx)
	s.safeCleanupAccessLogs(ctx)
	var dailyReportTimer *time.Timer
	if campusAgentDailyReportEnabled() {
		dailyReportTimer = time.NewTimer(durationUntilNextDailyReport(time.Now()))
		defer dailyReportTimer.Stop()
	}
	recommendTicker := time.NewTicker(5 * time.Minute)
	reconcileTicker := time.NewTicker(1 * time.Hour)
	flushTicker := time.NewTicker(10 * time.Second)
	notificationTicker := time.NewTicker(2 * time.Second)
	aiReplyTicker := time.NewTicker(5 * time.Second)
	aiAuditTicker := time.NewTicker(5 * time.Second)
	defer recommendTicker.Stop()
	defer reconcileTicker.Stop()
	defer flushTicker.Stop()
	defer notificationTicker.Stop()
	defer aiReplyTicker.Stop()
	defer aiAuditTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-recommendTicker.C:
			s.safeRefreshRecommendPool(ctx)
		case <-reconcileTicker.C:
			s.safeReconcile(ctx)
			s.safeCleanupAccessLogs(ctx)
		case <-flushTicker.C:
			if err := s.uc.FlushCampusBatches(ctx); err != nil {
				s.log.Warnf("flush campus batches failed: %v", err)
			}
		case <-notificationTicker.C:
			s.safeProcessNotificationOutbox(ctx)
		case <-aiReplyTicker.C:
			s.safeProcessAIReplyTasks(ctx)
		case <-aiAuditTicker.C:
			s.safeProcessAIContentAuditTasks(ctx)
		case <-dailyReportTimerC(dailyReportTimer):
			s.safeRunDailyCopilotReport(ctx)
			if dailyReportTimer != nil {
				dailyReportTimer.Reset(durationUntilNextDailyReport(time.Now()))
			}
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

func (s *CampusTaskServer) safeCleanupAccessLogs(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	deleted, err := s.uc.CleanupExpiredAccessLogs(taskCtx)
	if err != nil {
		s.log.Warnf("清理校园访问日志失败: %v", err)
		return
	}
	if deleted > 0 {
		s.log.Infof("清理校园访问日志完成: deleted=%d", deleted)
	}
}

func (s *CampusTaskServer) safeProcessNotificationOutbox(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.uc.ProcessPendingNotificationOutbox(taskCtx, 100); err != nil {
		s.log.Warnf("处理校园通知任务失败: %v", err)
	}
}

func (s *CampusTaskServer) safeProcessAIReplyTasks(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := s.uc.ProcessPendingAIReplyTasks(taskCtx, 10); err != nil {
		s.log.Warnf("处理 e仔 AI 回复任务失败: %v", err)
	}
}

func (s *CampusTaskServer) safeProcessAIContentAuditTasks(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := s.uc.ProcessPendingAIContentAuditTasks(taskCtx, 10); err != nil {
		s.log.Warnf("处理校园 AI 内容审核任务失败: %v", err)
	}
}

func (s *CampusTaskServer) safeRunDailyCopilotReport(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	run, err := s.uc.CreateScheduledAgentRun(taskCtx, "daily_ops", "请生成今天校园 e站运营巡检日报，重点关注审核积压、e仔/RAG质量和安全异常。")
	if err != nil {
		s.log.Warnf("生成 Copilot 每日巡检失败: %v", err)
		return
	}
	if run != nil {
		s.log.Infof("Copilot 每日巡检完成: run_id=%d risk=%s feishu=%s", run.ID, run.RiskLevel, run.FeishuStatus)
	}
}

func dailyReportTimerC(timer *time.Timer) <-chan time.Time {
	if timer == nil {
		return nil
	}
	return timer.C
}

func campusAgentDailyReportEnabled() bool {
	return !envBoolFalseServer(os.Getenv("CAMPUS_AGENT_DAILY_REPORT_ENABLED"))
}

func durationUntilNextDailyReport(now time.Time) time.Duration {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	localNow := now.In(loc)
	hour, minute := parseDailyReportTime(os.Getenv("CAMPUS_AGENT_DAILY_REPORT_TIME"))
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, loc)
	if !next.After(localNow) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(localNow)
}

func parseDailyReportTime(value string) (int, int) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "09:30"
	}
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 9, 30
	}
	hour, errHour := strconv.Atoi(strings.TrimSpace(parts[0]))
	minute, errMinute := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errHour != nil || errMinute != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 9, 30
	}
	return hour, minute
}

func envBoolFalseServer(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false", "off", "no", "disabled":
		return true
	default:
		return false
	}
}
