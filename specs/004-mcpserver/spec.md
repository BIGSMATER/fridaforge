# Feature Specification: MCP Server 集成

**Feature Branch**: `004-mcpserver`
**Created**: 2026-06-01
**Status**: Draft
**Input**: User description: "将 FridaForge 的 Hook 脚本生成和设备管理能力通过 MCP 协议暴露给 AI 编码工具，让大模型自动生成 Hook 脚本"

## Clarifications

### Session 2026-06-01

- Q: MCP 传输协议选择（stdio vs HTTP+SSE）？ → A: stdio — 标准输入/输出管道通信，AI 工具作为父进程启动 FridaForge MCP 子进程。
- Q: 模拟数据（设备/进程）存放位置？ → A: YAML/JSON 配置文件，启动时加载，便于测试和后续真机对接。

- Q: MCP 库选型？ → A: 官方 `modelcontextprotocol/go-sdk`，类型安全 handler 模式，内置 stdio transport，官方维护。
- Q: 校验发现多个字段错误时的报告粒度？ → A: 一次性返回所有字段错误（comprehensive），AI 一次修正全部问题，减少往返次数。
- Q: 操作日志输出目标？ → A: stderr（标准错误输出），stdout 保留给 MCP JSON-RPC 协议消息，两者隔离。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - AI 助手自动生成 Hook 脚本 (Priority: P1)

逆向工程师在 AI 编码助手中用自然语言描述 Hook 需求（如"帮我监控 `com.example.Test.hello` 方法的参数和返回值"），AI 助手将需求转换为结构化参数发送给 FridaForge，获得完整可执行的 Frida 脚本并展示。用户无需了解 Frida JavaScript API 细节即可完成 Hook 脚本编写。

**Why this priority**: M4 的核心价值交付——让 FridaForge 从 CLI 工具升级为 AI 可编程平台。M4 里程碑目标即"让大模型自动生成 Hook 脚本"，P1 直接达成此目标。

**Independent Test**: 启动 FridaForge MCP 服务，发送一个 overload 类型的 Hook 生成请求，验证返回的脚本语法正确且可直接注入目标进程。

**Acceptance Scenarios**:

1. **Given** FridaForge MCP 服务已启动，**When** AI 助手请求生成一个 overload 类型 Hook（目标类 `com.example.Test`，方法 `hello`），**Then** 返回包含目标方法拦截逻辑的完整 Hook 脚本。
2. **Given** FridaForge MCP 服务已启动，**When** AI 助手传入缺少必填字段的 Hook 参数，**Then** 返回明确的字段错误描述，帮助 AI 助手修正后重试。
3. **Given** AI 助手请求同时生成多种 Hook 类型（overload + override + native），**When** 执行生成，**Then** 返回的组合脚本包含所有 Hook，同类型 Hook 共享必要的运行时包装。

---

### User Story 2 - AI 助手校验 Hook 参数合法性 (Priority: P1)

逆向工程师在正式生成脚本前，通过 AI 助手先行校验 Hook 参数是否合法。AI 助手将参数发送给 FridaForge 校验，如有问题则提示用户修正，形成"先校验、后生成"的安全工作流。

**Why this priority**: 校验是生成的前置步骤。P1 因它与 US1 构成最小闭环——先校验再生成是 AI 交互中的自然流程。

**Independent Test**: 分别发送合法和非法的 Hook 参数进行校验，验证返回结果能正确区分通过/不通过，且不通过时给出具体字段和原因。

**Acceptance Scenarios**:

1. **Given** 传入完整的合法 Hook 配置，**When** 调用校验，**Then** 返回通过结果，错误列表为空。
2. **Given** 传入空类名的 Hook，**When** 调用校验，**Then** 返回不通过结果，错误信息指向具体字段（类名为空）。
3. **Given** 两个 Hook 声明完全重复，**When** 调用校验，**Then** 返回通过但附带警告提示存在重复配置。

---

### User Story 3 - AI 助手枚举可用调试设备 (Priority: P2)

逆向工程师需要了解当前连接了哪些调试设备。通过 AI 助手查询 FridaForge 获取设备列表，帮助用户选择合适的设备进行后续操作。

**Why this priority**: 设备枚举是 Attach 操作的前提步骤。P2 因为初期可用模拟数据独立验证，同时为后续里程碑的真机集成奠定框架。

**Independent Test**: 查询设备列表，验证返回列表包含设备标识和连接方式信息。

**Acceptance Scenarios**:

1. **Given** FridaForge 运行在模拟模式，**When** 查询设备列表，**Then** 返回包含设备标识、名称和连接类型的列表。
2. **Given** 模拟模式配置为空设备，**When** 查询设备列表，**Then** 返回空列表，不报错。

---

### User Story 4 - AI 助手枚举设备进程 (Priority: P2)

逆向工程师需要查看特定设备上运行了哪些应用。通过 AI 助手查询设备进程列表，定位目标应用。

**Why this priority**: 与 US3 并列，设备枚举后自然需要进程枚举。P2 因同样可模拟独立验证。

**Independent Test**: 查询指定设备的进程列表，验证返回列表包含进程 ID 和名称。

**Acceptance Scenarios**:

