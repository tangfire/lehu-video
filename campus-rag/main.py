import io
import os
import re
import threading
import time
import uuid
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple
from urllib.parse import urlparse, urlunparse

import jieba
import requests
from docx import Document as DocxDocument
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from pypdf import PdfReader
from qdrant_client import QdrantClient
from qdrant_client.http import models
from rank_bm25 import BM25Okapi


QDRANT_URL = os.getenv("QDRANT_URL", "http://qdrant:6333")
QDRANT_COLLECTION = os.getenv("QDRANT_COLLECTION", "campus_knowledge")
SILICONFLOW_API_KEY = os.getenv("SILICONFLOW_API_KEY", "")
SILICONFLOW_BASE_URL = os.getenv("SILICONFLOW_BASE_URL", "https://api.siliconflow.cn/v1").rstrip("/")
EMBEDDING_MODEL = os.getenv("CAMPUS_RAG_EMBEDDING_MODEL", "BAAI/bge-m3")
HOST_REWRITE = os.getenv("MINIO_PUBLIC_HOST_REWRITE", "localhost:19000=minio:9000")
CHUNK_SIZE = int(os.getenv("CAMPUS_RAG_CHUNK_SIZE", "800"))
CHUNK_OVERLAP = int(os.getenv("CAMPUS_RAG_CHUNK_OVERLAP", "120"))
QUERY_TIMEOUT = float(os.getenv("CAMPUS_RAG_HTTP_TIMEOUT", "20"))
MIN_CHUNK_CONFIDENCE = float(os.getenv("CAMPUS_RAG_MIN_CHUNK_CONFIDENCE", "0.48"))
BM25_CACHE_TTL = float(os.getenv("CAMPUS_RAG_BM25_CACHE_TTL", "60"))
BM25_MAX_POINTS = int(os.getenv("CAMPUS_RAG_BM25_MAX_POINTS", "5000"))
NO_EXPIRY_MS = 4102444800000  # 2100-01-01T00:00:00Z

app = FastAPI(title="campus-rag", version="1.0.0")
qdrant = QdrantClient(url=QDRANT_URL, timeout=QUERY_TIMEOUT)


class IndexRequest(BaseModel):
    document_id: int
    title: str
    category: str = "general"
    source: str = ""
    file_url: str = ""
    file_type: str = ""
    content: str = ""
    effective_at: str = ""
    expired_at: str = ""
    metadata: Dict[str, str] = Field(default_factory=dict)


class DeleteRequest(BaseModel):
    document_id: int


class QueryRequest(BaseModel):
    query: str
    top_k: int = 5
    categories: List[str] = Field(default_factory=list)
    context: str = ""


bm25_cache_lock = threading.Lock()
bm25_cache_version = 0
bm25_cache: Dict[str, Any] = {
    "version": -1,
    "built_at": 0.0,
    "points": [],
    "corpus": [],
    "bm25": None,
}


def now_ms() -> int:
    return int(time.time() * 1000)


def parse_time_ms(value: str, fallback: int) -> int:
    raw = str(value or "").strip()
    if not raw:
        return fallback
    if re.fullmatch(r"\d{10,13}", raw):
        parsed = int(raw)
        return parsed if parsed > 10_000_000_000 else parsed * 1000
    normalized = raw.replace("Z", "+00:00")
    for layout in ("%Y-%m-%d %H:%M:%S", "%Y-%m-%d %H:%M", "%Y-%m-%d"):
        try:
            parsed = datetime.strptime(raw, layout).replace(tzinfo=timezone.utc)
            return int(parsed.timestamp() * 1000)
        except ValueError:
            pass
    try:
        parsed = datetime.fromisoformat(normalized)
        if parsed.tzinfo is None:
            parsed = parsed.replace(tzinfo=timezone.utc)
        return int(parsed.timestamp() * 1000)
    except ValueError:
        return fallback


def request_time_ms(req: IndexRequest, field: str, fallback: int) -> int:
    value = getattr(req, field, "") or ""
    if not value and req.metadata:
        value = req.metadata.get(field, "")
    return parse_time_ms(value, fallback)


