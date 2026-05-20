# md2html（云枢集成）

基于开源项目 [haidang1810/md2html](https://github.com/haidang1810/md2html)（MIT）的 HTML 模板与组件规范，在本仓库中通过 Go 工具将 `docs/**/*.md` 转为**单文件、可离线分享**的 HTML 页面。

## 特性

- 侧边栏目录（滚动高亮）
- Mermaid 流程图 / 时序图（` ```mermaid ` 代码块）
- 引用块 → 彩色提示框（注意 / 禁止 / 决定等）
- 含「步骤 / 流程 / 链路」标题下的有序列表 → 时间线卡片
- 宽表格横向滚动
- 明暗主题切换、打印样式

## 一键生成核心文档

在仓库根目录执行：

```bash
go run ./cmd/md2html --bundle
```

输出目录：`docs/html/`，打开 `docs/html/index.html` 即可浏览全部入口。

## 单文件转换

```bash
go run ./cmd/md2html docs/alert-notify-guide.md
go run ./cmd/md2html docs/alert-notify-guide.md --out docs/html/custom.html
```

## 文件说明

| 文件 | 来源 |
|------|------|
| `template.html` | [md2html/template.html](https://github.com/haidang1810/md2html/blob/main/template.html) |
| `components.md` | 组件目录（供 AI / 手工精修参考） |
| `SKILL.md` | 上游 Agent Skill 说明 |

实现代码：`internal/md2html/`、`cmd/md2html/`。

## 与上游 Skill 的区别

上游 `/md2html` 由 AI 按 `components.md` **语义选组件**（质量更高）。本工具为 **确定性批量转换**，适合维护文档集；重要对外文档可在生成后用 Cursor 执行 `/md2html` 精修单篇。

## 更新模板

```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/haidang1810/md2html/main/template.html" -OutFile tools/md2html/template.html
```
