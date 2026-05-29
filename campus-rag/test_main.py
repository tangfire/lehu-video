from datetime import datetime, timezone
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))

from main import (
    NO_EXPIRY_MS,
    QueryRequest,
    need_knowledge,
    parse_time_ms,
    payload_is_effective,
    search_text,
)


class CampusRAGLogicTest(unittest.TestCase):
    def test_need_knowledge_covers_campus_question_variants(self):
        self.assertTrue(need_knowledge("宿舍要带被子吗"))
        self.assertTrue(need_knowledge("这个在哪里办"))
        self.assertTrue(need_knowledge("能不能申请走读"))
        self.assertFalse(need_knowledge("谢谢 e仔"))

    def test_search_text_includes_post_context(self):
        expanded = search_text("这个在哪里办", "标题：校园卡办理\n正文：新生校园卡领取地点说明")
        self.assertIn("这个在哪里办", expanded)
        self.assertIn("校园卡办理", expanded)

    def test_parse_time_ms_and_effective_payload(self):
        value = parse_time_ms("2026-05-29T00:00:00Z", 0)
        expected = int(datetime(2026, 5, 29, tzinfo=timezone.utc).timestamp() * 1000)
        self.assertEqual(value, expected)
        self.assertEqual(parse_time_ms("", NO_EXPIRY_MS), NO_EXPIRY_MS)
        self.assertTrue(payload_is_effective({"effective_at_ms": 0, "expired_at_ms": NO_EXPIRY_MS}))

    def test_query_request_accepts_context(self):
        req = QueryRequest(query="这个可以吗", context="标题：宿舍床帘规则")
        self.assertTrue(req.context)


if __name__ == "__main__":
    unittest.main()