def rewrite_url(raw_url: str) -> str:
    value = raw_url.strip()
    if not value or "=" not in HOST_REWRITE:
        return value
    source, target = HOST_REWRITE.split("=", 1)
    parsed = urlparse(value)
    if parsed.netloc == source:
        return urlunparse(parsed._replace(netloc=target))
    return value


def normalize_text(text: str) -> str:
    text = re.sub(r"\r\n?", "\n", text or "")
    text = re.sub(r"[ \t]+", " ", text)
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text.strip()


def parse_file(file_url: str, file_type: str) -> str:
    url = rewrite_url(file_url)
    resp = requests.get(url, timeout=QUERY_TIMEOUT)
    if resp.status_code >= 400:
        raise HTTPException(status_code=502, detail=f"download document failed: {resp.status_code}")
    raw = resp.content
    kind = (file_type or "").lower().strip(".")
    if kind == "pdf":
        reader = PdfReader(io.BytesIO(raw))
        return normalize_text("\n".join(page.extract_text() or "" for page in reader.pages))
    if kind == "docx":
        doc = DocxDocument(io.BytesIO(raw))
        return normalize_text("\n".join(p.text for p in doc.paragraphs))
    if kind in ("txt", "md", "markdown"):
        return normalize_text(raw.decode("utf-8", errors="ignore"))
    raise HTTPException(status_code=400, detail="unsupported file_type")


def is_heading(line: str) -> bool:
    value = line.strip()
    if not value or len(value) > 64:
        return False
    if re.match(r"^(#{1,6}\s*)?([一二三四五六七八九十]+[、.．]|第[一二三四五六七八九十0-9]+[章节条]|[0-9]+[、.．)]|[（(][一二三四五六七八九十0-9]+[）)])", value):
        return True
    if re.match(r"^【[^】]{2,30}】$", value):
        return True
    heading_words = ("须知", "指南", "流程", "安排", "说明", "时间", "地点", "材料", "路线", "FAQ", "问答", "政策", "规则", "入口")
    if any(value.endswith(word) for word in heading_words):
        return True
    return False


def split_long_block(text: str, limit: int) -> List[str]:
    text = text.strip()
    if len(text) <= limit:
        return [text] if text else []
    sentences = [item.strip() for item in re.split(r"(?<=[。！？!?；;])", text) if item.strip()]
    if len(sentences) <= 1:
        chunks: List[str] = []
        start = 0
        while start < len(text):
            end = min(len(text), start + limit)
            chunk = text[start:end].strip()
            if chunk:
                chunks.append(chunk)
            if end >= len(text):
                break
            start = max(end - CHUNK_OVERLAP, start + 1)
        return chunks
    chunks = []
    current = ""
    for sentence in sentences:
        if current and len(current) + len(sentence) + 1 > limit:
            chunks.append(current.strip())
            overlap = current[-CHUNK_OVERLAP:].strip() if CHUNK_OVERLAP > 0 else ""
            current = overlap + ("\n" if overlap else "") + sentence
        else:
            current = (current + sentence) if current else sentence
    if current.strip():
        chunks.append(current.strip())
    return chunks


def build_text_blocks(text: str) -> List[str]:
    lines = [line.strip() for line in normalize_text(text).split("\n")]
    blocks: List[str] = []
    current: List[str] = []
    section_title = ""

    def flush_current() -> None:
        nonlocal current
        if not current:
            return
        block = "\n".join(current).strip()
        if block:
            blocks.append(block)
        current = []

    for line in lines:
        if not line:
            flush_current()
            continue
        if is_heading(line):
            flush_current()
            section_title = line
            current = [line]
            continue
        if section_title and not current:
            current.append(section_title)
        current.append(line)
    flush_current()
    return blocks


def chunk_text(text: str) -> List[str]:
    text = normalize_text(text)
    if not text:
        return []
    blocks = build_text_blocks(text)
    if not blocks:
        return []
    chunks: List[str] = []
    current = ""
    for block in blocks:
        if len(block) > CHUNK_SIZE:
            if current.strip():
                chunks.append(current.strip())
                current = ""
            chunks.extend(split_long_block(block, CHUNK_SIZE))
            continue
        candidate = f"{current}\n\n{block}".strip() if current else block
        if len(candidate) <= CHUNK_SIZE:
            current = candidate
            continue
        if current.strip():
            chunks.append(current.strip())
        current = block
    if current.strip():
        chunks.append(current.strip())
    return chunks


