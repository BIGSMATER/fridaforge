# Feature Specification: FridaForge CLI 命令行工具

**Feature Branch**: `001-fridaforge-cli`  
**Created**: 2026-05-09  
**Status**: Draft  
**Input**: User description: "构建 FridaForge 的 CLI 命令行工具。支持用户通过 YAML 配置文件声明 Frida Hook 目标（要 Hook 的应用包名、类名、方法名、Hook 类型），并解析为内存中的结构体。CLI 至少支持两个子命令：`fridaforge device list` 列出连接的设备，`fridaforge spec validate <文件>` 校验配置文件的合法性。"

## Clarifications

### Session 2026-05-09

- Q: YAML 文件顶层结构：是单个文件对应一个应用（共享 app_package），还是每个 Hook 独立声明 app_package？ → A: 单个文件对应一个应用，顶层声明共享的 app_package，下面列出该应用的所有 Hook 目标。
- Q: `device list` 时 Frida 服务不可达时如何处理？ → A: 输出明确错误信息（"Frida 服务不可达"），退出码非 0。
- Q: CLI 是否需要 verbose/debug 详细输出模式？ → A: M1 不引入 verbose 模式，仅依赖错误信息的清晰度。
- Q: 重载方法（同名不同参数）如何处理？ → A: M1 接受重载歧义，作为已知限制记录。Hook 时由 Frida 运行时匹配第一个找到的方法，M3 引入参数签名后解决。
- Q: `spec validate` 是否同时输出解析后的结构化数据？ → A: `validate` 只做校验，不输出解析结果。解析功能作为库内部接口供其他模块（codegen、engine）调用，不暴露为独立 CLI 命令。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 校验 Hook 配置文件 (Priority: P1)

逆向工程师编写了一个 YAML 配置文件，声明要对某个 Android 应用的特定方法进行 Hook。在执行 Hook 之前，用户需要确认配置文件的格式正确、所有必填字段完整、Hook 类型合法。用户运行 `fridaforge spec validate hooks.yaml`，系统解析文件并报告验证结果——通过则显示成功信息，失败则指出具体的错误位置和原因。

**Why this priority**: 配置验证是所有后续操作（代码生成、Hook 执行）的基础。没有可靠的验证，用户会在执行阶段才发现配置错误，浪费时间且难以排查。

**Independent Test**: 提供一个合法的 YAML 文件和一个非法的 YAML 文件，分别运行 `fridaforge spec validate`，验证系统给出正确的通过/失败反馈及错误定位信息。这个功能可以独立演示，不依赖设备连接或代码生成。

**Acceptance Scenarios**:

1. **Given** 一个包含完整字段（应用包名、类名、方法名、Hook 类型）的合法 YAML 文件，**When** 用户运行 `fridaforge spec validate valid.yaml`，**Then** 系统输出验证通过信息，退出码为 0。
2. **Given** 一个 YAML 文件缺少必填字段（如缺少方法名），**When** 用户运行 `fridaforge spec validate invalid.yaml`，**Then** 系统输出清晰的错误信息，指出缺失的字段及在文件中的位置，退出码为非 0。
3. **Given** 一个 YAML 文件包含不支持的 Hook 类型（如 `"unknown_type"`），**When** 用户运行 `fridaforge spec validate badtype.yaml`，**Then** 系统输出错误信息，列出支持的 Hook 类型，并指出违规字段的位置。
4. **Given** 一个语法错误的非合法 YAML 文件，**When** 用户运行 `fridaforge spec validate broken.yaml`，**Then** 系统输出 YAML 解析错误信息，包含行号等定位信息。
5. **Given** 指定的文件路径不存在，**When** 用户运行 `fridaforge spec validate nonexistent.yaml`，**Then** 系统输出文件不存在的错误信息。

---

### User Story 2 - 列出连接的设备 (Priority: P2)

