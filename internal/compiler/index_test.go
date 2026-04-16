package compiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xoai/sage-wiki/internal/memory"
)

func TestIndexRawSources_SkipsCompiledEntry(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	memStore := memory.NewStore(db)
	items := NewCompileItemStore(db)

	// Create a temporary source file
	projectDir := t.TempDir()
	srcPath := "raw/existing.md"
	os.MkdirAll(filepath.Join(projectDir, "raw"), 0755)
	os.WriteFile(filepath.Join(projectDir, srcPath), []byte("# Existing Content\n\nThis source has been compiled."), 0644)

	// Pre-add a compiled entry (non-prefixed ID = compiled article entry)
	memStore.Add(memory.Entry{
		ID:      srcPath, // no "src:" prefix — this is a compiled entry
		Content: "Compiled summary of existing content.",
		Tags:    []string{"md"},
	})

	// Create compile item at Tier 0
	items.Upsert(CompileItem{
		SourcePath: srcPath,
		Hash:       "abc123",
		FileType:   "md",
		Tier:       0,
		SourceType: "compiler",
	})

	sources := []CompileItem{
		{SourcePath: srcPath, FileType: "md"},
	}

	// indexRawSources should skip adding a "src:" entry because a compiled entry exists
	indexed := indexRawSources(projectDir, sources, memStore, items)
	if indexed != 1 {
		t.Errorf("indexed = %d, want 1 (should count as indexed even when skipped)", indexed)
	}

	// Verify that the "src:" prefixed entry was NOT added
	srcEntry, _ := memStore.Get("src:" + srcPath)
	if srcEntry != nil {
		t.Error("expected no 'src:' entry when compiled entry exists, but found one")
	}

	// Verify the original compiled entry still exists
	compiledEntry, _ := memStore.Get(srcPath)
	if compiledEntry == nil {
		t.Error("compiled entry should still exist")
	}

	// Verify pass was marked
	item, _ := items.GetByPath(srcPath)
	if item == nil {
		t.Fatal("compile item should exist")
	}
	if !item.PassIndexed {
		t.Error("pass_indexed should be true after indexRawSources")
	}
}

func TestIndexRawSources_IndexesNewSource(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	memStore := memory.NewStore(db)
	items := NewCompileItemStore(db)

	// Create a source file with no compiled entry
	projectDir := t.TempDir()
	srcPath := "raw/new.md"
	os.MkdirAll(filepath.Join(projectDir, "raw"), 0755)
	os.WriteFile(filepath.Join(projectDir, srcPath), []byte("# New Content\n\nThis is new and uncompiled."), 0644)

	items.Upsert(CompileItem{
		SourcePath: srcPath,
		Hash:       "def456",
		FileType:   "md",
		Tier:       0,
		SourceType: "compiler",
	})

	sources := []CompileItem{
		{SourcePath: srcPath, FileType: "md"},
	}

	indexed := indexRawSources(projectDir, sources, memStore, items)
	if indexed != 1 {
		t.Errorf("indexed = %d, want 1", indexed)
	}

	// Verify the "src:" entry WAS added
	srcEntry, _ := memStore.Get("src:" + srcPath)
	if srcEntry == nil {
		t.Error("expected 'src:' entry for new source")
	}
}