def tokenize(text: str) -> List[str]:
    words = [w.strip().lower() for w in jieba.cut(text or "") if w.strip()]
    grams = re.findall(r"[a-zA-Z0-9_]+", text or "")
    return words + [g.lower() for g in grams]


def compact_query_context(text: str) -> str:
    value = normalize_text(text)
    if not value:
        return ""
    value = re.sub(r"(标题|正文|版块|类型|图片|视频)：", " ", value)
    value = re.sub(r"\s+", " ", value)
    return value[:600]


def search_text(query: str, context: str = "") -> str:
    parts = [normalize_text(query)]
    ctx = compact_query_context(context)
    if ctx:
        parts.append(ctx)
    return "\n".join(part for part in parts if part).strip()


def need_knowledge(query: str) -> bool:
    q = query.strip()
    if len(q) <= 4:
        return False
    casual_patterns = ["谢谢", "感谢", "哈哈", "你好", "在吗", "收到", "好的", "没事", "辛苦"]
    if len(q) <= 12 and any(item == q or item in q for item in casual_patterns):
        return False
    if len(q) <= 8 and q in {"可以吗", "行不行", "对吗", "真的吗", "咋办", "怎么说"}:
        return True
    keywords = [
        "报到", "宿舍", "校区", "校园网", "军训", "快递", "交通", "路线", "教务", "课表",
        "选课", "学费", "缴费", "深圳职业技术大学", "深汕", "社团", "新生", "学院", "通知",
        "什么时候", "在哪里", "怎么去", "怎么办", "要求", "规定", "政策",
        "食堂", "饭堂", "餐厅", "校车", "公交", "地铁", "图书馆", "医保", "银行卡", "校园卡",
        "一卡通", "宿舍电费", "电费", "水电", "门禁", "洗衣", "热水", "饮水", "空调", "宽带",
        "体检", "体测", "入学教育", "辅导员", "班级群", "快递点", "取件", "打印", "复印",
        "奖学金", "助学金", "贷款", "请假", "假条", "校历", "考试", "成绩", "补考",
        "寝室", "几人间", "床位", "床帘", "被子", "行李", "材料", "证件", "录取通知书", "身份证",
        "户口", "档案", "团组织", "党组织", "转接", "照片", "寸照", "报销", "充值", "缴费入口",
        "澡堂", "浴室", "插座", "断电", "熄灯", "门禁时间", "自习室", "实验室", "教学楼",
        "在哪里办", "去哪办", "去哪儿办", "能不能", "可不可以", "要不要", "要带", "带什么",
        "准备什么", "怎么申请", "怎么绑定", "怎么开通", "怎么预约", "截止", "开学", "放假",
    ]
    if any(k in q for k in keywords):
        return True
    question_markers = ("吗", "么", "嘛", "？", "?", "怎么", "咋", "哪里", "哪儿", "几点", "多久", "多少")
    campus_markers = (
        "校", "院", "宿", "课", "费", "证", "卡", "网", "餐", "饭", "车", "楼", "寝", "办",
        "带", "交", "缴", "群", "表", "水", "电", "假", "考", "训", "快递",
    )
    if any(marker in q for marker in question_markers) and any(marker in q for marker in campus_markers):
        return True
    if re.search(r"(要|能|可不可以|能不能|需要).{0,8}(带|交|办|申请|准备|缴|预约|绑定)", q):
        return True
    return False


def embed_texts(texts: List[str]) -> List[List[float]]:
    if not SILICONFLOW_API_KEY:
        raise HTTPException(status_code=503, detail="SILICONFLOW_API_KEY is not configured")
    resp = requests.post(
        f"{SILICONFLOW_BASE_URL}/embeddings",
        headers={"Authorization": f"Bearer {SILICONFLOW_API_KEY}", "Content-Type": "application/json"},
        json={"model": EMBEDDING_MODEL, "input": texts},
        timeout=QUERY_TIMEOUT,
    )
    if resp.status_code >= 400:
        raise HTTPException(status_code=502, detail=f"embedding failed: {resp.status_code} {resp.text[:200]}")
    data = resp.json().get("data") or []
    vectors = [item.get("embedding") for item in data if item.get("embedding")]
    if len(vectors) != len(texts):
        raise HTTPException(status_code=502, detail="embedding response size mismatch")
    return vectors


