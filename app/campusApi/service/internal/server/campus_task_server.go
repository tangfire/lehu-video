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
	uc      *biz.CampusUsecase
	log     *log.Helper
	cancel  context.CancelFunc
	done    chan struct{}
	mu      sync.Mutex
	wg      sync.WaitGroup
	running map[string]bool
}

func NewCampusTaskServer(uc *biz.CampusUsecase, logger log.Logger) *CampusTaskServer {
	return &CampusTaskServer{
		uc:      uc,
		log:     log.NewHelper(logger),
		done:    make(chan struct{}),
		running: map[string]bool{},
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
	workersDone := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(workersDone)
	}()
	select {
	case <-workersDone:
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
	s.runExclusive(ctx, "recommend_pool", s.safeRefreshRecommendPool)
	s.runExclusive(ctx, "notification_outbox", s.safeProcessNotificationOutbox)
	s.runExclusive(ctx, "ops_alerts", s.safeProcessOpsAlerts)
	s.runExclusive(ctx, "ops_sla_alerts", s.safeProcessOpsSLAAlerts)
	s.runExclusive(ctx, "ai_replies", s.safeProcessAIReplyTasks)
	s.runExclusive(ctx, "ai_audit", s.safeProcessAIContentAuditTasks)
	s.runExclusive(ctx, "access_log_cleanup", s.safeCleanupAccessLogs)
	s.runExclusive(ctx, "rag_eval_drafts", s.safeSeedRAGEvalDrafts)
	var dailyReportTimer *time.Timer
	if campusAgentDailyReportEnabled() {
		dailyReportTimer = time.NewTimer(durationUntilNextDailyReport(time.Now()))
		defer dailyReportTimer.Stop()
	}
	recommendTicker := time.NewTicker(5 * time.Minute)
	reconcileTicker := time.NewTicker(1 * time.Hour)
	flushTicker := time.NewTicker(10 * time.Second)
	notificationTicker := time.NewTicker(2 * time.Second)
	opsAlertTicker := time.NewTicker(5 * time.Second)
	opsSLATicker := time.NewTicker(5 * time.Minute)
	aiReplyTicker := time.NewTicker(5 * time.Second)
	aiAuditTicker := time.NewTicker(5 * time.Second)
	defer recommendTicker.Stop()
	defer reconcileTicker.Stop()
	defer flushTicker.Stop()
	defer notificationTicker.Stop()
	defer opsAlertTicker.Stop()
	defer opsSLATicker.Stop()
	defer aiReplyTicker.Stop()
	defer aiAuditTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-recommendTicker.C:
			s.runExclusive(ctx, "recommend_pool", s.safeRefreshRecommendPool)
		case <-reconcileTicker.C:
			s.runExclusive(ctx, "stats_reconcile", s.safeReconcile)
			s.runExclusive(ctx, "access_log_cleanup", s.safeCleanupAccessLogs)
			s.runExclusive(ctx, "rag_eval_drafts", s.safeSeedRAGEvalDrafts)
		case <-flushTicker.C:
			if err := s.uc.FlushCampusBatches(ctx); err != nil {
				s.log.Warnf("flush campus batches failed: %v", err)
			}
		case <-notificationTicker.C:
			s.runExclusive(ctx, "notification_outbox", s.safeProcessNotificationOutbox)
		case <-opsAlertTicker.C:
			s.runExclusive(ctx, "ops_alerts", s.safeProcessOpsAlerts)
		case <-opsSLATicker.C:
			s.runExclusive(ctx, "ops_sla_alerts", s.safeProcessOpsSLAAlerts)
		case <-aiReplyTicker.C:
			s.runExclusive(ctx, "ai_replies", s.safeProcessAIReplyTasks)
		case <-aiAuditTicker.C:
			s.runExclusive(ctx, "ai_audit", s.safeProcessAIContentAuditTasks)
		case <-dailyReportTimerC(dailyReportTimer):
			s.runExclusive(ctx, "daily_agent_report", s.safeRunDailyAgentReport)
			if dailyReportTimer != nil {
				dailyReportTimer.Reset(durationUntilNextDailyReport(time.Now()))
			}
		}
	}
}

func (s *CampusTaskServer) runExclusive(ctx context.Context, name string, fn func(context.Context)) {
	s.mu.Lock()
	if s.running[name] {
		s.mu.Unlock()
		return
	}
	s.running[name] = true
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.running, name)
			s.mu.Unlock()
		}()
		fn(ctx)
	}()
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

func (s *CampusTaskServer) safeProcessOpsAlerts(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := s.uc.ProcessPendingOpsAlerts(taskCtx, 20); err != nil {
		s.log.Warnf("处理校园运营值班提醒失败: %v", err)
	}
}

func (s *CampusTaskServer) safeProcessOpsSLAAlerts(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.uc.ProcessOpsSLAAlerts(taskCtx); err != nil {
		s.log.Warnf("处理校园运营 SLA 提醒失败: %v", err)
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
	perTaskTimeout := envDurationServer("CAMPUS_AI_AUDIT_TASK_TIMEOUT", 10*time.Second)
	limit := envIntServer("CAMPUS_AI_AUDIT_BATCH_SIZE", 2)
	if limit < 1 {
		limit = 1
	}
	if limit > 3 {
		limit = 3
	}
	taskCtx, cancel := context.WithTimeout(ctx, time.Duration(limit)*perTaskTimeout+2*time.Second)
	defer cancel()
	if err := s.uc.ProcessPendingAIContentAuditTasks(taskCtx, limit); err != nil {
		s.log.Warnf("处理校园 AI 内容审核任务失败: %v", err)
	}
}

func (s *CampusTaskServer) safeRunDailyAgentReport(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	run, err := s.uc.CreateScheduledAgentRun(taskCtx, "daily_ops", "请生成今天校园 e站运营巡检日报，重点关注审核积压、e仔/RAG质量和安全异常。")
	if err != nil {
		s.log.Warnf("生成值班 Agent 每日巡检失败: %v", err)
		return
	}
	if run != nil {
		s.log.Infof("值班 Agent 每日巡检完成: run_id=%d risk=%s feishu=%s", run.ID, run.RiskLevel, run.FeishuStatus)
	}
}

func (s *CampusTaskServer) safeSeedRAGEvalDrafts(ctx context.Context) {
	taskCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	created, err := s.uc.SeedRAGEvalDraftsFromLogs(taskCtx, envIntServer("CAMPUS_RAG_EVAL_DRAFT_BATCH_SIZE", 30))
	if err != nil {
		s.log.Warnf("沉淀 RAG 评测草稿失败: %v", err)
		return
	}
	if created > 0 {
		s.log.Infof("沉淀 RAG 评测草稿完成: created=%d", created)
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

func envIntServer(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDurationServer(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
