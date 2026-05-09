# YAML 结构契约

**功能**: 001-fridaforge-cli
**日期**: 2026-05-09

## HookSpec 文件结构

### 版本

结构版本 v1 (M1)。无需显式版本字段；所有文件均视为 v1。

### 结构

```yaml
# 必填：顶层应用包名
app_package: <string>

# 必填：Hook 目标列表（至少一个）
hooks:
  - class_name: <string>    # 必填：Dalvik 类全限定名
    method_name: <string>   # 必填：要 Hook 的方法名
    hook_type: <enum>       # 必填："overload" | "replace"
```

### 约束

| 字段 | 约束 |
|------|------|
| `app_package` | 非空字符串。点分隔的段。 |
| `hooks` | 必须为列表，≥ 1 个条目。 |
| `hooks[].class_name` | 非空字符串。点分隔的 Java 类全限定名。 |
| `hooks[].method_name` | 非空字符串。仅方法名（M1 无参数签名）。 |
| `hooks[].hook_type` | 必须严格为 `"overload"` 或 `"replace"`。 |

### 校验顺序

1. 文件存在且可读
2. UTF-8 编码检查
3. YAML 语法解析
4. 顶层结构：`app_package` 和 `hooks` 字段存在
5. `app_package` 非空
6. `hooks` 为列表且 ≥ 1 个条目
7. 每个 hook 条目：
   a. 必填字段存在：`class_name`、`method_name`、`hook_type`
   b. 必填字段非空
   c. `hook_type` 值为合法值（`overload` 或 `replace`）
8. 警告（不阻断通过）：
   a. 未知顶层键
   b. hook 条目中的未知键
   c. 重复的 `class_name` + `method_name` 组合

### 合法完整示例

```yaml
app_package: com.example.bank
hooks:
  - class_name: com.example.bank.MainActivity
    method_name: onCreate
    hook_type: overload

  - class_name: com.example.bank.crypto.AES
    method_name: encrypt
    hook_type: replace

  - class_name: com.example.bank.network.ApiClient
    method_name: sendRequest
    hook_type: overload
```

### 非法示例

**缺少必填字段**:
```yaml
app_package: com.example.app
hooks:
  - class_name: com.example.Foo
    # method_name 缺失 → 错误
    hook_type: overload
```

**非法 hook 类型**:
```yaml
app_package: com.example.app
hooks:
  - class_name: com.example.Foo
    method_name: bar
    hook_type: patch    # → 错误：必须是 "overload" 或 "replace"
```

**空 hooks 列表**:
```yaml
app_package: com.example.app
hooks: []    # → 错误：至少需要一个 hook
```

**缺失顶层字段**:
```yaml
# app_package 缺失 → 错误
hooks:
  - class_name: com.example.Foo
    method_name: bar
    hook_type: overload
```

### 文件命名约定

- 扩展名：`.yaml` 或 `.yml`
- 编码：UTF-8

### 可扩展性（未来）

- M3 将为 `hooks[]` 增加可选 `params` 字段用于参数签名。
- M4 可能增加顶层 `config` 块用于全局 Hook 设置。
- 保留键前缀：`x-` 用于用户扩展（将被静默忽略）。
