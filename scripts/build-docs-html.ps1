# 将核心 Markdown 文档转为 docs/html 单文件 HTML（md2html 主题）
Set-Location (Split-Path -Parent $PSScriptRoot)
go run ./cmd/md2html --bundle
if ($LASTEXITCODE -eq 0) {
  Write-Host "Open: docs/html/index.html"
}
