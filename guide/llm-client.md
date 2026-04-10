# LLM 客户端

本文档深入解析 `internal/llm/` 模块，sage-wiki的LLM抽象层。

## 快速索引

- `Client`: 统一LLM客户端
- `Provider`: 提供商接口
- `Message`: 聊天消息结构
- `CallOpts`: 调用选项
- `Response`: 响应结构

## 模块结构

```
internal/llm/
├── client.go      # 统一客户端和核心逻辑
├── anthropic.go   # Claude实现
├── openai.go      # GPT实现
├── gemini.go      # Gemini实现
├── cache.go       # 提示缓存
├── batch.go       # 批量API
├── cost.go        # 成本追踪
└── stream.go      # 流式响应
```

## 核心类型

### Client - 统一客户端

```go
type Client struct {
    provider Provider      // 底层提供商实现
    limiter  *rateLimiter  // 速率限制器
    client   http.Client   // HTTP客户端
    tracker  *CostTracker  // 可选成本追踪
    pass     string        // 当前编译阶段
    cacheID  string        // 活动缓存ID
}
```

**创建客户端**:
```go
client, err := llm.NewClient("anthropic", apiKey, "", 10)
```

### Provider - 提供商接口

```go
type Provider interface {
    Name() string
    ChatCompletion(messages []Message, opts CallOpts) (*Response, error)
    StreamCompletion(messages []Message, opts CallOpts) (<-chan StreamEvent, error)
}
```

**支持的提供商**:
- `anthropic`: Claude (claude-3-opus, claude-3-sonnet)
- `openai`: GPT (gpt-4o, gpt-4-turbo)
- `gemini`: Gemini (gemini-1.5-pro, gemini-1.5-flash)

### Message - 消息结构

```go
type Message struct {
    Role        string // "system", "user", "assistant"
    Content     string // 消息内容
    ImageBase64 string // base64图片数据（视觉消息）
    ImageMime   string // 图片MIME类型
}
```

### CallOpts - 调用选项

```go
type CallOpts struct {
    Model       string    // 模型名称
    MaxTokens   int       // 最大输出token
    Temperature float64   // 温度参数
}
```

### Response - 响应结构

```go
type Response struct {
    Content   string  // 响应内容
    Model     string  // 使用的模型
    TokensUsed int    // 总token数
    Usage     Usage   // 详细使用统计
}

type Usage struct {
    InputTokens  int // 输入token
    OutputTokens int // 输出token
    CachedTokens int // 缓存token
}
```

## 提供商实现

### Anthropic (Claude)

**文件**: `internal/llm/anthropic.go`

```go
type anthropicProvider struct {
    apiKey  string
    baseURL string
}

func (p *anthropicProvider) ChatCompletion(messages []Message, opts CallOpts) (*Response, error) {
    // 构建Anthropic API请求
    // 处理响应
}
```

**支持的模型**:
- `claude-3-opus-20240229`
- `claude-3-sonnet-20240229`
- `claude-3-haiku-20240307`

### OpenAI (GPT)

**文件**: `internal/llm/openai.go`

```go
type openaiProvider struct {
    apiKey  string
    baseURL string
}
```

**支持的模型**:
- `gpt-4o`
- `gpt-4-turbo`
- `gpt-3.5-turbo`

### Gemini

**文件**: `internal/llm/gemini.go`

```go
type geminiProvider struct {
    apiKey  string
    baseURL string
}
```

**支持的模型**:
- `gemini-1.5-pro`
- `gemini-1.5-flash`

## 提示缓存

**文件**: `internal/llm/cache.go`

```go
// 设置缓存
func (c *Client) SetupCache(cacheID string, systemPrompt string) error

// 使用缓存
func (c *Client) ChatCompletion(messages []Message, opts CallOpts) (*Response, error)
```

**缓存策略**:
- 系统提示可缓存
- 缓存命中时成本降低90%
- Anthropic和Gemini原生支持

## 批量API

**文件**: `internal/llm/batch.go`

```go
type BatchClient struct {
    provider string
    client   *Client
}

// 提交批量任务
func (b *BatchClient) Submit(messages []Message, opts CallOpts) (string, error)

// 检查状态
func (b *BatchClient) Status(batchID string) (*BatchStatus, error)

// 获取结果
func (b *BatchClient) Results(batchID string) ([]Response, error)
```

**优势**:
- 异步处理
- 50%成本折扣
- 适合大批量文档

## 成本追踪

**文件**: `internal/llm/cost.go`

```go
type CostTracker struct {
    entries []CostEntry
    mu      sync.Mutex
}

type CostEntry struct {
    Timestamp   time.Time
    Provider    string
    Model       string
    InputTokens int
    OutputTokens int
    CachedTokens int
    Cost        float64
}

type CostReport struct {
    TotalCost   float64
    ByProvider  map[string]float64
    ByModel     map[string]float64
    TotalTokens int
}
```

**使用方式**:
```go
tracker := llm.NewCostTracker()
client.SetTracker(tracker)

// 编译后获取报告
report := tracker.Report()
fmt.Printf("Total cost: $%.2f\n", report.TotalCost)
```

## 速率限制

```go
type rateLimiter struct {
    ticker    *time.Ticker
    tokens    chan struct{}
}

func (l *rateLimiter) Wait() {
    <-l.tokens
}
```

**配置**:
- Anthropic: 默认10请求/分钟
- OpenAI: 默认60请求/分钟
- Gemini: 默认15请求/分钟

## 重试机制

```go
func (c *Client) ChatCompletion(messages []Message, opts CallOpts) (*Response, error) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        resp, err := c.doRequest(messages, opts)
        if err == nil {
            return resp, nil
        }
        if isRateLimitError(err) {
            time.Sleep(exponentialBackoff(i))
            continue
        }
        return nil, err
    }
}
```

## 设计要点

1. **统一接口**: 所有提供商实现相同的`Provider`接口
2. **速率限制**: 内置速率限制避免API限流
3. **重试机制**: 自动重试可恢复错误
4. **成本追踪**: 可选的详细成本统计
5. **提示缓存**: 支持Anthropic/Gemini的提示缓存
6. **批量优化**: 支持批量API降低成本
