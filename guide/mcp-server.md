# MCP 服务器

本文档深入解析 `internal/mcp/` 模块，sage-wiki对外暴露的MCP协议接口。

## 快速索引

- `Server`: MCP服务器结构
- `NewServer()`: 创建服务器
- `Serve()`: 启动服务
- 工具注册: `registerReadTools()`, `registerWriteTools()`, `registerCompoundTools()`

## 模块结构

```
internal/mcp/
├── server.go          # 服务器主逻辑
├── tools_write.go     # 写入工具
├── tools_compound.go  # 复合工具
└── server_test.go     # 测试
```

## Server 结构

```go
type Server struct {
    mcp        *server.MCPServer  // MCP协议服务器
    projectDir string             // 项目目录
    db         *storage.DB        // 数据库连接
    mem        *memory.Store      // 文档存储
    vec        *vectors.Store     // 向量存储
    ont        *ontology.Store    // 本体存储
    searcher   *hybrid.Searcher   // 混合搜索器
    embedder   embed.Embedder     // 嵌入生成器
    cfg        *config.Config     // 配置
}
```

## 服务器创建

```go
func NewServer(projectDir string) (*Server, error) {
    // 加载配置
    cfg, _ := config.Load(filepath.Join(projectDir, "config.yaml"))
    
    // 打开数据库
    db, _ := storage.Open(filepath.Join(projectDir, ".sage", "wiki.db"))
    
    // 初始化存储
    mem := memory.NewStore(db)
    vec := vectors.NewStore(db)
    ont := ontology.NewStore(db, validRelations)
    searcher := hybrid.NewSearcher(mem, vec)
    
    // 创建MCP服务器
    mcpServer := server.NewMCPServer("sage-wiki", "0.1.0")
    
    s := &Server{
        projectDir: projectDir,
        db:         db,
        mem:        mem,
        vec:        vec,
        ont:        ont,
        searcher:   searcher,
        embedder:   embed.NewFromConfig(cfg),
        cfg:        cfg,
        mcp:        mcpServer,
    }
    
    // 注册工具
    s.registerReadTools()
    s.registerWriteTools()
    s.registerCompoundTools()
    
    return s, nil
}
```

## 工具注册

### 读取工具

**文件**: `internal/mcp/server.go`

```go
func (s *Server) registerReadTools() {
    // read_entry - 读取单个条目
    s.mcp.AddTool(mcp.Tool{
        Name:        "read_entry",
        Description: "Read a wiki entry by ID or path",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "id":   map[string]any{"type": "string"},
                "path": map[string]any{"type": "string"},
            },
        },
    }, s.handleReadEntry)
    
    // search - 搜索wiki
    s.mcp.AddTool(mcp.Tool{
        Name:        "search",
        Description: "Search the wiki using hybrid search",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "query": map[string]any{"type": "string"},
                "limit": map[string]any{"type": "integer", "default": 10},
            },
            Required: []string{"query"},
        },
    }, s.handleSearch)
    
    // query - 问答
    s.mcp.AddTool(mcp.Tool{
        Name:        "query",
        Description: "Ask a question against the wiki",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "question": map[string]any{"type": "string"},
            },
            Required: []string{"question"},
        },
    }, s.handleQuery)
}
```

### 写入工具

**文件**: `internal/mcp/tools_write.go`

```go
func (s *Server) registerWriteTools() {
    // write_entry - 写入条目
    s.mcp.AddTool(mcp.Tool{
        Name:        "write_entry",
        Description: "Write or update a wiki entry",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "path":    map[string]any{"type": "string"},
                "title":   map[string]any{"type": "string"},
                "content": map[string]any{"type": "string"},
                "summary": map[string]any{"type": "string"},
            },
            Required: []string{"path", "content"},
        },
    }, s.handleWriteEntry)
    
    // delete_entry - 删除条目
    s.mcp.AddTool(mcp.Tool{
        Name:        "delete_entry",
        Description: "Delete a wiki entry",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "id": map[string]any{"type": "string"},
            },
            Required: []string{"id"},
        },
    }, s.handleDeleteEntry)
}
```

### 复合工具

**文件**: `internal/mcp/tools_compound.go`