def ensure_collection(vector_size: int) -> None:
    collections = qdrant.get_collections().collections
    if any(item.name == QDRANT_COLLECTION for item in collections):
        return
    qdrant.create_collection(
        collection_name=QDRANT_COLLECTION,
        vectors_config=models.VectorParams(size=vector_size, distance=models.Distance.COSINE),
    )


def point_id(document_id: int, chunk_index: int) -> str:
    return str(uuid.uuid5(uuid.NAMESPACE_URL, f"campus-knowledge:{document_id}:{chunk_index}"))


def collection_exists() -> bool:
    collections = qdrant.get_collections().collections
    return any(item.name == QDRANT_COLLECTION for item in collections)


def invalidate_bm25_cache() -> None:
    global bm25_cache_version
    with bm25_cache_lock:
        bm25_cache_version += 1
        bm25_cache.update({"version": -1, "built_at": 0.0, "points": [], "corpus": [], "bm25": None})


def active_filter(categories: Optional[List[str]] = None) -> models.Filter:
    conditions = [
        models.FieldCondition(key="status", match=models.MatchValue(value="active")),
    ]
    if categories:
        conditions.append(models.FieldCondition(key="category", match=models.MatchAny(any=categories)))
    return models.Filter(must=conditions)


def payload_is_effective(payload: Dict[str, Any]) -> bool:
    now = now_ms()
    effective_at = int(payload.get("effective_at_ms") or 0)
    expired_at = int(payload.get("expired_at_ms") or NO_EXPIRY_MS)
    return effective_at <= now < expired_at


def payload_to_chunk(point: Any, score: float = 0.0, score_key: str = "_dense_score") -> Dict[str, Any]:
    payload = point.payload or {}
    item = {
        "chunk_id": str(payload.get("chunk_id") or point.id),
        "document_id": str(payload.get("document_id") or ""),
        "title": payload.get("title") or "",
        "category": payload.get("category") or "general",
        "content": payload.get("content") or "",
        "source": payload.get("source") or "",
        "score": round(float(score), 4),
    }
    item[score_key] = float(score or 0)
    return item


def scroll_active(categories: Optional[List[str]] = None, limit: int = 1000) -> List[Any]:
    if not collection_exists():
        return []
    points, _ = qdrant.scroll(
        collection_name=QDRANT_COLLECTION,
        scroll_filter=active_filter(categories),
        limit=limit,
        with_payload=True,
        with_vectors=False,
    )
    return [point for point in points if payload_is_effective(point.payload or {})]


def cached_bm25_index() -> Tuple[List[Any], List[List[str]], Optional[BM25Okapi]]:
    if not collection_exists():
        return [], [], None
    now = time.time()
    with bm25_cache_lock:
        if (
            bm25_cache["bm25"] is not None
            and bm25_cache["version"] == bm25_cache_version
            and now - float(bm25_cache["built_at"]) <= BM25_CACHE_TTL
        ):
            return bm25_cache["points"], bm25_cache["corpus"], bm25_cache["bm25"]
    points = scroll_active(None, limit=BM25_MAX_POINTS)
    corpus = [tokenize((p.payload or {}).get("content", "")) for p in points]
    bm25 = BM25Okapi(corpus) if points and any(corpus) else None
    with bm25_cache_lock:
        bm25_cache.update({
            "version": bm25_cache_version,
            "built_at": now,
            "points": points,
            "corpus": corpus,
            "bm25": bm25,
        })
    return points, corpus, bm25


def meaningful_terms(text: str) -> set:
    stopwords = {
        "的", "了", "呢", "吗", "啊", "呀", "和", "与", "以及", "一个", "一下", "怎么", "什么", "哪里",
        "什么时候", "怎么办", "可以", "需要", "有没有", "是不是", "我们", "你们", "学校", "校园",
    }
    terms = set()
    for word in tokenize(text):
        value = word.strip().lower()
        if not value or value in stopwords:
            continue
        if len(value) == 1 and not re.match(r"[a-zA-Z0-9]", value):
            continue
        terms.add(value)
    return terms


