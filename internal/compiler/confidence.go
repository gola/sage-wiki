package compiler

import (
	"strings"

	"github.com/xoai/sage-wiki/internal/manifest"
	"github.com/xoai/sage-wiki/internal/ontology"
)

// ConfidenceScores holds quality metrics for a compiled article.
type ConfidenceScores struct {
	SourceCoverage         float64 // % of source key phrases found in article
	ExtractionCompleteness float64 // % of concepts that have articles
	CrossRefDensity        float64 // % of concepts with ontology relations
	Combined               float64 // weighted average (0.0-1.0)
}

// ScoreArticle computes confidence for a compiled article.
// sourceText is the raw source content, articleText is the compiled article.
// Scores are per-concept: ExtractionCompleteness and CrossRefDensity evaluate
// THIS concept specifically, not the global manifest.
func ScoreArticle(articleText string, sourceText string, conceptName string, mf *manifest.Manifest, ontStore *ontology.Store) ConfidenceScores {
	scores := ConfidenceScores{}

	// Source coverage: what fraction of source key phrases appear in the article
	scores.SourceCoverage = computeSourceCoverage(articleText, sourceText)

	// Extraction completeness: does THIS concept have an article?
	scores.ExtractionCompleteness = computeExtractionCompleteness(conceptName, mf)

	// Cross-reference density: does THIS concept have ontology relations?
	scores.CrossRefDensity = computeCrossRefDensity(conceptName, ontStore)

	// Weighted combination
	scores.Combined = scores.SourceCoverage*0.4 +
		scores.ExtractionCompleteness*0.3 +
		scores.CrossRefDensity*0.3

	return scores
}

// computeSourceCoverage extracts key phrases from the source and checks
// how many appear in the article text.
func computeSourceCoverage(articleText, sourceText string) float64 {
	if sourceText == "" || articleText == "" {
		return 0
	}

	// Extract sentences from source (split on . ! ? and newlines)
	phrases := extractKeyPhrases(sourceText)
	if len(phrases) == 0 {
		return 0
	}

	articleLower := strings.ToLower(articleText)
	found := 0
	for _, phrase := range phrases {
		if strings.Contains(articleLower, strings.ToLower(phrase)) {
			found++
		}
	}

	return float64(found) / float64(len(phrases))
}

// extractKeyPhrases extracts short, meaningful phrases from text.
// Uses a simple approach: take unique multi-word segments (3-6 words) from sentences.
func extractKeyPhrases(text string) []string {
	// Split into sentences
	sentences := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == '\n'
	})

	seen := make(map[string]bool)
	var phrases []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		words := strings.Fields(sentence)

		// Take 3-word windows as key phrases
		for i := 0; i+3 <= len(words) && i < 5; i++ {
			phrase := strings.Join(words[i:i+3], " ")
			phrase = strings.ToLower(phrase)
			if len(phrase) < 10 {
				continue // skip very short phrases
			}
			if !seen[phrase] {
				seen[phrase] = true
				phrases = append(phrases, phrase)
			}
		}
	}

	// Cap at 50 phrases to keep scoring fast
	if len(phrases) > 50 {
		phrases = phrases[:50]
	}

	return phrases
}

// computeExtractionCompleteness checks if THIS concept has an article.
// Returns 1.0 if the concept has an article path, 0.0 otherwise.
func computeExtractionCompleteness(conceptName string, mf *manifest.Manifest) float64 {
	if mf == nil || conceptName == "" {
		return 0
	}
	c, ok := mf.Concepts[conceptName]
	if ok && c.ArticlePath != "" {
		return 1.0
	}
	return 0.0
}

// computeCrossRefDensity checks if THIS concept has ontology relations.
// Returns min(1.0, relationsCount / 2.0) — a concept with 2+ relations
// scores 1.0 (diminishing returns beyond that).
func computeCrossRefDensity(conceptName string, ontStore *ontology.Store) float64 {
	if ontStore == nil || conceptName == "" {
		return 0
	}

	entity, _ := ontStore.GetEntity(conceptName)
	if entity == nil {
		return 0
	}
	relations, _ := ontStore.GetRelations(entity.ID, ontology.Both, "")
	count := float64(len(relations))
	if count >= 2 {
		return 1.0
	}
	return count / 2.0
}
