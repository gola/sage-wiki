# 编译器流水线

本文档深入解析 `internal/compiler/pipeline.go` 及相关模块，这是sage-wiki文档处理的核心。

## 快速索引

- `Compile()`: 主编译入口
- `CompileOpts`: 编译选项配置
- `CompileResult`: 编译结果统计
- `CompileState`: 检查点状态

## 编译流程概览

```
Compile(projectDir, opts)
    │
    ├─▶ Pass 0: Diff (检测变更)
    │       └── diff.go
    │
    ├─▶ Pass 1: Summarize (生成摘要)
    │       └── summarize.go
    │
    ├─▶ Pass 2: Extract Concepts (提取概念)
    │       └── concepts.go
    │
    └─▶ Pass 3: Write Articles (写入文章)
            └── write.go
```

## 核心类型

### CompileOpts - 编译选项

```go
type CompileOpts struct {
    DryRun   bool         // 仅显示变更，不执行
    Fresh    bool         // 忽略检查点，从头开始
    Batch    bool         // 使用批量API（异步，50%折扣）
    NoCache  bool         // 禁用提示缓存
    Tracker  *llm.CostTracker // 可选的成本追踪器
}
```

### CompileResult - 编译结果

```go
type CompileResult struct {
    Added              int  // 新增文档数
    Modified           int  // 修改文档数
    Removed            int  // 删除文档数
    Summarized         int  // 生成摘要数
    ConceptsExtracted  int  // 提取概念数
    ArticlesWritten    int  // 写入文章数
    Errors             int  // 错误数
    CostReport         *llm.CostReport // LLM成本报告
}
```

### CompileState - 检查点状态

```go
type CompileState struct {
    CompileID  string          // 编译会话ID
    StartedAt  string          // 开始时间
    Pass       int             // 当前阶段
    Completed  []string        // 已完成的文件
    Pending    []string        // 待处理的文件
    Failed     []FailedSource  // 失败的文件
    Batch      *BatchState     // 批量任务状态
}
```

## Pass 0: Diff - 变更检测

**文件**: `internal/compiler/diff.go`

**功能**: 比较源文件与manifest，检测新增、修改、删除的文件。

```go
type DiffResult struct {
    Added    []string  // 新增文件
    Modified []string  // 修改文件（内容hash变化）
    Removed  []string  // 删除文件
}
```

**检测逻辑**:
1. 扫描 `raw/` 目录所有文件
2. 计算每个文件的内容hash
3. 与 `.manifest.json` 中的记录对比
4. 输出变更列表

## Pass 1: Summarize - 摘要生成

**文件**: `internal/compiler/summarize.go`

**功能**: 使用LLM为每个文档生成结构化摘要。

**处理流程**:
```
源文件
    │
    ▼
ExtractText() (internal/extract/)
    │
    ▼
LLM Summarize (prompts/summarize_article.txt)
    │
    ▼
Memory Store (存储摘要)
```

**摘要结构**:
```yaml
title: 文档标题
summary: 一段话摘要
key_points:
  - 要点1
  - 要点2
tags: [tag1, tag2]
```

## Pass 2: Extract Concepts - 概念提取

**文件**: `internal/compiler/concepts.go`

**功能**: 从摘要中提取概念并建立关系。

**处理流程**:
```
文档摘要
    │
    ▼
LLM Extract Concepts (prompts/extract_concepts.txt)
    │
    ▼
Ontology Store (存储概念和关系)
```

**概念结构**:
```json
{
  "name": "概念名称",
  "definition": "概念定义",
  "related_to": ["相关概念1", "相关概念2"]
}
```

## Pass 3: Write Articles - 文章写入

**文件**: `internal/compiler/write.go`

**功能**: 生成wiki文章并添加交叉引用。

**处理流程**:
```
摘要 + 概念
    │
    ▼
LLM Write Article (prompts/write_article.txt)
    │
    ▼
Wiki Article (Markdown)
```

**文章结构**:
```markdown
# 标题

> 来源: [[source-file.pdf]]

## 摘要

...

## 关键概念

- [[Concept1]]: 描述
- [[Concept2]]: 描述

## 相关文章

- [[article-1]]
- [[article-2]]
```

## 检查点与恢复

编译过程支持断点续传：

```go
// 保存状态
statePath := filepath.Join(projectDir, ".sage", "compile-state.json")
saveCompileState(statePath, state)

// 恢复状态
if !opts.Fresh {
    state, _ = loadCompileState(statePath)
}
```

**恢复逻辑**:
1. 加载 `.sage/compile-state.json`
2. 跳过已完成的文件
3. 从上次中断处继续

## 批量API支持

当 `opts.Batch = true` 时：

```go
type BatchState struct {
    BatchID     string // 批量任务ID
    Provider    string // 提供商名称
    Pass        string // 当前阶段
    ResultsRef  string // 结果引用URL
    SubmittedAt string // 提交时间
}
```

**优势**:
- 异步处理，不阻塞
- 50%成本折扣（Anthropic/OpenAI）
- 适合大批量文档处理

## 监听模式

**文件**: `internal/compiler/watch.go`

```go
func Watch(projectDir string, opts CompileOpts) error {
    // 使用fsnotify监听文件变化
    watcher, _ := fsnotify.NewWatcher()
    
    // 检测到变化时自动编译
    for event := range watcher.Events {
        if event.Op&fsnotify.Write == fsnotify.Write {
            Compile(projectDir, opts)
        }
    }
}
```

## 设计要点

1. **增量处理**: 只处理变更的文件，避免重复计算
2. **检查点恢复**: 支持断点续传，适合长时间编译
3. **批量优化**: 支持批量API降低成本
4. **监听模式**: 开发时实时编译
5. **成本追踪**: 可选的LLM成本统计
