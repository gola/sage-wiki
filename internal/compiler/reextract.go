package compiler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xoai/sage-wiki/internal/config"
	"github.com/xoai/sage-wiki/internal/embed"
	"github.com/xoai/sage-wiki/internal/llm"
	"github.com/xoai/sage-wiki/internal/log"
	"github.com/xoai/sage-wiki/internal/manifest"
	"github.com/xoai/sage-wiki/internal/memory"
	"github.com/xoai/sage-wiki/internal/ontology"
	"github.com/xoai/sage-wiki/internal/prompts"
	"github.com/xoai/sage-wiki/internal/storage"
	"github.com/xoai/sage-wiki/internal/vectors"
)

// ReExtract re-runs Pass 2 (concept extraction) and Pass 3 (article writing)
// using existing summaries from wiki/summaries/. Skips Pass 0 and Pass 1.
func ReExtract(projectDir string) (*CompileResult, error) {
	result := &CompileResult{}

	cfg, err := config.Load(filepath.Join(projectDir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("re-extract: load config: %w", err)
	}

	// Load user prompt overrides if prompts/ directory exists
	if err := prompts.LoadFromDir(filepath.Join(projectDir, "prompts")); err != nil {
		log.Warn("failed to load custom prompts", "error", err)
	}

	mf, err := manifest.Load(filepath.Join(projectDir, ".manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("re-extract: load manifest: %w", err)
	}

	// Read existing summaries from disk
	summaryDir := filepath.Join(projectDir, cfg.Output, "summaries")
	entries, err := os.ReadDir(summaryDir)
	if err != nil {
		return nil, fmt.Errorf("re-extract: no summaries found at %s: %w", summaryDir, err)
	}

	var summaries []SummaryResult
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(summaryDir, e.Name()))
		if err != nil {
			continue
		}
		summaries = append(summaries, SummaryResult{
			SourcePath:  e.Name(),
			SummaryPath: filepath.Join(cfg.Output, "summaries", e.Name()),
			Summary:     string(data),
		})
	}

	log.Info("re-extract: loaded existing summaries", "count", len(summaries))

	if len(summaries) == 0 {
		return result, fmt.Errorf("re-extract: no summaries found — run sage-wiki compile first")
	}

	// Create LLM client
	client, err := llm.NewClient(cfg.API.Provider, cfg.API.APIKey, cfg.API.BaseURL, cfg.API.RateLimit, cfg.API.TimeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("re-extract: create LLM client: %w", err)
	}

	// Open DB
	db, err := storage.Open(filepath.Join(projectDir, ".sage", "wiki.db"))
	if err != nil {
		return nil, fmt.Errorf("re-extract: open db: %w", err)
	}
	defer db.Close()

	memStore := memory.NewStore(db)
	vecStore := vectors.NewStore(db)
	merged := ontology.MergedRelations(cfg.Ontology.Relations)
	ontStore := ontology.NewStore(db, ontology.ValidRelationNames(merged))
	embedder := embed.NewFromConfig(cfg)

	// Pass 2: Concept extraction
	extractModel := cfg.Models.Extract
	if extractModel == "" {
		extractModel = cfg.Models.Summarize
	}

	log.Info("Pass 2: extracting concepts", "from_summaries", len(summaries))
	concepts, err := ExtractConcepts(summaries, mf.Concepts, client, extractModel)
	if err != nil {
		return nil, fmt.Errorf("re-extract: concept extraction: %w", err)
	}
	result.ConceptsExtracted = len(concepts)

	// Update manifest with concepts
	for _, c := range concepts {
		mf.AddConcept(c.Name, filepath.Join(cfg.Output, "concepts", c.Name+".md"), c.Sources)
	}

	// Pass 3: Write articles
	if len(concepts) > 0 {
		writeModel := cfg.Models.Write
		if writeModel == "" {
			writeModel = extractModel
		}
		articleMaxTokens := cfg.Compiler.ArticleMaxTokens
		if articleMaxTokens <= 0 {
			articleMaxTokens = 4000
		}

		relPatterns := ontology.RelationPatterns(merged)
		log.Info("Pass 3: writing articles", "concepts", len(concepts))
		chunkTokens := embed.GetChunkTokens(cfg)
		articles := WriteArticles(projectDir, cfg.Output, concepts, client, writeModel, articleMaxTokens, cfg.Compiler.MaxParallel, memStore, vecStore, ontStore, embedder, cfg.Compiler.UserTimeLocation(), cfg.Compiler.ArticleFields, relPatterns, chunkTokens)

		for _, ar := range articles {
			if ar.Error != nil {
				result.Errors++
			} else {
				result.ArticlesWritten++
			}
		}
	}

	// Save manifest
	if err := mf.Save(filepath.Join(projectDir, ".manifest.json")); err != nil {
		return nil, fmt.Errorf("re-extract: save manifest: %w", err)
	}

	log.Info("re-extract complete", "concepts", result.ConceptsExtracted, "articles", result.ArticlesWritten, "errors", result.Errors)
	return result, nil
}

