package retrieval

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"superdevstudio/internal/store"
)

type Service struct {
	store *store.Store
	now   func() time.Time
}

type Request struct {
	ProjectID string
	Query     string
	MaxItems  int
}

type Evidence struct {
	SourceType string         `json:"source_type"`
	SourceID   string         `json:"source_id"`
	Title      string         `json:"title"`
	Snippet    string         `json:"snippet"`
	Score      float64        `json:"score"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

func NewService(s *store.Store) *Service {
	return &Service{store: s, now: time.Now}
}

func (s *Service) Retrieve(ctx context.Context, req Request) ([]Evidence, error) {
	if req.MaxItems <= 0 {
		req.MaxItems = 8
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, nil
	}

	results := make([]Evidence, 0, req.MaxItems*2)

	memories, err := s.store.ListMemories(ctx, req.ProjectID, req.MaxItems*3)
	if err != nil {
		return nil, err
	}
	for _, item := range memories {
		score := hybridScore(query, item.Content, item.Importance, item.CreatedAt, s.now())
		if score <= 0 {
			continue
		}
		results = append(results, Evidence{
			SourceType: "memory",
			SourceID:   item.ID,
			Title:      item.Role,
			Snippet:    trimSnippet(item.Content, 220),
			Score:      score,
			Metadata: map[string]any{
				"tags":       item.Tags,
				"importance": item.Importance,
			},
		})
	}

	knowledge, err := s.store.SearchKnowledge(ctx, req.ProjectID, query, req.MaxItems*3)
	if err != nil {
		return nil, err
	}
	for _, item := range knowledge {
		results = append(results, Evidence{
			SourceType: "knowledge",
			SourceID:   strconv.FormatInt(item.ID, 10),
			Title:      "knowledge-chunk",
			Snippet:    trimSnippet(item.Content, 220),
			Score:      hybridScore(query, item.Content, normalizeKnowledgeScore(item.Score), item.CreatedAt, s.now()),
			Metadata: map[string]any{
				"document_id": item.DocumentID,
				"chunk_index": item.ChunkIndex,
			},
		})
	}

	tasks, err := s.store.ListTasks(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	for _, item := range tasks {
		body := strings.TrimSpace(item.Title + "\n" + item.Description)
		score := hybridScore(query, body, taskPriorityWeight(item.Priority), item.UpdatedAt, s.now())
		if score <= 0 {
			continue
		}
		results = append(results, Evidence{
			SourceType: "task",
			SourceID:   item.ID,
			Title:      item.Title,
			Snippet:    trimSnippet(item.Description, 220),
			Score:      score,
			Metadata: map[string]any{
				"status":   item.Status,
				"priority": item.Priority,
			},
		})
	}

	runs, err := s.store.ListPipelineRuns(ctx, req.ProjectID, req.MaxItems)
	if err != nil {
		return nil, err
	}
	for _, item := range runs {
		body := strings.TrimSpace(item.Prompt + "\n" + item.Stage + "\n" + item.Status)
		score := hybridScore(query, body, 0.6, item.UpdatedAt, s.now())
		if score <= 0 {
			continue
		}
		results = append(results, Evidence{
			SourceType: "run",
			SourceID:   item.ID,
			Title:      item.Stage,
			Snippet:    trimSnippet(item.Prompt, 220),
			Score:      score,
			Metadata: map[string]any{
				"status":          item.Status,
				"change_batch_id": item.ChangeBatchID,
			},
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if math.Abs(results[i].Score-results[j].Score) < 0.0001 {
			return results[i].SourceID < results[j].SourceID
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > req.MaxItems {
		results = results[:req.MaxItems]
	}
	return rerank(results), nil
}

func EncodeMetadata(metadata map[string]any) string {
	if len(metadata) == 0 {
		return "{}"
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func rerank(items []Evidence) []Evidence {
	return items
}

func hybridScore(query, text string, baseWeight float64, createdAt, now time.Time) float64 {
	overlap := lexicalOverlap(query, text)
	if overlap <= 0 {
		return 0
	}
	hours := math.Max(1, now.Sub(createdAt).Hours())
	recency := math.Exp(-hours / 120.0)
	semanticProxy := semanticProxyScore(query, text)
	return overlap*0.55 + semanticProxy*0.25 + recency*0.10 + baseWeight*0.10
}

func lexicalOverlap(query, text string) float64 {
	qTokens := tokenize(query)
	if len(qTokens) == 0 {
		return 0
	}
	tTokens := tokenize(text)
	if len(tTokens) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(tTokens))
	for _, token := range tTokens {
		set[token] = struct{}{}
	}
	matched := 0
	for _, token := range qTokens {
		if _, ok := set[token]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(qTokens))
}

func semanticProxyScore(query, text string) float64 {
	qBigrams := characterBigrams(query)
	tBigrams := characterBigrams(text)
	if len(qBigrams) == 0 || len(tBigrams) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(tBigrams))
	for _, token := range tBigrams {
		set[token] = struct{}{}
	}
	matched := 0
	for _, token := range qBigrams {
		if _, ok := set[token]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(qBigrams))
}

func tokenize(input string) []string {
	parts := strings.FieldsFunc(strings.ToLower(input), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	out := make([]string, 0, len(parts))
	for _, token := range parts {
		if token != "" {
			out = append(out, token)
		}
	}
	return out
}

func characterBigrams(input string) []string {
	runes := []rune(strings.ToLower(strings.TrimSpace(input)))
	if len(runes) < 2 {
		return nil
	}
	out := make([]string, 0, len(runes)-1)
	for idx := 0; idx < len(runes)-1; idx++ {
		pair := string(runes[idx : idx+2])
		if strings.TrimSpace(pair) != "" {
			out = append(out, pair)
		}
	}
	return out
}

func trimSnippet(input string, limit int) string {
	text := strings.TrimSpace(input)
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

func normalizeKnowledgeScore(score float64) float64 {
	if score <= 0 {
		return 0.5
	}
	if score > 1 {
		return 1
	}
	return score
}

func taskPriorityWeight(priority string) float64 {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return 1
	case "low":
		return 0.4
	default:
		return 0.7
	}
}
