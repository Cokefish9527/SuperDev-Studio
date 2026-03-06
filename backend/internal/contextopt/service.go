package contextopt

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"superdevstudio/internal/store"
)

type Service struct {
	store *store.Store
	now   func() time.Time
}

func NewService(s *store.Store) *Service {
	return &Service{store: s, now: time.Now}
}

type BuildRequest struct {
	ProjectID   string
	Query       string
	TokenBudget int
	MaxItems    int
}

type scoredMemory struct {
	Memory store.Memory
	Score  float64
}

func (s *Service) BuildContextPack(ctx context.Context, req BuildRequest) (store.ContextPack, error) {
	if req.TokenBudget <= 0 {
		req.TokenBudget = 1200
	}
	if req.MaxItems <= 0 {
		req.MaxItems = 8
	}

	memories, err := s.store.ListMemories(ctx, req.ProjectID, 200)
	if err != nil {
		return store.ContextPack{}, err
	}
	knowledge, err := s.store.SearchKnowledge(ctx, req.ProjectID, req.Query, req.MaxItems*2)
	if err != nil {
		return store.ContextPack{}, err
	}

	scored := make([]scoredMemory, 0, len(memories))
	for _, m := range memories {
		scored = append(scored, scoredMemory{Memory: m, Score: memoryScore(req.Query, m, s.now())})
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	pack := store.ContextPack{
		Query:       req.Query,
		TokenBudget: req.TokenBudget,
	}

	usedTokens := 0
	for _, candidate := range scored {
		if len(pack.Memories) >= req.MaxItems {
			break
		}
		estimated := estimateTokens(candidate.Memory.Content)
		if usedTokens+estimated > req.TokenBudget {
			continue
		}
		pack.Memories = append(pack.Memories, candidate.Memory)
		usedTokens += estimated
	}

	for _, chunk := range knowledge {
		if len(pack.Knowledge) >= req.MaxItems {
			break
		}
		estimated := estimateTokens(chunk.Content)
		if usedTokens+estimated > req.TokenBudget {
			continue
		}
		pack.Knowledge = append(pack.Knowledge, chunk)
		usedTokens += estimated
	}

	pack.EstimatedTokens = usedTokens
	pack.Summary = buildSummary(pack)
	return pack, nil
}

func memoryScore(query string, memory store.Memory, now time.Time) float64 {
	overlap := lexicalOverlap(query, memory.Content)
	hours := math.Max(1, now.Sub(memory.CreatedAt).Hours())
	recency := math.Exp(-hours / 72.0)
	importance := memory.Importance
	if importance < 0 {
		importance = 0
	}
	if importance > 1 {
		importance = 1
	}
	return importance*3.0 + overlap*2.0 + recency
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

	tSet := map[string]struct{}{}
	for _, t := range tTokens {
		tSet[t] = struct{}{}
	}

	matched := 0
	for _, q := range qTokens {
		if _, ok := tSet[q]; ok {
			matched++
		}
	}
	return float64(matched) / float64(len(qTokens))
}

func tokenize(input string) []string {
	parts := strings.FieldsFunc(strings.ToLower(input), func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func estimateTokens(text string) int {
	runes := utf8.RuneCountInString(text)
	if runes == 0 {
		return 0
	}
	tokens := runes / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}

func buildSummary(pack store.ContextPack) string {
	lines := []string{}
	if len(pack.Memories) > 0 {
		lines = append(lines, "记忆模块提要：")
		for i, item := range pack.Memories {
			if i >= 3 {
				break
			}
			lines = append(lines, "- "+trimForSummary(item.Content, 120))
		}
	}
	if len(pack.Knowledge) > 0 {
		lines = append(lines, "知识库提要：")
		for i, item := range pack.Knowledge {
			if i >= 3 {
				break
			}
			lines = append(lines, "- "+trimForSummary(item.Content, 120))
		}
	}
	if len(lines) == 0 {
		return "暂无可用上下文，建议先沉淀记忆或导入知识库文档。"
	}
	return strings.Join(lines, "\n")
}

func trimForSummary(input string, maxRunes int) string {
	text := strings.TrimSpace(input)
	if utf8.RuneCountInString(text) <= maxRunes {
		return text
	}
	runes := []rune(text)
	return string(runes[:maxRunes]) + "..."
}
