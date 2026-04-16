# Model Quality Benchmark Suite

Run benchmarks to compare model quality across passes:

```bash
# Run with default config (uses configured models)
go test ./internal/compiler/ -run TestBenchmarkQuality -v -timeout 30m

# Compare specific models
SAGE_BENCH_SUMMARIZE=ollama/qwen2.5:7b SAGE_BENCH_WRITE=gpt-4o go test ./internal/compiler/ -run TestBenchmarkQuality -v
```

## Document Categories

Place test documents in these subdirectories:

- `prose/` — 20 markdown/text documents (articles, docs, README files)
- `code/` — 15 source code files (Go, Python, TypeScript, etc.)
- `structured/` — 15 structured data files (JSON, YAML, config files)

## Metrics

The benchmark measures per-pass:
- **Source coverage**: % of source key phrases found in output
- **Extraction completeness**: % of concepts that get articles
- **Hallucination rate**: claims in output not traceable to source (manual review)

## Results

Results are written to `benchmark-results.json` after each run.
Compare across model configurations to find the cost/quality sweet spot.