// ReWrite re-writes only failed articles from previous --re-extract run.
// It reads the manifest to find all concepts, checks which articles exist,
// and re-writes only those that are missing or failed.
func ReWrite(projectDir string) (*CompileResult, error) {
	result := &CompileResult{}

	cfg, err := config.Load(filepath.Join(projectDir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("re-write: load config: %w", err)
	}

	// Load user prompt overrides if prompts/ directory exists
	if err := prompts.LoadFromDir(filepath.Join(projectDir, "prompts")); err != nil {
		log.Warn("failed to load custom prompts", "error", err)
	}

	mf, err := manifest.Load(filepath.Join(projectDir, ".manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("re-write: load manifest: %w", err)
	}

	// Get all concepts from manifest
	concepts := mf.Concepts
	if len(concepts) == 0 {
		return result, fmt.Errorf("re-write: no concepts found in manifest — run sage-wiki compile first")
	}

	// Filter to only concepts that need re-writing (article file doesn't exist)
	// manifest.Concepts is a map[string]Concept, key is the concept name
	var conceptsToWrite []string
	for name := range concepts {
		articlePath := filepath.Join(projectDir, cfg.Output, "concepts", name+".md")
		if _, err := os.Stat(articlePath); os.IsNotExist(err) {
			conceptsToWrite = append(conceptsToWrite, name)
		}
	}

	if len(conceptsToWrite) == 0 {
		log.Info("re-write: all articles already exist, nothing to do")
		return result, nil
	}

	log.Info("re-write: found concepts needing articles", "count", len(conceptsToWrite))

	// Create LLM client
	client, err := llm.NewClient(cfg.API.Provider, cfg.API.APIKey, cfg.API.BaseURL, cfg.API.RateLimit, cfg.API.TimeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("re-write: create LLM client: %w", err)
	}

	// Open DB
	db, err := storage.Open(filepath.Join(projectDir, ".sage", "wiki.db"))
	if err != nil {
		return nil, fmt.Errorf("re-write: open db: %w", err)
	}
	defer db.Close()

	memStore := memory.NewStore(db)
	vecStore := vectors.NewStore(db)
	merged := ontology.MergedRelations(cfg.Ontology.Relations)
	ontStore := ontology.NewStore(db, ontology.ValidRelationNames(merged))
	embedder := embed.NewFromConfig(cfg)

	// Convert concept names to ExtractedConcept
	extractedConcepts := make([]ExtractedConcept, len(conceptsToWrite))
	for i, name := range conceptsToWrite {
		c := concepts[name]
		extractedConcepts[i] = ExtractedConcept{
			Name:    name,
			Sources: c.Sources,
		}
	}

	// Write articles
	writeModel := cfg.Models.Write
	if writeModel == "" {
		writeModel = cfg.Models.Summarize
	}
	articleMaxTokens := cfg.Compiler.ArticleMaxTokens
	if articleMaxTokens <= 0 {
		articleMaxTokens = 4000
	}

	relPatterns := ontology.RelationPatterns(merged)
	log.Info("re-write: writing missing articles", "concepts", len(extractedConcepts))
	chunkTokens := embed.GetChunkTokens(cfg)
	articles := WriteArticles(projectDir, cfg.Output, extractedConcepts, client, writeModel, articleMaxTokens, cfg.Compiler.MaxParallel, memStore, vecStore, ontStore, embedder, cfg.Compiler.UserTimeLocation(), cfg.Compiler.ArticleFields, relPatterns, chunkTokens)

	for _, ar := range articles {
		if ar.Error != nil {
			result.Errors++
		} else {
			result.ArticlesWritten++
		}
	}

	log.Info("re-write complete", "articles", result.ArticlesWritten, "errors", result.Errors)
	return result, nil
}
