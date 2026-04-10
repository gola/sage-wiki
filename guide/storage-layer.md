# 存储层

本文档深入解析 `internal/storage/`、`internal/vectors/`、`internal/memory/` 模块，sage-wiki的持久化层。

## 快速索引

- `DB`: SQLite数据库连接管理
- `MemoryStore`: 文档条目存储
- `VectorStore`: 向量嵌入存储
- `OntologyStore`: 概念本体存储

## 模块结构

```
internal/
├── storage/
│   └── db.go           # SQLite连接管理
├── memory/
│   └── entries.go      # 文档条目CRUD
├── vectors/
│   └── store.go        # 向量索引和搜索
└── ontology/
    └── ontology.go     # 概念关系存储
```

## SQLite 数据库

**文件**: `internal/storage/db.go`

### DB 结构

```go
type DB struct {
    write     *sql.DB      // 写连接（单连接）
    read      *sql.DB      // 读连接池
    writeMu   sync.Mutex   // 写锁
    closeOnce sync.Once    // 关闭一次
}
```

### 连接管理

```go
func Open(path string) (*DB, error) {
    // 写连接 - 单连接
    writeDB, _ := sql.Open("sqlite", path)
    writeDB.SetMaxOpenConns(1)
    
    // 启用WAL模式
    writeDB.Exec("PRAGMA journal_mode=WAL")
    writeDB.Exec("PRAGMA busy_timeout=5000")
    writeDB.Exec("PRAGMA foreign_keys=ON")
    
    // 读连接池 - 4个连接
    readDB, _ := sql.Open("sqlite", path+"?mode=ro")
    readDB.SetMaxOpenConns(4)
    
    return &DB{write: writeDB, read: readDB}, nil
}
```

### 数据库模式

```sql
-- 文档条目表
CREATE TABLE entries (
    id          TEXT PRIMARY KEY,
    path        TEXT UNIQUE NOT NULL,
    title       TEXT,
    summary     TEXT,
    content     TEXT,
    hash        TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

-- 向量嵌入表
CREATE TABLE embeddings (
    id          TEXT PRIMARY KEY,
    entry_id    TEXT NOT NULL,
    embedding   BLOB NOT NULL,
    model       TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    FOREIGN KEY (entry_id) REFERENCES entries(id)
);

-- 概念表
CREATE TABLE concepts (
    id          TEXT PRIMARY KEY,
    name        TEXT UNIQUE NOT NULL,
    definition  TEXT,
    created_at  TEXT NOT NULL
);

-- 概念关系表
CREATE TABLE relations (
    id          TEXT PRIMARY KEY,
    source      TEXT NOT NULL,
    target      TEXT NOT NULL,
    relation    TEXT NOT NULL,
    weight      REAL DEFAULT 1.0,
    FOREIGN KEY (source) REFERENCES concepts(id),
    FOREIGN KEY (target) REFERENCES concepts(id)
);
```

## Memory Store

**文件**: `internal/memory/entries.go`

### Entry 结构

```go
type Entry struct {
    ID        string    // UUID
    Path      string    // 源文件路径
    Title     string    // 文档标题
    Summary   string    // 摘要
    Content   string    // 完整内容
    Hash      string    // 内容hash
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### CRUD 操作

```go
type Store struct {
    db *storage.DB
}

// 创建条目
func (s *Store) Create(entry *Entry) error

// 获取条目
func (s *Store) Get(id string) (*Entry, error)
func (s *Store) GetByPath(path string) (*Entry, error)

// 更新条目
func (s *Store) Update(entry *Entry) error

// 删除条目
func (s *Store) Delete(id string) error

// 列出条目
func (s *Store) List(opts ListOpts) ([]Entry, error)

// 搜索条目（BM25）
func (s *Store) Search(query string, limit int) ([]Entry, error)
```

## Vector Store

**文件**: `internal/vectors/store.go`

### Embedding 结构

```go
type Embedding struct {
    ID        string    // UUID
    EntryID   string    // 关联条目ID
    Vector    []float32 // 嵌入向量
    Model     string    // 嵌入模型
    CreatedAt time.Time
}
```

### 向量操作

```go
type Store struct {
    db *storage.DB
}

