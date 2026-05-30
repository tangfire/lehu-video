package biz

import (
	"strings"
	"testing"
)

func TestOpsAlertFeishuPayloadIncludesReportTargetSummary(t *testing.T) {
	uc := &CampusUsecase{}
	payload := uc.opsAlertFeishuPayload(&CampusOpsAlert{
		ID:         9,
		AlertType:  CampusOpsAlertTypeReportCreated,
		Priority:   CampusOpsAlertPriorityHigh,
		TargetType: "report",
		TargetID:   8,
		Title:      "校园 e站收到新举报",
		Summary:    "张三 举报了评论「不友好评论」：辱骂",
		Payload: map[string]interface{}{
			"target_type":     "comment",
			"target_id":       "123",
			"comment_excerpt": "不友好评论",
			"post_id":         "456",
			"post_title":      "食堂讨论",
			"reporter_id":     "7",
			"reporter_name":   "张三",
			"reason":          "辱骂",
			"detail":          "评论里有人身攻击",
			"admin_path":      "/admin/moderation?tab=reports&status=0",
		},
	})
	findings, ok := payload["findings"].([]map[string]interface{})
	if !ok || len(findings) < 4 {
		t.Fatalf("findings = %#v", payload["findings"])
	}
	rendered := ""
	for _, item := range findings {
		rendered += strings.TrimSpace(opsMapString(item, "title")+" "+opsMapString(item, "detail")) + "\n"
	}
	for _, want := range []string{"被举报评论 123", "不友好评论", "举报原因：辱骂", "张三（7）"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("findings missing %q: %s", want, rendered)
		}
	}
}

func TestRenderPrometheusMetricsEscapesAndSortsLabels(t *testing.T) {
	text := renderPrometheusMetrics([]CampusMetricSeries{{
		Name:  "campus_ops_alerts",
		Value: 2,
		Labels: map[string]string{
			"status":     "pending",
			"alert_type": "report\ncreated",
		},
	}})
	if !strings.Contains(text, "# TYPE campus_ops_alerts gauge") {
		t.Fatalf("missing type header: %s", text)
	}
	if !strings.Contains(text, `campus_ops_alerts{alert_type="report\ncreated",status="pending"} 2`) {
		t.Fatalf("unexpected metric text: %s", text)
	}
}