def overlap_score(query: str, content: str) -> float:
    query_terms = meaningful_terms(query)
    if not query_terms:
        return 0.0
    content_terms = meaningful_terms(content)
    if not content_terms:
        return 0.0
    matched = len(query_terms & content_terms)
    return min(1.0, matched / max(1, min(len(query_terms), 5)))


def clamp01(value: float) -> float:
    return max(0.0, min(1.0, float(value or 0)))


def chunk_confidence(item: Dict[str, Any], query: str) -> float:
    dense_score = clamp01(item.get("_dense_score", 0.0))
    sparse_score = clamp01(item.get("_sparse_score", 0.0))
    lexical_score = overlap_score(query, item.get("content") or "")
    if dense_score and sparse_score:
        confidence = dense_score * 0.58 + sparse_score * 0.25 + lexical_score * 0.17
    elif dense_score:
        confidence = dense_score * 0.72 + lexical_score * 0.18
    elif sparse_score:
        confidence = sparse_score * 0.65 + lexical_score * 0.2
    else:
        confidence = lexical_score * 0.4
    if lexical_score == 0 and sparse_score == 0:
        confidence *= 0.78
    return clamp01(confidence)


def rrf_fuse(dense: List[Dict[str, Any]], sparse: List[Dict[str, Any]], top_k: int, query_text: str) -> List[Dict[str, Any]]:
    scores: Dict[str, float] = {}
    payloads: Dict[str, Dict[str, Any]] = {}
    for rank, item in enumerate(dense):
        key = item["chunk_id"]
        scores[key] = scores.get(key, 0.0) + 1.0 / (60 + rank + 1)
        payloads[key] = {**payloads.get(key, {}), **item}
    for rank, item in enumerate(sparse):
        key = item["chunk_id"]
        scores[key] = scores.get(key, 0.0) + 1.0 / (60 + rank + 1)
        payloads[key] = {**payloads.get(key, {}), **item}
    fused = sorted(scores.items(), key=lambda kv: kv[1], reverse=True)
    if not fused:
        return []
    out = []
    for key, score in fused[:top_k]:
        item = dict(payloads[key])
        item["_rrf_score"] = score
        confidence = chunk_confidence(item, query_text)
        if confidence < MIN_CHUNK_CONFIDENCE:
            continue
        item["score"] = round(confidence, 4)
        item.pop("_dense_score", None)
        item.pop("_sparse_score", None)
        item.pop("_rrf_score", None)
        out.append(item)
    return out


@app.get("/healthz")
def healthz() -> Dict[str, Any]:
    status = "ok"
    qdrant_status = "ok"
    last_error = ""
    chunk_count = 0
    try:
        collections = qdrant.get_collections().collections
        if any(item.name == QDRANT_COLLECTION for item in collections):
            info = qdrant.count(
                collection_name=QDRANT_COLLECTION,
                count_filter=models.Filter(
                    must=[models.FieldCondition(key="status", match=models.MatchValue(value="active"))]
                ),
                exact=False,
            )
            chunk_count = info.count
    except Exception as exc:  # noqa: BLE001
        status = "degraded"
        qdrant_status = "unavailable"
        last_error = str(exc)
    return {
        "status": status,
        "qdrant": qdrant_status,
        "chunk_count": chunk_count,
        "failed_count": 0,
        "last_error": last_error,
        "embedding_model": EMBEDDING_MODEL,
    }


@app.post("/internal/rag/index-text")
def index_text(req: IndexRequest) -> Dict[str, Any]:
    return index_content(req, normalize_text(req.content))


@app.post("/internal/rag/index-document")
def index_document(req: IndexRequest) -> Dict[str, Any]:
    if not req.file_url:
        raise HTTPException(status_code=400, detail="file_url is required")
    text = parse_file(req.file_url, req.file_type)
    return index_content(req, text)


