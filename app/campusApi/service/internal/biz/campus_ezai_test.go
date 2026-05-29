package biz

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeEzaiPersonaConfigFillsDefaultsAndClampsLength(t *testing.T) {
	got := normalizeEzaiPersonaConfig(&CampusEzaiPersonaConfig{
		Name:          strings.Repeat("e", 40),
		MaxReplyChars: 999,
	})
	if len([]rune(got.Name)) != 24 {
		t.Fatalf("name length = %d, want 24", len([]rune(got.Name)))
	}
	if got.Role == "" || got.FallbackReply == "" {
		t.Fatalf("defaults not filled: %#v", got)
	}
	if got.MaxReplyChars != 220 {
		t.Fatalf("max reply chars = %d, want 220", got.MaxReplyChars)
	}

	got = normalizeEzaiPersonaConfig(&CampusEzaiPersonaConfig{MaxReplyChars: 1})
	if got.MaxReplyChars != 60 {
		t.Fatalf("min reply chars = %d, want 60", got.MaxReplyChars)
	}
}

func TestBuildEzaiSystemPromptIncludesPersonaAndKnowledgeRule(t *testing.T) {
	persona := normalizeEzaiPersonaConfig(&CampusEzaiPersonaConfig{
		Name:          "测试e仔",
		Role:          "校园助手",
		MaxReplyChars: 120,
	})
	prompt := buildEzaiSystemPrompt(persona, true)
	for _, want := range []string{"测试e仔", "校园助手", "目前资料显示", "120 字以内"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q: %s", want, prompt)
		}
	}
}

func TestShouldUseEzaiNoKnowledgeReply(t *testing.T) {
	resp := &CampusRAGQueryResponse{NeedKnowledge: true, Confidence: 0.1}
	if !shouldUseEzaiNoKnowledgeReply(resp, "", nil) {
		t.Fatalf("should use no knowledge reply when knowledge is needed but no context")
	}
	if shouldUseEzaiNoKnowledgeReply(resp, "资料命中", nil) {
		t.Fatalf("should not use no knowledge reply when context exists")
	}
	if shouldUseEzaiNoKnowledgeReply(resp, "", errors.New("rag down")) {
		t.Fatalf("should not use no knowledge reply when rag failed")
	}
}
