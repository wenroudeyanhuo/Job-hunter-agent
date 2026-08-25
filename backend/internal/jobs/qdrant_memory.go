package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultQdrantCollection = "job_hunter_memory"

type externalSemanticMemoryStore interface {
	Upsert(context.Context, SemanticMemoryItem) error
	Search(context.Context, SemanticMemoryQuery, []float64) ([]SemanticMemoryMatch, error)
}

type qdrantSemanticMemoryStore struct {
	baseURL    string
	collection string
	apiKey     string
	client     *http.Client
}

func configuredExternalSemanticMemoryStore() externalSemanticMemoryStore {
	if ConfiguredSemanticMemoryProviderName() != SemanticMemoryProviderQdrant {
		return nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("QDRANT_URL")), "/")
	if baseURL == "" {
		return nil
	}
	collection := strings.TrimSpace(os.Getenv("QDRANT_COLLECTION"))
	if collection == "" {
		collection = defaultQdrantCollection
	}
	return qdrantSemanticMemoryStore{
		baseURL:    baseURL,
		collection: collection,
		apiKey:     strings.TrimSpace(os.Getenv("QDRANT_API_KEY")),
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (s qdrantSemanticMemoryStore) Upsert(ctx context.Context, item SemanticMemoryItem) error {
	if item.ID <= 0 || len(item.Embedding) == 0 {
		return nil
	}
	if err := s.ensureCollection(ctx); err != nil {
		return err
	}
	payload := map[string]any{
		"points": []map[string]any{
			{
				"id":     item.ID,
				"vector": item.Embedding,
				"payload": map[string]any{
					"kind":                item.Kind,
					"reference_id":        item.ReferenceID,
					"title":               item.Title,
					"content":             item.Content,
					"metadata":            item.Metadata,
					"embedding_provider":  SemanticMemoryProviderQdrant,
					"embedding_dimension": item.EmbeddingDimension,
					"updated_at":          item.UpdatedAt.Format(time.RFC3339),
				},
			},
		},
	}
	return s.doJSON(ctx, http.MethodPut, "/collections/"+url.PathEscape(s.collection)+"/points?wait=true", payload, nil)
}

func (s qdrantSemanticMemoryStore) Search(ctx context.Context, query SemanticMemoryQuery, vector []float64) ([]SemanticMemoryMatch, error) {
	if len(vector) == 0 {
		return []SemanticMemoryMatch{}, nil
	}
	body := map[string]any{
		"vector":       vector,
		"limit":        query.Limit,
		"with_payload": true,
	}
	if strings.TrimSpace(query.Kind) != "" {
		body["filter"] = map[string]any{
			"must": []map[string]any{
				{
					"key":   "kind",
					"match": map[string]string{"value": strings.TrimSpace(query.Kind)},
				},
			},
		}
	}
	var response struct {
		Result []struct {
			ID      any           `json:"id"`
			Score   float64       `json:"score"`
			Payload qdrantPayload `json:"payload"`
		} `json:"result"`
	}
	if err := s.doJSON(ctx, http.MethodPost, "/collections/"+url.PathEscape(s.collection)+"/points/search", body, &response); err != nil {
		return nil, err
	}
	matches := make([]SemanticMemoryMatch, 0, len(response.Result))
	for _, result := range response.Result {
		updatedAt, _ := time.Parse(time.RFC3339, result.Payload.UpdatedAt)
		matches = append(matches, SemanticMemoryMatch{
			ID:                 qdrantPointID(result.ID),
			Kind:               result.Payload.Kind,
			ReferenceID:        result.Payload.ReferenceID,
			Title:              result.Payload.Title,
			Content:            result.Payload.Content,
			Metadata:           result.Payload.Metadata,
			Score:              result.Score,
			EmbeddingProvider:  defaultText(result.Payload.EmbeddingProvider, SemanticMemoryProviderQdrant),
			EmbeddingDimension: firstPositiveInt(result.Payload.EmbeddingDimension, SemanticMemoryDimensions),
			UpdatedAt:          updatedAt,
		})
	}
	return matches, nil
}

func (s qdrantSemanticMemoryStore) ensureCollection(ctx context.Context) error {
	body := map[string]any{
		"vectors": map[string]any{
			"size":     SemanticMemoryDimensions,
			"distance": "Cosine",
		},
	}
	return s.doJSON(ctx, http.MethodPut, "/collections/"+url.PathEscape(s.collection), body, nil)
}

func (s qdrantSemanticMemoryStore) doJSON(ctx context.Context, method string, path string, body any, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode qdrant request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("build qdrant request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant returned %s", resp.Status)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode qdrant response: %w", err)
	}
	return nil
}

type qdrantPayload struct {
	Kind               string            `json:"kind"`
	ReferenceID        int64             `json:"reference_id"`
	Title              string            `json:"title"`
	Content            string            `json:"content"`
	Metadata           map[string]string `json:"metadata"`
	EmbeddingProvider  string            `json:"embedding_provider"`
	EmbeddingDimension int               `json:"embedding_dimension"`
	UpdatedAt          string            `json:"updated_at"`
}

func qdrantPointID(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case string:
		id, _ := strconv.ParseInt(typed, 10, 64)
		return id
	default:
		return 0
	}
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
