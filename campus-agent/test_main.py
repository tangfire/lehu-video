import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))

from main import (
    TASK_TOOLS,
    AgentResult,
    ModerationAuditRequest,
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
