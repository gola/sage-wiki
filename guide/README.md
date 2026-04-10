# sage-wiki 代码库指南

本指南帮助开发者快速理解 sage-wiki 项目的架构和核心模块。

## 文档索引

| 文档 | 内容 |
|------|------|
| [map.md](map.md) | 项目结构、执行链路和推荐阅读顺序 |
| [cli-commands.md](cli-commands.md) | CLI命令入口和参数处理 |
| [compiler-pipeline.md](compiler-pipeline.md) | 编译器流水线：文档处理的核心逻辑 |
| [llm-client.md](llm-client.md) | LLM客户端抽象和多Provider支持 |
| [storage-layer.md](storage-layer.md) | 存储层：SQLite数据库和向量存储 |
| [mcp-server.md](mcp-server.md) | MCP服务器：对外暴露的工具接口 |

## 项目概述

sage-wiki 是一个 LLM 编译的个人知识库系统。它将原始文档（PDF、Markdown、Word等）编译成结构化、相互链接的 wiki 文章，支持概念提取、交叉引用发现和混合搜索。

**核心特性**：
- 多格式文档提取（PDF、Word、Excel、EPUB、图片等）
- LLM驱动的摘要和概念提取
- 混合搜索（BM25 + 语义搜索）
- MCP协议支持，可与任何LLM代理集成
- 单一二进制文件，无需额外依赖

## 快速开始

```bash
# 初始化项目
mkdir my-wiki && cd my-wiki
sage-wiki init

# 添加源文件到 raw/ 目录
cp ~/papers/*.pdf raw/papers/

# 编译
sage-wiki compile

# 搜索
sage-wiki search "attention mechanism"

# 问答
sage-wiki query "How does flash attention work?"
```

## 推荐阅读顺序

1. **新手**：先读 [map.md](map.md) 了解整体架构
2. **理解编译流程**：读 [compiler-pipeline.md](compiler-pipeline.md)
3. **理解LLM集成**：读 [llm-client.md](llm-client.md)
4. **理解存储设计**：读 [storage-layer.md](storage-layer.md)
5. **理解MCP接口**：读 [mcp-server.md](mcp-server.md)
