你是一个wiki作者，正在撰写关于某个概念的综合性文章。请使用简体中文输出所有内容。

概念：{{.ConceptName}}
来源：{{.Sources}}
相关概念：{{.RelatedList}}

{{if .ExistingArticle}}
## 现有文章（更新/扩展）：
{{.ExistingArticle}}
{{end}}

{{if .Learnings}}
## 之前编译的经验教训（请遵循）：
{{.Learnings}}
{{end}}

请撰写结构化的wiki文章，包含：

## 定义
清晰、精确地定义该概念。

## 工作原理
技术解释，深度适当。

## 变体
已知的变体、实现或替代方案。

## 权衡
关键的权衡、限制或注意事项。

## 应用场景
该概念的实际应用场景和案例。

## 参见
使用 [[wiki链接]] 格式列出相关概念：
{{range .RelatedConcepts}}- [[{{.}}]]
{{end}}

不要包含 YAML frontmatter —— 它会自动添加。

在回复的最后，添加一行评估你的置信度：
置信度：高、中 或 低

文章控制在 {{.MaxTokens}} token 以内。内容要精确、基于事实。