1. **Given** 模拟模式且指定有效设备 ID，**When** 查询进程列表，**Then** 返回包含进程 ID 和名称的列表。
2. **Given** 指定不存在的设备 ID，**When** 查询进程列表，**Then** 返回错误提示设备不存在。

---

### Edge Cases

- 客户端在未建立连接前发送请求？→ 服务拒绝请求，返回错误要求先完成连接握手。
- Hook 参数包含超长字符串（如恶意构造的极大类名）？→ 服务设置参数大小上限，超限返回错误。
- 客户端同时发送大量并发请求？→ 服务按请求到达顺序逐个响应，正确处理并发。
- 客户端意外断开连接？→ 服务检测到断开后优雅退出，释放已占用资源。
- 生成 native 类型 Hook 时缺少 `module_name`？→ 返回明确的参数校验错误，提示缺少模块名。
- MCP 协议版本兼容性？→ 由底层 MCP 通信库处理版本协商，FridaForge 不感知特定协议版本号。

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 实现标准 AI 工具集成协议 (MCP)，使 AI 编码助手能发现和调用 FridaForge 功能。
- **FR-002**: 系统 MUST 暴露"生成 Hook 脚本"能力：接受 Hook 目标描述（应用包名、类名、方法名、Hook 类型等），返回完整可执行的 Frida 脚本。
- **FR-003**: 系统 MUST 暴露"校验 Hook 参数"能力：接受 Hook 配置，返回通过/不通过状态及具体字段错误描述。
- **FR-004**: 系统 MUST 暴露"枚举设备"能力：返回已连接调试设备的列表信息（设备 ID、名称、连接类型）。初期通过配置文件加载模拟数据。
- **FR-005**: 系统 MUST 暴露"枚举进程"能力：接受设备标识，返回该设备上运行进程的列表信息（进程 ID、进程名）。初期通过配置文件加载模拟数据。
- **FR-006**: 每种能力 MUST 附带结构化输入参数定义和功能描述，使 AI 助手无需示例即可理解如何使用。
- **FR-007**: 系统 MUST 记录每次能力调用的信息（时间、能力名、参数摘要）到标准错误输出（stderr），用于审计和问题排查。
- **FR-008**: 系统 MUST 在客户端断开连接时完成优雅退出，不残留未释放的资源。
- **FR-009**: 能力调用返回的错误信息 MUST 足够明确，使 AI 助手能据此自主修正参数后重试。
- **FR-010**: 系统 MUST NOT 通过 MCP 暴露任意代码执行入口（如直接执行用户提供的脚本内容）。
- **FR-011**: "生成 Hook 脚本"和"校验 Hook 参数"能力 MUST 复用已有的代码生成和配置校验逻辑，保持行为一致性。
- **FR-012**: FridaForge 命令行 MUST 提供启动 MCP 服务的入口，通过标准输入/输出管道（stdio）与 AI 工具通信，使 AI 工具能将其作为子进程启动。
- **FR-013**: MCP 服务 MUST 在启动时自动完成与客户端的协议握手，无需用户手动配置。

### Key Entities

- **MCP 服务**: FridaForge 对外暴露的能力入口。管理生命周期、能力注册、请求调度和操作日志。
- **能力 (Capability)**: 每项 FridaForge 功能的 MCP 封装——包含名称、功能描述、输入参数格式定义和执行逻辑。
- **连接会话**: 从 AI 助手连接到断开的一次交互周期。通过取消机制管理生命周期。
- **Hook 配置**: 描述 Hook 目标的数据结构——目标应用、类名、方法名、Hook 类型等字段。
- **生成结果**: 代码生成能力的输出——包含完整脚本内容和元数据（生成时间、Hook 数量等）。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 服务能在 1 秒内完成启动并与 AI 助手建立连接。
- **SC-002**: 单个 Hook 目标的脚本生成在 2 秒内返回结果，用户感知即时响应。
- **SC-003**: 参数校验在 500 毫秒内返回结果，适合 AI 快速迭代修正的交互模式。
- **SC-004**: AI 助手不依赖任何示例，仅凭能力描述即能正确构造 90% 以上的调用请求。
- **SC-005**: AI 助手配置 FridaForge 为集成服务后，能在一次对话中完成"生成一个 overload 类型 Hook 的完整脚本"的任务。
- **SC-006**: 关键功能路径（服务启停、4 项能力调用、参数校验、错误处理）可通过模拟环境完整验证，覆盖率 ≥ 80%。
- **SC-007**: 所有能力调用可在单开发者机器上重现和调试，不依赖远程服务或特定硬件。

## Assumptions

- MCP 是 AI 工具集成的事实标准协议，目标 AI 助手支持通过 stdio 管道启动本地子进程方式连接 MCP 服务。
- 设备枚举和进程枚举的初期实现使用模拟数据。真实设备对接推迟到后续里程碑，当 MCP 服务需与 Frida 运行时环境交互时。
- Hook 脚本生成的输入参数采用结构化格式（等效于现有 Hook 配置的数据结构），因协议原生支持结构化参数。
- 协议版本兼容性由底层协议实现处理，FridaForge 无需自行实现多版本适配。
- 本里程碑仅实现协议中的"工具/能力调用"原语，不涉及"资源""提示"等可选原语。
- MCP 服务与 AI 助手在同一台机器上本地运行，无需网络认证机制。
