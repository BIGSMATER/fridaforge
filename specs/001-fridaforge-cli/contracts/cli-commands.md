# CLI 命令契约

**功能**: 001-fridaforge-cli
**日期**: 2026-05-09

## 命令树

```
fridaforge                           # 根命令
├── device list                      # 列出已连接的设备
├── spec validate <文件>             # 校验 HookSpec YAML 文件
└── help [命令]                      # 显示帮助（cobra 内置）
```

## 全局标志

| 标志 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `--help`, `-h` | bool | false | 显示任意命令的帮助信息 |
| `--version`, `-v` | bool | false | 打印版本信息 |

注意：M1 不引入 `--verbose`/`--debug`（根据澄清 Q3）。

---

## 命令: `fridaforge device list`

列出所有已连接的 Frida 可用设备。

**用法**: `fridaforge device list`

**标志**: 无 (M1)

**退出码**:

| 码 | 含义 |
|----|------|
| 0 | 成功：列出设备或显示"无设备"消息 |
| 1 | 错误：Frida 服务不可达 |

**Stdout（成功，有设备）**:
```
ID              NAME                    TYPE
emulator-5554   Android Emulator 5554   emulator
R5CT1234ABCD    Samsung Galaxy S21      usb
```

**Stdout（成功，无设备）**:
```
未发现已连接的设备。
```

**Stderr（Frida 不可达）**:
```
错误：Frida 服务不可达。请确保目标设备上 frida-server 正在运行。
```
退出码: 1

---

## 命令: `fridaforge spec validate <文件>`

校验 HookSpec YAML 文件的结构正确性。

**用法**: `fridaforge spec validate <文件>`

**参数**:

| 参数 | 必填 | 描述 |
|------|------|------|
| `<文件>` | 是 | YAML 规格文件的路径 |

**退出码**:

| 码 | 含义 |
|----|------|
| 0 | 校验通过 |
| 1 | 校验失败（文件不存在、YAML 语法错误、字段缺失、Hook 类型非法） |

**Stdout（成功）**:
```
✓ 配置有效: /path/to/hooks.yaml
  目标应用: com.example.app
  Hook 数量: 3
```

**Stdout（校验失败）**:
```
✗ 配置无效: /path/to/hooks.yaml
  hooks[0].class_name: 不能为空（第 3 行）
  hooks[1].hook_type: 不支持的值 "patch"（第 8 行）
  支持的 Hook 类型: overload, replace
```
退出码: 1

**Stderr（文件不存在）**:
```
错误：文件不存在: /path/to/missing.yaml
```
退出码: 1

**Stderr（YAML 语法错误）**:
```
错误：无法解析 YAML: yaml: line 5: did not find expected key
```
退出码: 1

**Stderr（IO 错误）**:
```
错误：无法读取文件: permission denied
```
退出码: 1

---

## 伦理声明流程

首次调用任意命令时，根命令的 `PersistentPreRun` 执行：

1. 检查标记文件 `~/.fridaforge/agreed`
2. 如果不存在：打印声明并提示用户输入 `AGREE`
3. 如果用户输入 `AGREE`：创建标记文件，继续执行
4. 如果用户拒绝：退出码 1

**声明文本**:
```
╔══════════════════════════════════════════════════════════╗
║  FridaForge — 伦理使用声明                               ║
║                                                          ║
║  本工具仅供授权安全测试和教育目的使用。                      ║
║                                                          ║
║  使用 FridaForge 前，你必须获得目标应用和设备               ║
║  所有者的明确许可。                                        ║
║                                                          ║
║  未经授权的使用可能违反适用法律。                           ║
║                                                          ║
║  输入 'AGREE' 表示接受并继续：                             ║
╚══════════════════════════════════════════════════════════╝
```
