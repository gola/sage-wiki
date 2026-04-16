package compiler

import (
	"testing"

	"github.com/xoai/sage-wiki/internal/manifest"
)

func TestComputeSourceCoverage(t *testing.T) {
	source := "Self-attention computes contextual representations. Flash attention optimizes memory access patterns for transformer models."
	article := "Self-attention computes contextual representations by weighing input tokens. Flash attention reduces memory overhead."

	coverage := computeSourceCoverage(article, source)
	if coverage < 0.2 {
		t.Errorf("coverage = %.2f, expected >= 0.2 (some source phrases should appear)", coverage)
	}
	if coverage > 1.0 {
		t.Errorf("coverage = %.2f, should be <= 1.0", coverage)
	}
}

func TestComputeSourceCoverage_Empty(t *testing.T) {
	if c := computeSourceCoverage("", "some source"); c != 0 {
		t.Errorf("empty article should give 0, got %.2f", c)
	}
	if c := computeSourceCoverage("some article", ""); c != 0 {
		t.Errorf("empty source should give 0, got %.2f", c)
	}
}

func TestComputeExtractionCompleteness_PerConcept(t *testing.T) {
	mf := manifest.New()
	mf.AddConcept("concept-a", "wiki/concepts/concept-a.md", []string{"raw/a.md"})
	mf.AddConcept("concept-b", "", []string{"raw/b.md"}) // no article path

	// concept-a has an article → 1.0
	if c := computeExtractionCompleteness("concept-a", mf); c != 1.0 {
		t.Errorf("concept-a completeness = %.2f, want 1.0 (has article)", c)
	}

	// concept-b has no article → 0.0
	if c := computeExtractionCompleteness("concept-b", mf); c != 0.0 {
		t.Errorf("concept-b completeness = %.2f, want 0.0 (no article)", c)
	}

	// unknown concept → 0.0
	if c := computeExtractionCompleteness("unknown", mf); c != 0.0 {
		t.Errorf("unknown completeness = %.2f, want 0.0", c)
	}
}

func TestScoreArticle_Combined(t *testing.T) {
	source := "Neural networks learn hierarchical features. Backpropagation computes gradients efficiently."
	article := "Neural networks learn hierarchical features through multiple layers. Backpropagation is used to compute gradients."

	mf := manifest.New()
	mf.AddConcept("neural-networks", "wiki/concepts/neural-networks.md", []string{"raw/nn.md"})

	scores := ScoreArticle(article, source, "neural-networks", mf, nil)
	if scores.Combined < 0 || scores.Combined > 1 {
		t.Errorf("combined score = %.2f, should be 0-1", scores.Combined)
	}
	if scores.SourceCoverage <= 0 {
		t.Error("source coverage should be positive")
	}
	if scores.ExtractionCompleteness != 1.0 {
		t.Errorf("extraction completeness = %.2f, want 1.0", scores.ExtractionCompleteness)
	}
}

func TestExtractKeyPhrases(t *testing.T) {
	text := "Self-attention computes contextual representations by weighing tokens. This enables parallel processing of sequences."
	phrases := extractKeyPhrases(text)
	if len(phrases) == 0 {
		t.Fatal("expected at least one key phrase")
	}
	if len(phrases) > 50 {
		t.Errorf("phrases should be capped at 50, got %d", len(phrases))
	}
}
