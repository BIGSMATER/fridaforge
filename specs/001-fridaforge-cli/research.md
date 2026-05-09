# 研究笔记: FridaForge CLI 命令行工具

**功能**: 001-fridaforge-cli
**日期**: 2026-05-09

## 技术决策

### 决策 1: CLI 框架 — cobra

**决策**: 使用 `github.com/spf13/cobra` 做 CLI 命令路由。

**理由**:
- Cobra 是 Go 生态中事实标准的 CLI 框架（被 kubectl、hugo、GitHub CLI 使用）。
- 提供内置帮助文本生成（`--help`）、子命令嵌套和 PersistentPreRun 钩子。
- PersistentPreRun 钩子可以实现宪法要求的首次运行伦理声明检查。
- 对齐项目的学习目标——理解 Go CLI 模式。

**备选方案**:
- `flag`（标准库）：过于原始，不支持子命令层级；缺少帮助生成。
- `urfave/cli`：可行但 Go 工具链中使用较少；cobra 生态集成更丰富（viper、自动补全）。

### 决策 2: 配置管理 — viper + yaml.v3

**决策**: 使用 `github.com/spf13/viper` 做配置文件路径解析，`gopkg.in/yaml.v3` 做直接 YAML 反序列化。

**理由**:
- Viper 处理文件路径解析、环境变量绑定和配置合并（未来全局配置如 `~/.fridaforge.yaml` 需要这些能力）。
- yaml.v3 提供直接的 `yaml.Unmarshal` 配合结构体标签——比 viper 的 `Unmarshal` 更适合自定义校验。
- 组合方式：viper 负责"文件在哪"，yaml.v3 负责"文件里有什么"。

**备选方案**:
- yaml.v2：旧版本；v3 增加了往返保留和更好的错误信息。
- 纯 viper 反序列化：自定义校验逻辑不够灵活（需要检查原始 YAML 节点以获取精确行号）。

### 决策 3: 设备管理器架构 — 接口 + 桩

**决策**: 在 `pkg/device/` 中定义 `DeviceLister` 接口，M1 使用桩实现。M2 提供真实 Frida 实现。

**理由**:
- 允许 CLI 代码在无实际 Frida 依赖的情况下编写和测试。
- 接口契约早期定义；M2 只需替换实现即可。
- 桩可以返回预定义设备用于测试和演示。
- 遵循 Go 惯例：接受接口，返回结构体。

**接口设计**:
```go
type DeviceLister interface {
    ListDevices(ctx context.Context) ([]Device, error)
}
type Device struct {
    ID          string
    Name        string
    ConnectType string // "usb", "network", "emulator"
}
```

**备选方案**:
- M1 直接集成 Frida：违反"无实际 Frida 连接"约束。
- 无抽象（硬编码在 CLI 中）：M2 集成更困难且无法测试。

### 决策 4: 测试 — 标准库 Table-Driven

**决策**: 使用 Go 标准库 `testing` 配合 table-driven 测试模式。测试文件与源码同目录（`*_test.go`）。

**理由**:
- Table-driven 测试是 Go 社区标准，也是宪法强制要求（2.5）。
- 无外部测试框架依赖，降低 Go 学习项目的认知负担。
- 同目录测试提高可发现性，遵循 Go 惯例。

**备选方案**:
- testify：流行但增加依赖；宪法未强制要求；标准库足以满足 M1 校验器和解析器的测试需求。

### 决策 5: 错误模型 — 类型化校验错误

**决策**: 在 `pkg/spec/errors.go` 中定义结构化的校验错误类型，而非返回裸字符串。

**理由**:
- FR-006 要求错误包含字段路径和行号——结构化错误使其可测试。
- 类型化错误允许下游消费者编程区分错误类别。
- 实现 `error` 接口，提供描述性 `Error()` 字符串。

**错误类型**:
- `ValidationError`：多个字段级错误
- `FieldError`：单个字段错误，含路径、行号、消息

### 决策 6: 伦理声明 — 首次运行检查

**决策**: 使用标记文件（如 `~/.fridaforge/agreed`）追踪用户是否已接受伦理声明。根命令的 `PersistentPreRun` 检查此文件，不存在则提示用户。

**理由**:
- 宪法 3.4 要求所有 CLI 命令首次运行时显示伦理声明。
- 标记文件方案是标准做法（git、cocoapods、npm 对类似 EULA 流程均用此方式）。
- 避免每次调用都要求输入 `AGREE`。
- `.fridaforge/` 目录同时作为未来的配置主目录。

### 决策 7: YAML 结构 — 单文件单应用

**决策**: 每个规格文件针对一个目标应用。顶层 `app_package` 共享；Hook 目标列在 `hooks` 数组中。

**结构**:
```yaml
app_package: com.example.app
hooks:
  - class_name: com.example.MainActivity
    method_name: onCreate
    hook_type: overload
  - class_name: com.example.Utils
    method_name: encrypt
    hook_type: replace
```

**理由**: 参见 spec.md 的 Clarifications 章节 Q1。简化校验（每个文件一个应用上下文），自然映射到 Frida 的 attach 模型。

## 已解决的澄清项

所有技术未知项已由用户输入解决：
- Go 1.25、cobra、viper、yaml.v3 ✅
- 项目布局：`cmd/fridaforge/`、`pkg/{config,spec,device}/` ✅
- 测试：标准库 + table-driven、同目录 `*_test.go` ✅
- M1 范围：无网络、无数据库、无 Frida 连接 ✅

无遗留 NEEDS CLARIFICATION 标记。
