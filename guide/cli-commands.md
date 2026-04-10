# CLI 命令入口

本文档深入解析 `cmd/sage-wiki/main.go`，sage-wiki 的命令行入口。

## 快速索引

- `rootCmd`: 根命令，定义全局参数
- `initCmd`: 初始化新项目
- `compileCmd`: 编译文档到wiki
- `serveCmd`: 启动MCP服务器
- `searchCmd`: 搜索wiki
- `queryCmd`: 问答查询
- `statusCmd`: 显示项目状态
- `lintCmd`: 运行lint检查
- `ingestCmd`: 摄入新文档

## 命令结构

```go
// 根命令
var rootCmd = &cobra.Command{
    Use:   "sage-wiki",
    Short: "LLM-compiled personal knowledge base",
}

// 子命令注册
func init() {
    rootCmd.AddCommand(initCmd)
    rootCmd.AddCommand(compileCmd)
    rootCmd.AddCommand(serveCmd)
    rootCmd.AddCommand(searchCmd)
    rootCmd.AddCommand(queryCmd)
    rootCmd.AddCommand(statusCmd)
    rootCmd.AddCommand(lintCmd)
    rootCmd.AddCommand(ingestCmd)
    rootCmd.AddCommand(tuiCmd)
}
```

## 全局参数

```go
var (
    projectDir string   // 项目目录，默认当前目录
    configPath string   // 配置文件路径
    verbosity  int      // 日志级别 (0-2)
)
```

## 核心命令详解

### init - 项目初始化

```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize a new sage-wiki project",
    RunE:  runInit,
}
```

**功能**：
- 创建项目目录结构 (`raw/`, `wiki/`, `.sage/`)
- 生成默认 `config.yaml`
- 初始化 `.manifest.json`

**选项**：
- `--vault`: Obsidian vault覆盖模式

### compile - 编译文档

```go
var compileCmd = &cobra.Command{
    Use:   "compile",
    Short: "Compile sources into wiki articles",
    RunE:  runCompile,
}
```

**功能**：
- 执行编译流水线（diff → summarize → concepts → write）
- 支持增量编译和检查点恢复

**选项**：
- `--watch`: 监听文件变化，自动重新编译
- `--fresh`: 忽略检查点，从头开始
- `--batch`: 使用批量API（异步，50%折扣）
- `--dry-run`: 仅显示变更，不执行

### serve - MCP服务器

```go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start MCP server",
    RunE:  runServe,
}
```

**功能**：
- 启动MCP协议服务器
- 暴露工具接口供LLM代理调用

**选项**：
- `--ui`: 同时启动Web UI
- `--port`: HTTP端口（默认3333）

### search - 搜索wiki

```go
var searchCmd = &cobra.Command{
    Use:   "search [query]",
    Short: "Search the wiki",
    Args:  cobra.MinimumNArgs(1),
    RunE:  runSearch,
}
```

**功能**：
- 执行混合搜索（BM25 + 语义）
- 返回相关文档列表

**选项**：
- `--limit`: 结果数量限制
- `--format`: 输出格式（text/json）

### query - 问答查询

```go
var queryCmd = &cobra.Command{
    Use:   "query [question]",
    Short: "Ask a question against the wiki",
    Args:  cobra.MinimumNArgs(1),
    RunE:  runQuery,
}
```

**功能**：
- 自然语言问答
- 基于检索增强生成（RAG）

## 命令执行流程

```
main()
    │
    ▼
rootCmd.Execute()
    │
    ├── init ──────────▶ wiki.Init(projectDir)
    │
    ├── compile ───────▶ compiler.Compile(projectDir, opts)
    │
    ├── serve ─────────▶ mcp.NewServer(projectDir).Serve()
    │
    ├── search ────────▶ hybrid.NewSearcher().Search(query)
    │
    ├── query ─────────▶ query.Run(projectDir, question)
    │
    └── status ────────▶ wiki.Status(projectDir)
```

## 配置加载

每个命令执行时都会：

1. 解析全局参数（`--project-dir`, `--config`, `-v`）
2. 加载配置文件 `config.yaml`
3. 初始化日志级别
4. 执行命令逻辑

```go
PersistentPreRun: func(cmd *cobra.Command, args []string) {
    log.SetVerbosity(verbosity)
},
```

## 设计要点

1. **Cobra框架**：使用spf13/cobra实现子命令和参数解析
2. **错误处理**：使用`RunE`返回错误，Cobra自动处理退出码
3. **全局参数**：使用`PersistentFlags`在所有子命令中可用
4. **最小参数检查**：使用`cobra.MinimumNArgs`验证参数数量
