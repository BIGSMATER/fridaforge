# 快速入门: FridaForge CLI

**功能**: 001-fridaforge-cli
**日期**: 2026-05-09

## 前置条件

- 已安装 Go 1.25
- 已安装 Git

## 构建

```bash
git clone <仓库地址>
cd fridaforge
go build -o fridaforge ./cmd/fridaforge/
```

或安装到 `$GOPATH/bin`:

```bash
go install ./cmd/fridaforge/
```

## 验证

```bash
./fridaforge --help
./fridaforge --version
```

## 基本用法

### 校验 Hook 规格文件

创建 `test.yaml`:

```yaml
app_package: com.example.app
hooks:
  - class_name: com.example.MainActivity
    method_name: onCreate
    hook_type: overload
```

校验:

```bash
./fridaforge spec validate test.yaml
# ✓ 配置有效: test.yaml
#   目标应用: com.example.app
#   Hook 数量: 1
```

### 列出设备

```bash
./fridaforge device list
```

## 运行测试

```bash
go test ./...
go test -cover ./...
go test -v ./pkg/config/
```

## 项目布局

```
cmd/fridaforge/   — CLI 入口点（cobra 命令）
pkg/config/       — YAML 加载与校验
pkg/spec/         — 数据类型（HookSpec、HookTarget 等）
pkg/device/       — 设备管理器接口
```

## 故障排查

| 问题 | 解决方案 |
|------|----------|
| `fridaforge: 未找到命令` | 确保 `$GOPATH/bin` 在 `$PATH` 中，或使用 `./fridaforge` |
| `go: 未找到模块` | 先运行 `go mod tidy` |
| 构建失败 | 确保 Go 1.25 已安装：`go version` |