def index_content(req: IndexRequest, text: str) -> Dict[str, Any]:
    chunks = chunk_text(text)
    if not chunks:
        raise HTTPException(status_code=400, detail="document content is empty")
    vectors = embed_texts(chunks)
    ensure_collection(len(vectors[0]))
    delete_document(DeleteRequest(document_id=req.document_id))
    points = []
    response_chunks = []
    effective_at_ms = request_time_ms(req, "effective_at", 0)
    expired_at_ms = request_time_ms(req, "expired_at", NO_EXPIRY_MS)
    for idx, (chunk, vector) in enumerate(zip(chunks, vectors)):
        pid = point_id(req.document_id, idx)
        keywords = tokenize(chunk)[:16]
        payload = {
            "chunk_id": pid,
            "document_id": str(req.document_id),
            "chunk_index": idx,
            "title": req.title,
            "category": req.category or "general",
            "content": chunk,
            "summary": chunk[:180],
            "keywords": keywords,
            "source": req.source or "",
            "status": "active",
            "qdrant_point_id": pid,
            "embedding_status": "done",
            "effective_at_ms": effective_at_ms,
            "expired_at_ms": expired_at_ms,
            "metadata": req.metadata or {},
            "updated_at": now_ms(),
        }
        points.append(models.PointStruct(id=pid, vector=vector, payload=payload))
        response_chunks.append(
            {
                "chunk_index": idx,
                "title": req.title,
                "content": chunk,
                "summary": chunk[:180],
                "category": req.category or "general",
                "keywords": keywords,
                "source": req.source or "",
                "status": "active",
                "qdrant_point_id": pid,
                "embedding_status": "done",
            }
        )
    qdrant.upsert(collection_name=QDRANT_COLLECTION, points=points, wait=True)
    invalidate_bm25_cache()
    return {"chunks": response_chunks}


@app.post("/internal/rag/delete-document")
def delete_document(req: DeleteRequest) -> Dict[str, Any]:
    if not collection_exists():
        return {"deleted": True}
    try:
        qdrant.delete(
            collection_name=QDRANT_COLLECTION,
            points_selector=models.FilterSelector(
                filter=models.Filter(
                    must=[
                        models.FieldCondition(
                            key="document_id",
                            match=models.MatchValue(value=str(req.document_id)),
                        )
                    ]
                )
            ),
            wait=True,
        )
    except Exception as exc:  # noqa: BLE001
        raise HTTPException(status_code=502, detail=f"delete document failed: {exc}") from exc
    invalidate_bm25_cache()
    return {"deleted": True}


@app.post("/internal/rag/query")
def query(req: QueryRequest) -> Dict[str, Any]:
    query_text = normalize_text(req.query)
    expanded_text = search_text(query_text, req.context)
    top_k = min(max(req.top_k or 5, 1), 10)
    should_search = need_knowledge(query_text) or need_knowledge(expanded_text)
    if not should_search:
        return {"need_knowledge": False, "confidence": 0, "chunks": []}
    if not collection_exists():
        return {"need_knowledge": True, "confidence": 0, "chunks": []}
    try:
        query_vector = embed_texts([expanded_text or query_text])[0]
        dense_points = qdrant.search(
            collection_name=QDRANT_COLLECTION,
            query_vector=query_vector,
            query_filter=active_filter(req.categories),
            limit=top_k * 8,
            with_payload=True,
        )
        dense = [
            payload_to_chunk(point, point.score, "_dense_score")
            for point in dense_points
            if payload_is_effective(point.payload or {})
        ][: top_k * 2]
    except Exception as exc:  # noqa: BLE001
        raise HTTPException(status_code=502, detail=f"vector search failed: {exc}") from exc

    sparse: List[Dict[str, Any]] = []
    points, _, bm25 = cached_bm25_index()
    tokenized_query = tokenize(expanded_text or query_text)
    if points and bm25 is not None and tokenized_query:
        scores = bm25.get_scores(tokenized_query)
        ranked = sorted(enumerate(scores), key=lambda item: item[1], reverse=True)[: top_k * 4]
        max_score = max((score for _, score in ranked), default=0) or 1
        allowed_categories = set(req.categories or [])
        for index, score in ranked:
            if score <= 0:
                continue
            payload = points[index].payload or {}
            if allowed_categories and payload.get("category") not in allowed_categories:
                continue
            if not payload_is_effective(payload):
                continue
            sparse.append(payload_to_chunk(points[index], score / max_score, "_sparse_score"))
            if len(sparse) >= top_k * 2:
                break
    fused = rrf_fuse(dense, sparse, top_k, expanded_text or query_text)
    confidence = fused[0]["score"] if fused else 0
    return {
        "need_knowledge": True,
        "confidence": confidence,
        "chunks": fused,
    }
