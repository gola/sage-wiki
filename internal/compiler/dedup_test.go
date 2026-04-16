package compiler

import (
	"testing"

	"github.com/xoai/sage-wiki/internal/vectors"
)

// mockEmbedder returns predictable embeddings for testing.
type mockEmbedder struct {
	embeddings map[string][]float32
}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	if vec, ok := m.embeddings[text]; ok {
		return vec, nil
	}
	// Generate a simple hash-based vector for unknown texts
	vec := make([]float32, 4)
	for i, r := range text {
		vec[i%4] += float32(r) / 1000.0
	}
	// Normalize
	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec, nil
}

func (m *mockEmbedder) Name() string       { return "mock" }
func (m *mockEmbedder) Dimensions() int    { return 4 }

func TestDedupCache_SimilarConcepts(t *testing.T) {
	// Create embeddings where "flash attention" and "flash-attention" are very similar
	// but "database indexing" is very different
	flashVec := []float32{0.9, 0.1, 0.0, 0.1}
	flashAltVec := []float32{0.88, 0.12, 0.01, 0.09}
	dbVec := []float32{0.1, 0.1, 0.9, 0.1}

	embedder := &mockEmbedder{
		embeddings: map[string][]float32{
			"flash attention":  flashVec,
			"flash-attention":  flashAltVec,
			"database indexing": dbVec,
		},
	}

	dc := NewDedupCache(embedder, nil, 0.85)

	// Seed with existing concept
	dc.Seed([]string{"flash attention"})

	// Check similar concept — should match
	match, score, vec := dc.CheckDuplicate("flash-attention")
	sim := vectors.CosineSimilarity(flashVec, flashAltVec)
	t.Logf("flash attention vs flash-attention: cosine=%.4f, match=%q, score=%.4f", sim, match, score)

	if sim < 0.85 {
		t.Skipf("mock embeddings not similar enough (%.4f), adjusting test", sim)
	}
	if match != "flash attention" {
		t.Errorf("expected match 'flash attention', got %q (score %.4f)", match, score)
	}
	if vec == nil {
		t.Error("expected non-nil vec from CheckDuplicate")
	}

	// Check dissimilar concept — should not match
	match2, score2, _ := dc.CheckDuplicate("database indexing")
	if match2 != "" {
		t.Errorf("expected no match for 'database indexing', got %q (score %.4f)", match2, score2)
	}
}

func TestDedupCache_Add(t *testing.T) {
	embedder := &mockEmbedder{embeddings: map[string][]float32{
		"concept-a": {0.5, 0.5, 0.0, 0.0},
	}}

	dc := NewDedupCache(embedder, nil, 0.85)
	if dc.Size() != 0 {
		t.Errorf("initial size = %d, want 0", dc.Size())
	}

	dc.Add("concept-a")
	if dc.Size() != 1 {
		t.Errorf("after add size = %d, want 1", dc.Size())
	}

	// Adding same concept again should not increase size
	dc.Add("concept-a")
	if dc.Size() != 1 {
		t.Errorf("after duplicate add size = %d, want 1", dc.Size())
	}
}

func TestDedupCache_NilEmbedder(t *testing.T) {
	dc := NewDedupCache(nil, nil, 0.85)

	dc.Seed([]string{"test"})
	if dc.Size() != 0 {
		t.Error("nil embedder should not seed")
	}

	match, _, _ := dc.CheckDuplicate("test")
	if match != "" {
		t.Error("nil embedder should return no match")
	}
}

func TestDedupCache_DefaultThreshold(t *testing.T) {
	dc := NewDedupCache(nil, nil, 0)
	if dc.threshold != 0.85 {
		t.Errorf("default threshold = %.2f, want 0.85", dc.threshold)
	}
}
