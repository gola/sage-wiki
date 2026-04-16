package compiler

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/xoai/sage-wiki/internal/embed"
	"github.com/xoai/sage-wiki/internal/extract"
	"github.com/xoai/sage-wiki/internal/extract/parsers"
	"github.com/xoai/sage-wiki/internal/llm"
	"github.com/xoai/sage-wiki/internal/log"
	"github.com/xoai/sage-wiki/internal/memory"
	"github.com/xoai/sage-wiki/internal/vectors"
)

// parserRegistry is the shared parser registry for code file parsing.
var parserRegistry = parsers.NewRegistry()

// indexRawSources indexes source files into FTS5 at Tier 0 (no embedding).
// Uses "src:" prefix on entry IDs to distinguish from compiled article entries.
// Skips sources that already have a compiled entry (non-prefixed) in FTS5.
func indexRawSources(projectDir string, sources []CompileItem, memStore *memory.Store, items *CompileItemStore) int {
	indexed := 0
	for _, src := range sources {
		// Skip if a compiled entry already exists (higher quality)
		if existing, _ := memStore.Get(src.SourcePath); existing != nil {
			if err := items.MarkPass(src.SourcePath, "indexed"); err != nil {
				log.Warn("mark pass failed", "path", src.SourcePath, "pass", "indexed", "error", err)
			}
			indexed++
			continue
		}

		absPath := filepath.Join(projectDir, src.SourcePath)
		content, err := extract.Extract(absPath, src.FileType)
		if err != nil {
			log.Warn("tier 0 index: extract failed", "path", src.SourcePath, "error", err)
			if merr := items.MarkError(src.SourcePath, err); merr != nil {
				log.Warn("mark error failed", "path", src.SourcePath, "error", merr)
			}
			continue
		}

		// Parse code structure if supported
		entryContent := content.Text
		tags := []string{src.FileType, "tier:0"}
		ext := strings.TrimPrefix(filepath.Ext(src.SourcePath), ".")
		if parserRegistry.Supports(ext) {
			if pr, perr := parserRegistry.Parse(src.SourcePath, []byte(content.Text)); perr == nil && pr != nil {
				entryContent = content.Text + "\n\n---\nStructure:\n" + pr.Structure
				tags = append(tags, "parsed")
			}
		}

		memStore.Add(memory.Entry{
			ID:      "src:" + src.SourcePath,
			Content: entryContent,
			Tags:    tags,
		})

		if err := items.MarkPass(src.SourcePath, "indexed"); err != nil {
			log.Warn("mark pass failed", "path", src.SourcePath, "pass", "indexed", "error", err)
		}
		indexed++
	}
	return indexed
}

// indexAndEmbedSources indexes + embeds sources at Tier 1.
// FTS5 indexing is synchronous; embedding uses BackpressureController for
// API rate limiting. Sources already indexed skip the FTS5 step.
func indexAndEmbedSources(
	projectDir string,
	sources []CompileItem,
	memStore *memory.Store,
	vecStore *vectors.Store,
	embedder embed.Embedder,
	items *CompileItemStore,
	bp *BackpressureController,
) (indexed, embedded int) {
	// Step 1: FTS5 index any sources not yet indexed
	for _, src := range sources {
		if src.PassIndexed {
			continue
		}

		// Skip if a compiled entry already exists
		if existing, _ := memStore.Get(src.SourcePath); existing != nil {
			if err := items.MarkPass(src.SourcePath, "indexed"); err != nil {
				log.Warn("mark pass failed", "path", src.SourcePath, "pass", "indexed", "error", err)
			}
			indexed++
			continue
		}

		absPath := filepath.Join(projectDir, src.SourcePath)
		content, err := extract.Extract(absPath, src.FileType)
		if err != nil {
			log.Warn("tier 1 index: extract failed", "path", src.SourcePath, "error", err)
			if merr := items.MarkError(src.SourcePath, err); merr != nil {
				log.Warn("mark error failed", "path", src.SourcePath, "error", merr)
			}
			continue
		}

		// Parse code structure if supported
		entryContent := content.Text
		tags := []string{src.FileType, "tier:1"}
		ext := strings.TrimPrefix(filepath.Ext(src.SourcePath), ".")
		if parserRegistry.Supports(ext) {
			if pr, perr := parserRegistry.Parse(src.SourcePath, []byte(content.Text)); perr == nil && pr != nil {
				entryContent = content.Text + "\n\n---\nStructure:\n" + pr.Structure
				tags = append(tags, "parsed")
				if err := items.MarkPass(src.SourcePath, "parsed"); err != nil {
					log.Warn("mark pass failed", "path", src.SourcePath, "pass", "parsed", "error", err)
				}
			}
		}

		memStore.Add(memory.Entry{
			ID:      "src:" + src.SourcePath,
			Content: entryContent,
			Tags:    tags,
		})

		if err := items.MarkPass(src.SourcePath, "indexed"); err != nil {
			log.Warn("mark pass failed", "path", src.SourcePath, "pass", "indexed", "error", err)
		}
		indexed++
	}

	// Step 2: Embed (parallel via BackpressureController or fixed semaphore)
	if embedder == nil {
		return indexed, 0
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	embeddedCount := 0

	for _, src := range sources {
		if src.PassEmbedded {
			continue
		}

		wg.Add(1)

		var release func()
		if bp != nil {
			release = bp.Acquire()
		} else {
			release = func() {}
		}

		go func(s CompileItem) {
			defer wg.Done()
			defer release()

			absPath := filepath.Join(projectDir, s.SourcePath)
			content, err := extract.Extract(absPath, s.FileType)
			if err != nil {
				log.Warn("tier 1 embed: extract failed", "path", s.SourcePath, "error", err)
				if merr := items.MarkError(s.SourcePath, err); merr != nil {
					log.Warn("mark error failed", "path", s.SourcePath, "error", merr)
				}
				return
			}

			vec, err := embedder.Embed(content.Text)
			if err != nil {
				if bp != nil && llm.IsRateLimitError(err) {
					delay := bp.OnRateLimit()
					log.Warn("embedding rate limited", "delay", delay)
				}
				log.Warn("tier 1 embed: embedding failed", "path", s.SourcePath, "error", err)
				if merr := items.MarkError(s.SourcePath, err); merr != nil {
					log.Warn("mark error failed", "path", s.SourcePath, "error", merr)
				}
				return
			}

			if bp != nil {
				bp.OnSuccess()
			}

			vecStore.Upsert("src:"+s.SourcePath, vec)
			if err := items.MarkPass(s.SourcePath, "embedded"); err != nil {
				log.Warn("mark pass failed", "path", s.SourcePath, "pass", "embedded", "error", err)
			}

			mu.Lock()
			embeddedCount++
			mu.Unlock()
		}(src)
	}

	wg.Wait()
	return indexed, embeddedCount
}