// 存储嵌入
func (s *Store) Store(embedding *Embedding) error

// 获取嵌入
func (s *Store) Get(id string) (*Embedding, error)
func (s *Store) GetByEntry(entryID string) (*Embedding, error)

// 向量搜索（余弦相似度）
func (s *Store) Search(vector []float32, limit int) ([]SearchResult, error)

type SearchResult struct {
    EntryID   string
    Score     float64  // 相似度分数
}
```

### 向量搜索实现

```go
func (s *Store) Search(query []float32, limit int) ([]SearchResult, error) {
    // 从数据库加载所有向量
    rows, _ := s.db.ReadDB().Query("SELECT id, entry_id, embedding FROM embeddings")
    defer rows.Close()
    
    var results []SearchResult
    for rows.Next() {
        var id, entryID string
        var blob []byte
        rows.Scan(&id, &entryID, &blob)
        
        // 反序列化向量
        vec := deserializeVector(blob)
        
        // 计算余弦相似度
        score := cosineSimilarity(query, vec)
        results = append(results, SearchResult{
            EntryID: entryID,
            Score:   score,
        })
    }
    
    // 按分数排序
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    return results[:limit], nil
}
```

## Ontology Store

**文件**: `internal/ontology/ontology.go`

### Concept 结构

```go
type Concept struct {
    ID         string    // UUID
    Name       string    // 概念名称
    Definition string    // 概念定义
    CreatedAt  time.Time
}
```

### Relation 结构

```go
type Relation struct {
    ID       string  // UUID
    Source   string  // 源概念ID
    Target   string  // 目标概念ID
    Relation string  // 关系类型
    Weight   float64 // 关系权重
}
```

### 本体操作

```go
type Store struct {
    db              *storage.DB
    validRelations  []string // 有效关系类型
}

// 概念操作
func (s *Store) CreateConcept(concept *Concept) error
func (s *Store) GetConcept(id string) (*Concept, error)
func (s *Store) GetConceptByName(name string) (*Concept, error)

// 关系操作
func (s *Store) CreateRelation(relation *Relation) error
func (s *Store) GetRelations(conceptID string) ([]Relation, error)

// 图遍历
func (s *Store) GetRelated(conceptID string, depth int) ([]Concept, error)
```

### 预定义关系类型

```go
var DefaultRelations = []string{
    "relates_to",    // 相关
    "depends_on",    // 依赖
    "extends",       // 扩展
    "contrasts",     // 对比
    "example_of",    // 示例
}
```

## 混合搜索

**文件**: `internal/hybrid/search.go`

```go
type Searcher struct {
    mem *memory.Store
    vec *vectors.Store
}

type HybridResult struct {
    Entry    *memory.Entry
    BM25     float64  // BM25分数
    Semantic float64  // 语义分数
    Combined float64  // 综合分数
}

func (s *Searcher) Search(query string, limit int) ([]HybridResult, error) {
    // 1. BM25搜索
    bm25Results, _ := s.mem.Search(query, limit*2)
    
    // 2. 向量搜索
    queryVec, _ := s.embed(query)
    vecResults, _ := s.vec.Search(queryVec, limit*2)
    
    // 3. 合并结果
    return s.merge(bm25Results, vecResults, limit), nil
}
```

### 分数合并策略

```go
func (s *Searcher) merge(bm25 []memory.Entry, vec []vectors.SearchResult, limit int) []HybridResult {
    // 归一化BM25分数
    // 归一化语义分数
    // 加权合并: Combined = 0.3*BM25 + 0.7*Semantic
    // 按Combined排序
}
```

## 设计要点

1. **WAL模式**: SQLite预写日志，支持并发读写
2. **单写多读**: 写连接单连接，读连接池化
3. **纯Go实现**: 使用modernc.org/sqlite，无CGO依赖
4. **混合搜索**: BM25 + 语义向量，兼顾精确和模糊匹配
5. **图结构**: 概念关系存储支持图遍历
