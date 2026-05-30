import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))

from main import TASK_TOOLS, AgentResult, fallback_result, parse_model_json, plan_tools_node


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


if __name__ == "__main__":
    unittest.main()
