# FridaForge

**声明式 Frida 脚本工程化平台** — 使用 YAML 声明 Hook 目标，自动生成 Frida JavaScript 脚本，管理多设备并发 Hook 会话。

## 状态

🚧 早期开发阶段（0.x）— API 可能发生 Breaking Changes。

## 快速开始

```bash
go install github.com/bigsmater/fridaforge/cmd/fridaforge@latest
fridaforge --help
```

## 项目结构

```
cmd/fridaforge/     # CLI 入口
pkg/                # 公共库（可被外部导入）
  config/           # 配置解析
  spec/             # Hook 声明模型
  fridaengine/      # Frida 调度引擎
  codegen/          # 代码生成器
  mcpserver/        # MCP Server
internal/           # 内部实现（不可被外部导入）
docs/               # 文档
  learn/            # 学习笔记
  reference/        # 参考文档
tests/              # 测试
  harness/          # 测试脚手架（靶机）
```

## 文档

- [项目宪法](.specify/memory/constitution.md)
- [SpecKit 各阶段设计哲学](docs/reference/speckit-rationale.md)

## 许可

MIT
