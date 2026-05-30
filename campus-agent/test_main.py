import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))

from main import (
    TASK_TOOLS,
    AgentResult,
    ModerationAuditRequest,
    build_agent_context,
    fallback_result,
    heuristic_moderation,
    normalize_moderation_result,
    parse_model_json,
    plan_tools_node,
)


class CampusAgentTest(unittest.TestCase):
    def test_task_allowlist_is_bounded(self):
        self.assertIn("daily_ops", TASK_TOOLS)
        self.assertLessEqual(len(TASK_TOOLS["moderation_advice"]), 6)

    def test_parse_model_json_from_markdown(self):
        parsed = parse_model_json('```json\n{"summary":"ok","risk_level":"low"}\n```')
        self.assertEqual(parsed["summary"], "ok")

    def test_fallback_result_uses_tool_data(self):
        result = fallback_result(
            "daily_ops",
            "",
            [
                {"tool": "admin_summary", "ok": True, "data": {"data": {"summary": {"pending_reports": 1, "pending_posts": 2}}}},
                {"tool": "security_overview", "ok": True, "data": {"data": {"security": {"today_errors": 3}}}},
            ],
        )
        self.assertIsInstance(result, AgentResult)
        self.assertEqual(result.risk_level, "medium")
        self.assertTrue(result.findings)

    def test_graph_plans_allowed_tools(self):
        state = plan_tools_node({"run_id": "1", "run_type": "rag_gap", "question": "看看知识库"})
        self.assertIn("rag_bad_logs", state["tool_names"])

    def test_heuristic_moderation_passes_low_risk_post(self):
        result = heuristic_moderation(ModerationAuditRequest(title="食堂新品", content="二楼今天有新套餐，味道还不错"))
        self.assertEqual(result.decision, "pass")
        self.assertEqual(result.risk_level, "low")
        self.assertGreaterEqual(result.confidence, 0.85)

    def test_heuristic_moderation_reviews_high_risk_post(self):
        result = heuristic_moderation(ModerationAuditRequest(title="兼职", content="有人代考联系我"))
        self.assertEqual(result.decision, "review")
        self.assertEqual(result.risk_level, "high")
        self.assertIn("keyword:代考", result.evidence)

    def test_heuristic_moderation_uses_request_words(self):
        result = heuristic_moderation(ModerationAuditRequest(title="校园墙", content="这里有暗号甲", high_risk_words=["暗号甲"], review_words=["暗号乙"]))
        self.assertEqual(result.risk_level, "high")
        self.assertIn("keyword:暗号甲", result.evidence)

    def test_build_agent_context_extracts_top_items(self):
        context = build_agent_context([
            {"tool": "reports", "ok": True, "data": {"reports": [
                {"id": "1", "target_type": "post", "target_id": "9", "reason": "广告", "detail": "二维码引流", "reporter": {"id": "u1"}, "created_at": "now"},
                {"id": "2", "target_type": "comment", "target_id": "8", "reason": "辱骂"},
                {"id": "3", "target_type": "post", "target_id": "7", "reason": "挂人"},
                {"id": "4", "target_type": "post", "target_id": "6", "reason": "多余"},
            ]}},
            {"tool": "feedback", "ok": True, "data": {"feedback": [
                {"id": "5", "feedback_type": "bug", "content": "页面打不开", "contact": "wx", "images": ["a"], "created_at": "now"}
            ]}},
        ])
        self.assertEqual(len(context["reports"]), 3)
        self.assertEqual(context["reports"][0]["reporter_id"], "u1")
        self.assertTrue(context["feedback"][0]["contact_present"])

    def test_normalize_moderation_result_clamps_invalid_model_output(self):
        result = normalize_moderation_result({
            "decision": "delete",
            "confidence": 2,
            "risk_level": "urgent",
            "reason": "x" * 200,
            "evidence": "raw",
        })
        self.assertEqual(result.decision, "review")
        self.assertEqual(result.confidence, 1.0)
        self.assertEqual(result.risk_level, "medium")
        self.assertLessEqual(len(result.reason), 120)
        self.assertEqual(result.evidence, ["raw"])


if __name__ == "__main__":
    unittest.main()
