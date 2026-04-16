package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xoai/sage-wiki/internal/config"
	"github.com/xoai/sage-wiki/internal/manifest"
)

func TestMigrateCheckpoint(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	projectDir := t.TempDir()
	sageDir := filepath.Join(projectDir, ".sage")
	os.MkdirAll(sageDir, 0755)

	// Create a compile-state.json with mixed state
	state := CompileState{
		CompileID: "20260414-120000",
		StartedAt: "2026-04-14T12:00:00Z",
		Pass:      1,
		Completed: []string{"raw/a.md", "raw/b.md"},
		Pending:   []string{"raw/c.md"},
		Failed: []FailedSource{
			{Path: "raw/d.md", Error: "rate limited", Attempts: 3},
		},
	}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(sageDir, "compile-state.json"), data, 0644)

	// Create a manifest with sources
	mf := manifest.New()
	mf.AddSource("raw/a.md", "sha256:aaa", "article", 1000)
	mf.MarkCompiled("raw/a.md", "wiki/summaries/a.md", []string{"concept-a"})
	mf.AddSource("raw/b.md", "sha256:bbb", "article", 2000)
	mf.MarkCompiled("raw/b.md", "wiki/summaries/b.md", []string{"concept-b"})
	mf.AddSource("raw/c.md", "sha256:ccc", "article", 3000)
	// c.md is pending (not compiled)
	mf.AddSource("raw/d.md", "sha256:ddd", "article", 500)
	// d.md failed

	cfg := &config.Config{
		Compiler: config.CompilerConfig{DefaultTier: 1},
	}

	migrated, err := MigrateCheckpoint(projectDir, db, mf, cfg)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !migrated {
		t.Fatal("expected migration to occur")
	}

	items := NewCompileItemStore(db)

	// Verify compiled sources (a.md and b.md)
	a, _ := items.GetByPath("raw/a.md")
	if a == nil {
		t.Fatal("expected raw/a.md in compile_items")
	}
	if a.Tier != 3 {
		t.Errorf("a.md tier = %d, want 3 (compiled)", a.Tier)
	}
	if !a.PassSummarized || !a.PassWritten {
		t.Error("a.md should have all passes complete (compiled status)")
	}

	// Verify pending source (c.md) — in checkpoint pending list
	c, _ := items.GetByPath("raw/c.md")
	if c == nil {
		t.Fatal("expected raw/c.md in compile_items")
	}
	if c.PassSummarized {
		t.Error("c.md should not have pass_summarized (pending)")
	}

	// Verify failed source (d.md) — has error
	d, _ := items.GetByPath("raw/d.md")
	if d == nil {
		t.Fatal("expected raw/d.md in compile_items")
	}
	if d.Error != "rate limited" {
		t.Errorf("d.md error = %q, want 'rate limited'", d.Error)
	}
	if d.ErrorCount != 3 {
		t.Errorf("d.md error_count = %d, want 3", d.ErrorCount)
	}

	// Verify compile-state.json was deleted
	if _, err := os.Stat(filepath.Join(sageDir, "compile-state.json")); !os.IsNotExist(err) {
		t.Error("compile-state.json should be deleted after migration")
	}

	// Verify total count
	count, _ := items.Count()
	if count != 4 {
		t.Errorf("total items = %d, want 4", count)
	}
}

func TestMigrateCheckpoint_NoFile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	projectDir := t.TempDir()
	mf := manifest.New()
	cfg := &config.Config{}

	migrated, err := MigrateCheckpoint(projectDir, db, mf, cfg)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated {
		t.Error("should return false when no checkpoint exists")
	}
}

func TestMigrateCheckpoint_BatchInFlight(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	projectDir := t.TempDir()
	sageDir := filepath.Join(projectDir, ".sage")
	os.MkdirAll(sageDir, 0755)

	// Checkpoint with batch in flight
	state := CompileState{
		CompileID: "20260414-120000",
		Pass:      1,
		Batch:     &BatchState{BatchID: "batch_abc", Provider: "anthropic"},
	}
	data, _ := json.Marshal(state)
	os.WriteFile(filepath.Join(sageDir, "compile-state.json"), data, 0644)

	mf := manifest.New()
	cfg := &config.Config{}

	migrated, err := MigrateCheckpoint(projectDir, db, mf, cfg)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if migrated {
		t.Error("should skip migration when batch is in flight")
	}

	// Verify compile-state.json still exists
	if _, err := os.Stat(filepath.Join(sageDir, "compile-state.json")); os.IsNotExist(err) {
		t.Error("compile-state.json should NOT be deleted when batch is in flight")
	}
}

func TestPopulateFromManifest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	mf := manifest.New()
	mf.AddSource("raw/a.md", "sha256:aaa", "article", 1000)
	mf.MarkCompiled("raw/a.md", "wiki/summaries/a.md", []string{"concept-a"})
	mf.AddSource("raw/b.json", "sha256:bbb", "data", 500)

	cfg := &config.Config{
		Compiler: config.CompilerConfig{
			DefaultTier: 1,
			TierDefaults: map[string]int{
				"json": 0,
			},
		},
	}

	populated, err := PopulateFromManifest(db, mf, cfg)
	if err != nil {
		t.Fatalf("populate: %v", err)
	}
	if populated != 2 {
		t.Errorf("populated = %d, want 2", populated)
	}

	items := NewCompileItemStore(db)

	// Compiled source should be Tier 3
	a, _ := items.GetByPath("raw/a.md")
	if a.Tier != 3 {
		t.Errorf("a.md tier = %d, want 3", a.Tier)
	}
	if !a.PassWritten {
		t.Error("a.md should have pass_written=true")
	}

	// JSON source should use tier_defaults
	b, _ := items.GetByPath("raw/b.json")
	if b.Tier != 0 {
		t.Errorf("b.json tier = %d, want 0 (from tier_defaults)", b.Tier)
	}

	// Re-running should not duplicate
	populated2, _ := PopulateFromManifest(db, mf, cfg)
	if populated2 != 0 {
		t.Errorf("second populate = %d, want 0 (already exists)", populated2)
	}
}
