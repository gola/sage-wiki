# sage-wiki 架构地图

本文档描述项目的分层结构、主要执行链路和推荐阅读路线。

## 项目分层

```
sage-wiki/
├── cmd/sage-wiki/          # CLI入口层
│   └── main.go             # Cobra命令定义和路由
│
├── internal/
│   ├── compiler/           # 编译器核心（文档处理流水线）
│   │   ├── pipeline.go     # 主编译流程
│   │   ├── diff.go         # 增量检测
│   │   ├── summarize.go    # 摘要生成
│   │   ├── concepts.go     # 概念提取
│   │   └── write.go        # 文章写入
│   │
│   ├── llm/                # LLM抽象层
│   │   ├── client.go       # 统一客户端接口
│   │   ├── anthropic.go    # Claude支持
│   │   ├── openai.go       # GPT支持
│   │   ├── gemini.go       # Gemini支持
│   │   └── cache.go        # 提示缓存
│   │
│   ├── storage/            # 持久化层
│   │   └── db.go           # SQLite连接管理
│   │
│   ├── vectors/            # 向量存储
│   │   └── store.go        # 嵌入向量索引
│   │
│   ├── memory/             # 文档记忆
│   │   └── entries.go      # 条目CRUD
│   │
│   ├── embed/              # 嵌入生成
│   │   └── embed.go        # 文本向量化
│   │
│   ├── extract/            # 文档提取
│   │   ├── pdf.go          # PDF解析
│   │   ├── office.go       # Word/Excel/PPT
│   │   ├── epub.go         # EPUB解析
│   │   └── email.go        # 邮件解析
│   │
│   ├── hybrid/             # 混合搜索
│   │   └── search.go       # BM25 + 语义搜索
│   │
│   ├── query/              # 问答系统
│   │   └── query.go        # 自然语言问答
│   │
│   ├── mcp/                # MCP服务器
│   │   ├── server.go       # 服务器主逻辑
│   │   ├── tools_write.go  # 写入工具
│   │   └── tools_compound.go # 复合工具
│   │
│   ├── ontology/           # 本体管理
│   │   └── ontology.go     # 概念关系
│   │
│   ├── prompts/            # 提示模板
│   │   └── prompts.go      # 模板加载
│   │
│   ├── config/             # 配置管理
│   │   └── config.go       # YAML配置
│   │
│   ├── wiki/               # Wiki操作
│   │   ├── init.go         # 项目初始化
│   │   ├── ingest.go       # 文档摄入
│   │   └── status.go       # 状态查询
│   │
│   ├── tui/                # 终端UI
│   │   ├── dashboard/      # 仪表盘
│   │   ├── compile/        # 编译视图
│   │   └── query/          # 问答视图
│   │
│   └── web/                # Web服务器
│       └── server.go       # HTTP API
│
└── web/                    # 前端（Preact）
    └── src/
        ├── components/     # UI组件
        └── lib/            # API客户端
```

## 主要执行链路

### 1. 编译流程 (sage-wiki compile)

```
用户文档 (raw/)
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│  Pass 0: Diff (internal/compiler/diff.go)              │
│  - 检测新增/修改/删除的文件                              │
│  - 与 .manifest.json 对比                               │
└─────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│  Pass 1: Summarize (internal/compiler/summarize.go)    │
│  - 提取文档文本 (internal/extract/)                     │
│  - 调用LLM生成摘要                                      │
│  - 存储到 memory store                                  │
└─────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│  Pass 2: Extract Concepts (internal/compiler/concepts) │
│  - 从摘要中提取概念                                      │
│  - 建立概念间关系                                        │
│  - 存储到 ontology store                                 │
└─────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│  Pass 3: Write Articles (internal/compiler/write.go)   │
│  - 生成wiki文章                                          │
│  - 添加交叉引用                                          │
│  - 写入 wiki/ 目录                                       │
└─────────────────────────────────────────────────────────┘
    │
    ▼
wiki/ (Markdown文章)
```

