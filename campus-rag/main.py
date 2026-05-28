import io
import os
import re
import time
import uuid
from typing import Any, Dict, List, Optional
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
    metadata: Dict[str, str] = Field(default_factory=dict)


class DeleteRequest(BaseModel):
    document_id: int


class QueryRequest(BaseModel):
    query: str
    top_k: int = 5
    categories: List[str] = Field(default_factory=list)


def now_ms() -> int:
    return int(time.time() * 1000)


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


def chunk_text(text: str) -> List[str]:
    text = normalize_text(text)
    if not text:
        return []
    chunks: List[str] = []
    start = 0
    while start < len(text):
        end = min(len(text), start + CHUNK_SIZE)
        chunk = text[start:end].strip()
        if chunk:
            chunks.append(chunk)
        if end >= len(text):
            break
        start = max(end - CHUNK_OVERLAP, start + 1)
    return chunks


def tokenize(text: str) -> List[str]:
    words = [w.strip().lower() for w in jieba.cut(text or "") if w.strip()]
    grams = re.findall(r"[a-zA-Z0-9_]+", text or "")
    return words + [g.lower() for g in grams]


def need_knowledge(query: str) -> bool:
    q = query.strip()
    if len(q) <= 4:
        return False
    keywords = [
        "报到", "宿舍", "校区", "校园网", "军训", "快递", "交通", "路线", "教务", "课表",
        "选课", "学费", "缴费", "深圳职业技术大学", "深汕", "社团", "新生", "学院", "通知",
        "什么时候", "在哪里", "怎么去", "怎么办", "要求", "规定", "政策",
    ]
    return any(k in q for k in keywords)


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


def payload_to_chunk(point: Any, score: float = 0.0) -> Dict[str, Any]:
    payload = point.payload or {}
    return {
        "chunk_id": str(payload.get("chunk_id") or point.id),
        "document_id": str(payload.get("document_id") or ""),
        "title": payload.get("title") or "",
        "category": payload.get("category") or "general",
        "content": payload.get("content") or "",
        "source": payload.get("source") or "",
        "score": round(float(score), 4),
    }


def scroll_active(categories: Optional[List[str]] = None, limit: int = 1000) -> List[Any]:
    conditions = [models.FieldCondition(key="status", match=models.MatchValue(value="active"))]
    if categories:
        conditions.append(models.FieldCondition(key="category", match=models.MatchAny(any=categories)))
    points, _ = qdrant.scroll(
        collection_name=QDRANT_COLLECTION,
        scroll_filter=models.Filter(must=conditions),
        limit=limit,
        with_payload=True,
        with_vectors=False,
    )
    return points


def rrf_fuse(dense: List[Dict[str, Any]], sparse: List[Dict[str, Any]], top_k: int) -> List[Dict[str, Any]]:
    scores: Dict[str, float] = {}
    payloads: Dict[str, Dict[str, Any]] = {}
    for rank, item in enumerate(dense):
        key = item["chunk_id"]
        scores[key] = scores.get(key, 0.0) + 1.0 / (60 + rank + 1)
        payloads[key] = item
    for rank, item in enumerate(sparse):
        key = item["chunk_id"]
        scores[key] = scores.get(key, 0.0) + 1.0 / (60 + rank + 1)
        payloads[key] = item
    fused = sorted(scores.items(), key=lambda kv: kv[1], reverse=True)
    if not fused:
        return []
    max_score = fused[0][1] or 1
    out = []
    for key, score in fused[:top_k]:
        item = dict(payloads[key])
        item["score"] = round(score / max_score, 4)
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
    return {"chunks": response_chunks}


@app.post("/internal/rag/delete-document")
def delete_document(req: DeleteRequest) -> Dict[str, Any]:
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
    except Exception:
        pass
    return {"deleted": True}


@app.post("/internal/rag/query")
def query(req: QueryRequest) -> Dict[str, Any]:
    query_text = normalize_text(req.query)
    top_k = min(max(req.top_k or 5, 1), 10)
    should_search = need_knowledge(query_text)
    if not should_search:
        return {"need_knowledge": False, "confidence": 0, "chunks": []}
    try:
        query_vector = embed_texts([query_text])[0]
        conditions = [models.FieldCondition(key="status", match=models.MatchValue(value="active"))]
        if req.categories:
            conditions.append(models.FieldCondition(key="category", match=models.MatchAny(any=req.categories)))
        dense_points = qdrant.search(
            collection_name=QDRANT_COLLECTION,
            query_vector=query_vector,
            query_filter=models.Filter(must=conditions),
            limit=top_k * 2,
            with_payload=True,
        )
        dense = [payload_to_chunk(point, point.score) for point in dense_points]
    except Exception as exc:  # noqa: BLE001
        raise HTTPException(status_code=502, detail=f"vector search failed: {exc}") from exc

    sparse: List[Dict[str, Any]] = []
    points = scroll_active(req.categories, limit=1000)
    if points:
        corpus = [tokenize((p.payload or {}).get("content", "")) for p in points]
        tokenized_query = tokenize(query_text)
        if tokenized_query and any(corpus):
            bm25 = BM25Okapi(corpus)
            scores = bm25.get_scores(tokenized_query)
            ranked = sorted(enumerate(scores), key=lambda item: item[1], reverse=True)[: top_k * 2]
            max_score = max((score for _, score in ranked), default=0) or 1
            for index, score in ranked:
                if score <= 0:
                    continue
                sparse.append(payload_to_chunk(points[index], score / max_score))
    fused = rrf_fuse(dense, sparse, top_k)
    confidence = fused[0]["score"] if fused else 0
    return {
        "need_knowledge": True,
        "confidence": confidence,
        "chunks": fused,
    }
