package biz

import (
	"strings"
	"testing"
)

func TestNormalizeAuditWordsSplitsDedupesAndFallsBack(t *testing.T) {
	got := normalizeAuditWords("暗号甲，暗号乙\n暗号甲  "+strings.Repeat("长", 40), []string{"默认"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %#v", len(got), got)
	}
	if got[0] != "暗号甲" || got[1] != "暗号乙" {
		t.Fatalf("words = %#v", got)
	}
	if len([]rune(got[2])) != 32 {
		t.Fatalf("long word length = %d, want 32", len([]rune(got[2])))
	}

	fallback := normalizeAuditWords("，， ", []string{"默认"})
	if len(fallback) != 1 || fallback[0] != "默认" {
		t.Fatalf("fallback = %#v, want 默认", fallback)
	}
}

func TestClassifyCampusPostByWordsUsesConfiguredWords(t *testing.T) {
	got := classifyCampusPostByWords("校园墙", "这里有暗号甲", []string{"暗号甲"}, []string{"暗号乙"})
	if got.RiskLevel != "high" || got.Decision != CampusAIContentAuditDecisionReview {
		t.Fatalf("high risk result = %#v", got)
	}

	got = classifyCampusPostByWords("校园墙", "这里有暗号乙", []string{"暗号甲"}, []string{"暗号乙"})
	if got.RiskLevel != "medium" {
		t.Fatalf("review result = %#v", got)
	}

	got = classifyCampusPostByWords("食堂新品", "二楼套餐味道还不错", []string{"暗号甲"}, []string{"暗号乙"})
	if got.RiskLevel != "low" || got.Decision != CampusAIContentAuditDecisionPass {
		t.Fatalf("low risk result = %#v", got)
	}
}