### 2. 搜索流程 (sage-wiki search)

```
查询字符串
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│  Hybrid Searcher (internal/hybrid/search.go)           │
│  ┌─────────────────┐    ┌─────────────────┐            │
│  │  BM25 Search    │    │  Vector Search  │            │
│  │  (关键词匹配)    │    │  (语义相似度)    │            │
│  └────────┬────────┘    └────────┬────────┘            │
│           └──────────┬──────────┘                      │
│                      ▼                                  │
│              结果合并与重排序                            │
└─────────────────────────────────────────────────────────┘
    │
    ▼
搜索结果 (带分数的文档列表)
```

### 3. MCP服务流程 (sage-wiki serve)

```
MCP客户端请求
    │
    ▼
┌─────────────────────────────────────────────────────────┐
│  MCP Server (internal/mcp/server.go)                   │
│  - 注册工具: read_entry, search, query, write_entry... │
│  - 处理工具调用                                         │
│  - 返回结构化响应                                       │
└─────────────────────────────────────────────────────────┘
    │
    ▼
调用内部模块 (memory, vectors, hybrid, query)
```

## 核心数据流

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│  Config  │────▶│ Compiler │────▶│  Memory  │
│ (YAML)   │     │ Pipeline  │     │  Store   │
└──────────┘     └──────────┘     └──────────┘
                      │                │
                      ▼                ▼
                ┌──────────┐     ┌──────────┐
                │   LLM    │     │  Vectors │
                │  Client  │     │  Store   │
                └──────────┘     └──────────┘
                      │                │
                      ▼                ▼
                ┌──────────┐     ┌──────────┐
                │ Prompts  │     │ Ontology │
                │ Templates│     │  Store   │
                └──────────┘     └──────────┘
                                       │
                                       ▼
                                 ┌──────────┐
                                 │   Wiki   │
                                 │  Articles│
                                 └──────────┘
```

## 推荐阅读路线

### 路线A：理解编译流程

1. [`cmd/sage-wiki/main.go`](../cmd/sage-wiki/main.go) - CLI入口
2. [`internal/compiler/pipeline.go`](../internal/compiler/pipeline.go) - 编译主流程
3. [`internal/compiler/diff.go`](../internal/compiler/diff.go) - 增量检测
4. [`internal/compiler/summarize.go`](../internal/compiler/summarize.go) - 摘要生成
5. [`internal/extract/`](../internal/extract/) - 文档提取

### 路线B：理解LLM集成

1. [`internal/llm/client.go`](../internal/llm/client.go) - 客户端抽象
2. [`internal/llm/anthropic.go`](../internal/llm/anthropic.go) - Claude实现
3. [`internal/llm/cache.go`](../internal/llm/cache.go) - 提示缓存
4. [`internal/prompts/`](../internal/prompts/) - 提示模板

### 路线C：理解存储和搜索

1. [`internal/storage/db.go`](../internal/storage/db.go) - SQLite管理
2. [`internal/memory/entries.go`](../internal/memory/entries.go) - 文档存储
3. [`internal/vectors/store.go`](../internal/vectors/store.go) - 向量索引
4. [`internal/hybrid/search.go`](../internal/hybrid/search.go) - 混合搜索

### 路线D：理解MCP接口

1. [`internal/mcp/server.go`](../internal/mcp/server.go) - 服务器主逻辑
2. [`internal/mcp/tools_write.go`](../internal/mcp/tools_write.go) - 写入工具
3. [`internal/mcp/tools_compound.go`](../internal/mcp/tools_compound.go) - 复合工具

## 关键设计决策

1. **纯Go实现**：无CGO依赖，单一二进制文件部署
2. **SQLite + WAL**：单写入多读取模式，支持并发
3. **多Provider LLM**：支持Anthropic、OpenAI、Gemini
4. **混合搜索**：BM25关键词 + 语义向量，兼顾精确和模糊匹配
5. **增量编译**：基于manifest的变更检测，避免重复处理
6. **检查点恢复**：编译状态持久化，支持断点续传
