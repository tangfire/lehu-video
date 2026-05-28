package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type CampusRAGClient interface {
	Health(ctx context.Context) (*CampusRAGHealth, error)
	IndexDocument(ctx context.Context, req *CampusRAGIndexRequest) (*CampusRAGIndexResponse, error)
	IndexText(ctx context.Context, req *CampusRAGIndexRequest) (*CampusRAGIndexResponse, error)
	DeleteDocument(ctx context.Context, documentID int64) error
	Query(ctx context.Context, req *CampusRAGQueryRequest) (*CampusRAGQueryResponse, error)
}

type CampusRAGHTTPClient struct {
	baseURL    string
	httpClient *http.Client
	log        *log.Helper
}

type CampusRAGHealth struct {
	Status      string `json:"status"`
	Qdrant      string `json:"qdrant"`
	ChunkCount  int64  `json:"chunk_count"`
	FailedCount int64  `json:"failed_count"`
	LastError   string `json:"last_error,omitempty"`
}

type CampusRAGIndexRequest struct {
	DocumentID int64             `json:"document_id"`
	Title      string            `json:"title"`
	Category   string            `json:"category"`
	Source     string            `json:"source"`
	FileURL    string            `json:"file_url,omitempty"`
	FileType   string            `json:"file_type,omitempty"`
	Content    string            `json:"content,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type CampusRAGIndexResponse struct {
	Chunks []*CampusKnowledgeChunk `json:"chunks"`
}

type CampusRAGQueryRequest struct {
	Query      string   `json:"query"`
	TopK       int      `json:"top_k"`
	Categories []string `json:"categories,omitempty"`
}

type CampusRAGQueryResponse struct {
	NeedKnowledge bool                   `json:"need_knowledge"`
	Confidence    float64                `json:"confidence"`
	Chunks        []*CampusRAGQueryChunk `json:"chunks"`
}

type CampusRAGQueryChunk struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	Category   string  `json:"category"`
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	Score      float64 `json:"score"`
}

func NewCampusRAGClient(logger log.Logger) CampusRAGClient {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("CAMPUS_RAG_BASE_URL")), "/")
	if baseURL == "" {
		return &noopCampusRAGClient{}
	}
	return &CampusRAGHTTPClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: envDuration("CAMPUS_RAG_TIMEOUT", 5*time.Second)},
		log:        log.NewHelper(logger),
	}
}

func (c *CampusRAGHTTPClient) Health(ctx context.Context) (*CampusRAGHealth, error) {
	var out CampusRAGHealth
	if err := c.do(ctx, http.MethodGet, "/healthz", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *CampusRAGHTTPClient) IndexDocument(ctx context.Context, req *CampusRAGIndexRequest) (*CampusRAGIndexResponse, error) {
	var out CampusRAGIndexResponse
	if err := c.do(ctx, http.MethodPost, "/internal/rag/index-document", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *CampusRAGHTTPClient) IndexText(ctx context.Context, req *CampusRAGIndexRequest) (*CampusRAGIndexResponse, error) {
	var out CampusRAGIndexResponse
	if err := c.do(ctx, http.MethodPost, "/internal/rag/index-text", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *CampusRAGHTTPClient) DeleteDocument(ctx context.Context, documentID int64) error {
	return c.do(ctx, http.MethodPost, "/internal/rag/delete-document", map[string]int64{"document_id": documentID}, nil)
}

func (c *CampusRAGHTTPClient) Query(ctx context.Context, req *CampusRAGQueryRequest) (*CampusRAGQueryResponse, error) {
	var out CampusRAGQueryResponse
	if err := c.do(ctx, http.MethodPost, "/internal/rag/query", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *CampusRAGHTTPClient) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("rag status=%d body=%s", resp.StatusCode, trimLimit(string(raw), 400))
	}
	if out == nil {
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

type noopCampusRAGClient struct{}

func (noopCampusRAGClient) Health(context.Context) (*CampusRAGHealth, error) {
	return &CampusRAGHealth{Status: "disabled", Qdrant: "disabled"}, nil
}
func (noopCampusRAGClient) IndexDocument(context.Context, *CampusRAGIndexRequest) (*CampusRAGIndexResponse, error) {
	return nil, fmt.Errorf("campus rag disabled")
}
func (noopCampusRAGClient) IndexText(context.Context, *CampusRAGIndexRequest) (*CampusRAGIndexResponse, error) {
	return nil, fmt.Errorf("campus rag disabled")
}
func (noopCampusRAGClient) DeleteDocument(context.Context, int64) error {
	return nil
}
func (noopCampusRAGClient) Query(context.Context, *CampusRAGQueryRequest) (*CampusRAGQueryResponse, error) {
	return &CampusRAGQueryResponse{NeedKnowledge: false}, nil
}

func envDuration(key string, fallback time.Duration) time.Duration {
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
