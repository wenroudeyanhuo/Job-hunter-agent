package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
)

const (
	SemanticMemoryKindJob    = "job"
	SemanticMemoryProvider   = "local_hash"
	SemanticMemoryDimensions = 64
	semanticMemoryMaxContent = 1600
)

type SemanticMemoryItem struct {
	ID                 int64             `json:"id"`
	Kind               string            `json:"kind"`
	ReferenceID        int64             `json:"reference_id"`
	Title              string            `json:"title"`
	Content            string            `json:"content"`
	Metadata           map[string]string `json:"metadata"`
	Embedding          []float64         `json:"-"`
	EmbeddingProvider  string            `json:"embedding_provider"`
	EmbeddingDimension int               `json:"embedding_dimension"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type SemanticMemoryMatch struct {
	ID                 int64             `json:"id"`
	Kind               string            `json:"kind"`
	ReferenceID        int64             `json:"reference_id"`
	Title              string            `json:"title"`
	Content            string            `json:"content"`
	Metadata           map[string]string `json:"metadata"`
	Score              float64           `json:"score"`
	EmbeddingProvider  string            `json:"embedding_provider"`
	EmbeddingDimension int               `json:"embedding_dimension"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type SemanticMemoryQuery struct {
	Query string
	Kind  string
	Limit int
}

type SemanticMemoryRebuildResult struct {
	Scanned int `json:"scanned"`
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

type SemanticMemoryStats struct {
	TotalItems int    `json:"total_items"`
	JobItems   int    `json:"job_items"`
	Provider   string `json:"provider"`
	Dimension  int    `json:"dimension"`
}

func (r *Repository) RebuildSemanticMemory(ctx context.Context) (SemanticMemoryRebuildResult, error) {
	jobs, err := r.ListJobs(ctx, ListFilter{})
	if err != nil {
		return SemanticMemoryRebuildResult{}, err
	}
	result := SemanticMemoryRebuildResult{Scanned: len(jobs)}
	for _, job := range jobs {
		if strings.TrimSpace(job.Title) == "" && strings.TrimSpace(job.Description) == "" {
			result.Skipped++
			continue
		}
		if _, err := r.UpsertSemanticMemoryItem(ctx, SemanticMemoryItemFromJob(job)); err != nil {
			return SemanticMemoryRebuildResult{}, err
		}
		result.Created++
	}
	return result, nil
}

func SemanticMemoryItemFromJob(job domain.Job) SemanticMemoryItem {
	title := strings.TrimSpace(job.Company + " " + job.Title)
	content := strings.Join([]string{
		"company: " + job.Company,
		"title: " + job.Title,
		"city: " + job.City,
		"directions: " + strings.Join(job.DirectionTags, ", "),
		"description: " + job.Description,
		"recommend reasons: " + strings.Join(job.RecommendReasons, ", "),
	}, "\n")
	if len(content) > semanticMemoryMaxContent {
		content = content[:semanticMemoryMaxContent]
	}
	return SemanticMemoryItem{
		Kind:        SemanticMemoryKindJob,
		ReferenceID: job.ID,
		Title:       title,
		Content:     content,
		Metadata: map[string]string{
			"company": job.Company,
			"city":    job.City,
			"status":  string(job.Status),
			"score":   fmt.Sprintf("%d", job.MatchScore),
		},
		EmbeddingProvider:  SemanticMemoryProvider,
		EmbeddingDimension: SemanticMemoryDimensions,
	}
}

func (r *Repository) UpsertSemanticMemoryItem(ctx context.Context, item SemanticMemoryItem) (SemanticMemoryItem, error) {
	item.Kind = strings.TrimSpace(item.Kind)
	item.Title = strings.TrimSpace(item.Title)
	item.Content = strings.TrimSpace(item.Content)
	if item.Kind == "" || item.ReferenceID <= 0 || item.Content == "" {
		return SemanticMemoryItem{}, fmt.Errorf("semantic memory kind, reference id, and content are required")
	}
	item.EmbeddingProvider = SemanticMemoryProvider
	item.EmbeddingDimension = SemanticMemoryDimensions
	item.Embedding = LocalHashEmbedding(item.Title + "\n" + item.Content)
	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return SemanticMemoryItem{}, fmt.Errorf("encode semantic memory metadata: %w", err)
	}
	embeddingJSON, err := json.Marshal(item.Embedding)
	if err != nil {
		return SemanticMemoryItem{}, fmt.Errorf("encode semantic memory embedding: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO semantic_memory_items (
			kind, reference_id, title, content, metadata_json, embedding_json,
			embedding_provider, embedding_dimension
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(kind, reference_id) DO UPDATE SET
			title = excluded.title,
			content = excluded.content,
			metadata_json = excluded.metadata_json,
			embedding_json = excluded.embedding_json,
			embedding_provider = excluded.embedding_provider,
			embedding_dimension = excluded.embedding_dimension,
			updated_at = CURRENT_TIMESTAMP
	`, item.Kind, item.ReferenceID, item.Title, item.Content, string(metadataJSON), string(embeddingJSON), item.EmbeddingProvider, item.EmbeddingDimension)
	if err != nil {
		return SemanticMemoryItem{}, fmt.Errorf("upsert semantic memory item: %w", err)
	}
	id, _ := result.LastInsertId()
	if id > 0 {
		return r.GetSemanticMemoryItem(ctx, id)
	}
	return r.GetSemanticMemoryItemByReference(ctx, item.Kind, item.ReferenceID)
}

func (r *Repository) SearchSemanticMemory(ctx context.Context, query SemanticMemoryQuery) ([]SemanticMemoryMatch, error) {
	query.Query = strings.TrimSpace(query.Query)
	query.Kind = strings.TrimSpace(query.Kind)
	if query.Limit <= 0 || query.Limit > 20 {
		query.Limit = 8
	}
	if query.Query == "" {
		return []SemanticMemoryMatch{}, nil
	}
	target := LocalHashEmbedding(query.Query)
	items, err := r.listSemanticMemoryItems(ctx, query.Kind, 200)
	if err != nil {
		return nil, err
	}
	matches := make([]SemanticMemoryMatch, 0, len(items))
	for _, item := range items {
		score := CosineSimilarity(target, item.Embedding)
		if score <= 0 {
			continue
		}
		matches = append(matches, SemanticMemoryMatch{
			ID:                 item.ID,
			Kind:               item.Kind,
			ReferenceID:        item.ReferenceID,
			Title:              item.Title,
			Content:            item.Content,
			Metadata:           item.Metadata,
			Score:              score,
			EmbeddingProvider:  item.EmbeddingProvider,
			EmbeddingDimension: item.EmbeddingDimension,
			UpdatedAt:          item.UpdatedAt,
		})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].UpdatedAt.After(matches[j].UpdatedAt)
		}
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > query.Limit {
		return matches[:query.Limit], nil
	}
	return matches, nil
}

func (r *Repository) GetSemanticMemoryStats(ctx context.Context) (SemanticMemoryStats, error) {
	stats := SemanticMemoryStats{Provider: SemanticMemoryProvider, Dimension: SemanticMemoryDimensions}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM semantic_memory_items`).Scan(&stats.TotalItems); err != nil {
		return SemanticMemoryStats{}, fmt.Errorf("count semantic memory items: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM semantic_memory_items WHERE kind = ?`, SemanticMemoryKindJob).Scan(&stats.JobItems); err != nil {
		return SemanticMemoryStats{}, fmt.Errorf("count semantic job memory items: %w", err)
	}
	return stats, nil
}

func (r *Repository) GetSemanticMemoryItem(ctx context.Context, id int64) (SemanticMemoryItem, error) {
	row := r.db.QueryRowContext(ctx, selectSemanticMemorySQL()+` WHERE id = ?`, id)
	return scanSemanticMemoryItem(row)
}

func (r *Repository) GetSemanticMemoryItemByReference(ctx context.Context, kind string, referenceID int64) (SemanticMemoryItem, error) {
	row := r.db.QueryRowContext(ctx, selectSemanticMemorySQL()+` WHERE kind = ? AND reference_id = ?`, kind, referenceID)
	return scanSemanticMemoryItem(row)
}

func (r *Repository) listSemanticMemoryItems(ctx context.Context, kind string, limit int) ([]SemanticMemoryItem, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := selectSemanticMemorySQL()
	args := []any{}
	if kind != "" {
		query += ` WHERE kind = ?`
		args = append(args, kind)
	}
	query += ` ORDER BY updated_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list semantic memory items: %w", err)
	}
	defer rows.Close()
	out := []SemanticMemoryItem{}
	for rows.Next() {
		item, err := scanSemanticMemoryItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semantic memory items: %w", err)
	}
	return out, nil
}

func selectSemanticMemorySQL() string {
	return `SELECT id, kind, reference_id, title, content, metadata_json, embedding_json, embedding_provider, embedding_dimension, created_at, updated_at FROM semantic_memory_items`
}

func scanSemanticMemoryItem(scanner interface{ Scan(dest ...any) error }) (SemanticMemoryItem, error) {
	var item SemanticMemoryItem
	var metadataJSON string
	var embeddingJSON string
	if err := scanner.Scan(&item.ID, &item.Kind, &item.ReferenceID, &item.Title, &item.Content, &metadataJSON, &embeddingJSON, &item.EmbeddingProvider, &item.EmbeddingDimension, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return SemanticMemoryItem{}, err
		}
		return SemanticMemoryItem{}, fmt.Errorf("scan semantic memory item: %w", err)
	}
	if err := json.Unmarshal([]byte(metadataJSON), &item.Metadata); err != nil {
		return SemanticMemoryItem{}, fmt.Errorf("decode semantic memory metadata: %w", err)
	}
	if err := json.Unmarshal([]byte(embeddingJSON), &item.Embedding); err != nil {
		return SemanticMemoryItem{}, fmt.Errorf("decode semantic memory embedding: %w", err)
	}
	return item, nil
}

var semanticTokenPattern = regexp.MustCompile(`[a-z0-9_+#]+|[\p{Han}]{2,}`)

func LocalHashEmbedding(text string) []float64 {
	vector := make([]float64, SemanticMemoryDimensions)
	for _, token := range semanticTokens(text) {
		h := fnv.New32a()
		_, _ = h.Write([]byte(token))
		index := int(h.Sum32() % uint32(SemanticMemoryDimensions))
		vector[index] += 1
	}
	normalizeVector(vector)
	return vector
}

func semanticTokens(text string) []string {
	raw := semanticTokenPattern.FindAllString(strings.ToLower(text), -1)
	out := make([]string, 0, len(raw)*2)
	for _, token := range raw {
		out = append(out, token)
		if strings.Contains(token, "_") {
			out = append(out, strings.Split(token, "_")...)
		}
	}
	return out
}

func normalizeVector(vector []float64) {
	var sum float64
	for _, value := range vector {
		sum += value * value
	}
	if sum == 0 {
		return
	}
	length := math.Sqrt(sum)
	for index := range vector {
		vector[index] = vector[index] / length
	}
}

func CosineSimilarity(a []float64, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var score float64
	for index := range a {
		score += a[index] * b[index]
	}
	return score
}