逆向工程师在进行 Hook 之前，需要确认有哪些设备（物理手机、模拟器）当前已通过 USB 或网络连接到开发环境。用户运行 `fridaforge device list`，系统列出所有已连接设备的标识符、设备名称和设备类型。

**Why this priority**: 设备发现是实际执行 Hook 的前置步骤。用户需要先知道有哪些可用设备，才能指定目标设备。它可以独立于配置验证工作。

**Independent Test**: 在有设备连接和无设备连接两种环境下分别运行 `fridaforge device list`，验证系统正确列出设备信息或显示"无可用设备"提示。

**Acceptance Scenarios**:

1. **Given** 至少一台已通过 USB 连接的 Android 设备，**When** 用户运行 `fridaforge device list`，**Then** 系统列出该设备，显示设备 ID、设备名称和类型（如 USB）。
2. **Given** 没有任何设备连接，**When** 用户运行 `fridaforge device list`，**Then** 系统输出"无可用设备"的提示信息，退出码为 0。
3. **Given** 多台设备同时连接（USB + 模拟器），**When** 用户运行 `fridaforge device list`，**Then** 系统列出所有设备，每台设备的信息清晰可区分。
4. **Given** Frida 服务未运行或不可达，**When** 用户运行 `fridaforge device list`，**Then** 系统输出"Frida 服务不可达"错误信息，退出码非 0。

---

### User Story 3 - YAML 配置解析为结构化数据 (Priority: P3)

用户的 Hook 配置文件通过验证后，需要被解析为内存中的结构化数据，供后续的代码生成器和调度引擎使用。解析后的结构化数据包含应用包名、类名、方法名和 Hook 类型等字段，可以直接被程序读取操作。

**Why this priority**: 这是验证和执行的桥梁。验证确认配置正确，解析将配置数据转化为程序可用的形式。P3 因为它依赖 P1（验证通过后才能解析），但它是后续 M2/M3 的基础。

**Independent Test**: 读取一个合法的 YAML 文件，检查解析后的结构体数据是否与文件内容完全一致（包括所有字段值）；读取时也应触发基础验证。

**Acceptance Scenarios**:

1. **Given** 一个包含单个 Hook 目标的合法 YAML 文件，**When** 系统解析该文件，**Then** 返回的结构化数据中应用包名、类名、方法名、Hook 类型与文件内容一致。
2. **Given** 一个包含多个 Hook 目标的合法 YAML 文件，**When** 系统解析该文件，**Then** 返回的结构化数据列表包含所有 Hook 目标，顺序与文件声明一致。
3. **Given** 一个合法的 YAML 文件，**When** 其他模块（如代码生成器）访问解析后的结构化数据，**Then** 所有字段均可读取且类型正确。

---

### Edge Cases

- 用户提供空的 YAML 文件（没有任何 Hook 定义）时，系统应报告"空配置"而非崩溃。
- Hook 目标中应用包名不符合 Android 包名格式（如包含非法字符）时，验证应发出警告或错误。
- 同一个文件中重复声明了相同的方法 Hook 时，系统应警告用户存在重复配置。
- 配置文件包含未知的额外字段（用户拼写错误）时，系统应提示未知字段而非静默忽略。
- YAML 文件中包含数百个 Hook 目标时，验证和解析应在 2 秒内完成（与 SC-001 单文件校验时间目标一致）。
- 文件编码不是 UTF-8 时，系统应能正确处理或报错。
- 配置文件路径包含特殊字符或空格时，系统应正确处理。
- Frida 服务/守护进程未启动时运行 `device list`，系统应输出清晰的"服务不可达"错误信息并返回非 0 退出码。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须提供 `fridaforge device list` 子命令，列出当前所有已连接的设备。若 Frida 服务不可达，系统必须输出明确错误信息并以非 0 退出码退出。
- **FR-002**: 系统必须提供 `fridaforge spec validate <文件路径>` 子命令，仅校验配置文件的合法性并报告结果，不输出解析后的结构化数据。
- **FR-003**: 每个 Hook 目标声明必须包含以下必填字段：应用包名（app_package）、类名（class_name）、方法名（method_name）、Hook 类型（hook_type）。
- **FR-003a**: 一个配置文件对应一个目标应用，应用包名（app_package）在文件顶层声明为共享值，各 Hook 目标继承该包名。
- **FR-004**: Hook 类型必须至少支持 `overload`（重载/参数拦截）和 `replace`（替换方法实现）两种类型。
- **FR-005**: 系统必须将合法的 YAML 配置文件解析为内存中的结构化数据，作为库内部接口供下游模块（代码生成器、调度引擎）直接使用，不暴露为独立 CLI 命令。
- **FR-006**: 校验失败时，系统必须输出清晰的错误信息，包括：错误原因、出错字段、在文件中的位置（行号）。
- **FR-007**: 所有子命令必须包含帮助信息，用户可通过 `--help` 或 `-h` 标志查看用法说明。
- **FR-008**: 系统必须使用标准的退出码：成功为 0，任何错误为非 0。
- **FR-009**: 系统必须处理文件 I/O 错误（文件不存在、无读取权限等），并提供用户友好的错误信息。
- **FR-010**: 设备列表中的每台设备必须至少包含设备标识符（ID）、设备名称和连接类型信息。