```go
func (s *Server) registerCompoundTools() {
    // add_concept - 添加概念
    s.mcp.AddTool(mcp.Tool{
        Name:        "add_concept",
        Description: "Add a concept with optional relations",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "name":       map[string]any{"type": "string"},
                "definition": map[string]any{"type": "string"},
                "relations": map[string]any{
                    "type": "array",
                    "items": map[string]any{
                        "type": "object",
                        "properties": map[string]any{
                            "target":   map[string]any{"type": "string"},
                            "relation": map[string]any{"type": "string"},
                        },
                    },
                },
            },
            Required: []string{"name"},
        },
    }, s.handleAddConcept)
    
    // link_entries - 链接条目
    s.mcp.AddTool(mcp.Tool{
        Name:        "link_entries",
        Description: "Create bidirectional links between entries",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]any{
                "source_id": map[string]any{"type": "string"},
                "target_id": map[string]any{"type": "string"},
                "relation":  map[string]any{"type": "string"},
            },
            Required: []string{"source_id", "target_id"},
        },
    }, s.handleLinkEntries)
}
```

## 工具处理器

### handleReadEntry

```go
func (s *Server) handleReadEntry(args map[string]any) (*mcp.CallToolResult, error) {
    id, _ := args["id"].(string)
    path, _ := args["path"].(string)
    
    var entry *memory.Entry
    var err error
    
    if id != "" {
        entry, err = s.mem.Get(id)
    } else if path != "" {
        entry, err = s.mem.GetByPath(path)
    } else {
        return mcp.NewToolResultError("must provide id or path"), nil
    }
    
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    // 返回JSON格式
    data, _ := json.MarshalIndent(entry, "", "  ")
    return mcp.NewToolResultText(string(data)), nil
}
```

### handleSearch

```go
func (s *Server) handleSearch(args map[string]any) (*mcp.CallToolResult, error) {
    query, _ := args["query"].(string)
    limit, _ := args["limit"].(float64)
    if limit == 0 {
        limit = 10
    }
    
    results, err := s.searcher.Search(query, int(limit))
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    // 格式化结果
    var output []string
    for _, r := range results {
        output = append(output, fmt.Sprintf("## %s\n\n%s\n\nScore: %.2f",
            r.Entry.Title,
            r.Entry.Summary,
            r.Combined,
        ))
    }
    
    return mcp.NewToolResultText(strings.Join(output, "\n---\n")), nil
}
```

### handleQuery

```go
func (s *Server) handleQuery(args map[string]any) (*mcp.CallToolResult, error) {
    question, _ := args["question"].(string)
    
    // 1. 搜索相关文档
    results, _ := s.searcher.Search(question, 5)
    
    // 2. 构建上下文
    var context strings.Builder
    for _, r := range results {
        context.WriteString(fmt.Sprintf("## %s\n\n%s\n\n",
            r.Entry.Title,
            r.Entry.Content,
        ))
    }
    
    // 3. 调用LLM生成答案
    answer, _ := s.llm.ChatCompletion([]llm.Message{
        {Role: "system", Content: "You are a helpful assistant. Answer based on the provided context."},
        {Role: "user", Content: fmt.Sprintf("Context:\n%s\n\nQuestion: %s", context.String(), question)},
    }, llm.CallOpts{Model: s.cfg.LLM.Model})
    
    return mcp.NewToolResultText(answer.Content), nil
}
```

## 服务启动

```go
func (s *Server) Serve() error {
    // 启动MCP服务器
    return s.mcp.Serve()
}
```

## MCP协议

MCP (Model Context Protocol) 是一个标准化的LLM代理工具协议：

```
┌─────────────┐         ┌─────────────┐
│  MCP Client │◄───────►│ MCP Server  │
│  (LLM代理)   │         │ (sage-wiki) │
└─────────────┘         └─────────────┘
      │                        │
      │  list_tools            │
      │───────────────────────►│
      │                        │
      │  tools_list             │
      │◄───────────────────────│
      │                        │
      │  call_tool(search, {}) │
      │───────────────────────►│
      │                        │
      │  tool_result           │
      │◄───────────────────────│
```

## 设计要点

1. **标准化协议**: 使用MCP协议，可与任何兼容的LLM代理集成
2. **工具分类**: 读取工具、写入工具、复合工具分离
3. **错误处理**: 使用`mcp.NewToolResultError`返回错误
4. **JSON响应**: 结构化数据使用JSON格式返回
5. **RAG问答**: query工具实现检索增强生成