### Key Entities

- **HookSpec（Hook 规格文件）**: 一个 YAML 配置文件的整体表示，一个文件对应一个目标应用。顶层声明共享的应用包名（app_package），其下列举该应用的一个或多个 Hook 目标（HookTarget）。配置文件不跨应用混合声明。
- **HookTarget（Hook 目标）**: 单条 Hook 声明，包含类全限定名、方法名、Hook 类型。M1 仅记录方法名（不含参数签名）。是 Hook 操作的最小可执行单元。
- **Device（设备）**: 通过 Frida 连接的目标设备，包含唯一标识符、设备名称和连接类型（USB/网络/模拟器）。
- **ValidationResult（校验结果）**: 校验操作的结果，包含是否通过、错误列表（如有），每条错误包含字段路径和描述信息。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 用户验证一个合法的配置文件能在 2 秒内完成并得到确认结果。
- **SC-002**: 对于包含字段缺失、类型错误等常见问题的配置文件，用户能在错误输出中直接定位问题（精确到字段名和行号），无需查阅额外文档。
- **SC-003**: 用户列出已连接设备能在 5 秒内返回结果。
- **SC-004**: 100% 的必填字段缺失和 Hook 类型非法情况能被校验捕获并报告。
- **SC-005**: 新用户仅通过 `fridaforge --help` 和 `fridaforge <子命令> --help` 的输出就能理解所有可用命令的用途和参数格式。
- **SC-006**: 解析后的结构化数据与原始 YAML 文件内容语义一致率达到 100%（所有字段值无遗漏、无错误类型转换）。

## Assumptions

- 目标用户已在系统中安装了 Frida 相关工具（frida-server 等），设备和 Frida 环境的连接由用户在外部管理。
- 配置文件使用 YAML 格式（与用户输入描述一致），后续版本可扩展支持 JSON 等格式。
- Hook 类型的语义定义遵循 Frida 社区标准：`overload` 表示参数拦截/重载（保留原方法调用），`replace` 表示完全替换方法实现。
- 应用包名遵循 Android 包名规范（如 `com.example.app`），但初级验证仅检查非空，格式严格度可在后续迭代中加强。
- CLI 工具作为单一可执行文件发布，支持 Linux、macOS 和 Windows 平台。
- YAML 文件的编码格式为 UTF-8。
- 当前阶段不考虑 HookTarget 中包含的参数签名详细定义（如 `(int, String)`），仅记录方法名。参数签名将在 M3 阶段补充。因此存在重载方法歧义：若目标类中存在多个同名方法，系统无法区分，由 Frida 运行时匹配第一个找到的方法。这作为 M1 的已知限制。
- M1 不引入 verbose/debug 详细输出模式，诊断能力通过错误信息清晰度保证。
