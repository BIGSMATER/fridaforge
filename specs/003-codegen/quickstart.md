# Quickstart: 声明式代码生成器

**功能**: 003-codegen | **日期**: 2026-05-18

## 概述

FridaForge Codegen 读取 YAML Hook 规格文件，自动生成可注入 Frida 的 JavaScript 脚本。

## 前置条件

- Go 1.25 (已满足)
- 一个合法的 YAML Hook 规格文件

## 编写 YAML 规格

```yaml
# hooks.yaml
app_package: com.example.myapp

hooks:
  # Java overload Hook — 拦截加密方法，保留原行为
  - class_name: com.example.myapp.Crypto
    method_name: encrypt
    hook_type: overload
    method_signature: java.lang.String, byte[]

  # Java override Hook — 完全替换反调试检查
  - class_name: com.example.myapp.DebugDetector
    method_name: isDebuggable
    hook_type: override
    # method_signature 留空 → 匹配第一个 isDebuggable()

  # Native Hook — 拦截 libnative.so 中的 open() 函数
  - class_name: ""          # native hook 不需要
    method_name: open
    hook_type: native
    module_name: libnative-lib.so
```

## 校验配置

```bash
fridaforge spec validate hooks.yaml
# ✓ 配置有效: hooks.yaml
#   目标应用: com.example.myapp
#   Hook 数量: 3
```

## 生成脚本

```bash
# 输出到 stdout
fridaforge spec generate hooks.yaml

# 输出到文件
fridaforge spec generate hooks.yaml -o hooks.js

# 仅生成指定方法的脚本
fridaforge spec generate hooks.yaml -t "com.example.myapp.Crypto.encrypt"
```

## 生成的脚本结构

```javascript
Java.perform(function() {
    // === com.example.myapp.Crypto.encrypt (overload) ===
    var Crypto = Java.use("com.example.myapp.Crypto");
    Crypto.encrypt.overload('java.lang.String', 'byte[]').implementation = function() {
        console.log("[+] Crypto.encrypt called");
        send(JSON.stringify({event: "enter", args: Array.prototype.slice.call(arguments)}));
        var result = this.encrypt.apply(this, arguments);
        send(JSON.stringify({event: "leave", result: result}));
        return result;
    };

    // === com.example.myapp.DebugDetector.isDebuggable (override) ===
    var DebugDetector = Java.use("com.example.myapp.DebugDetector");
    DebugDetector.isDebuggable.overload().implementation = function() {
        console.log("[+] DebugDetector.isDebuggable hooked (override)");
        send(JSON.stringify({event: "override", args: Array.prototype.slice.call(arguments)}));
        return false;
    };
});

// === Native: libnative-lib.so / open ===
var nativeModule = Process.findModuleByName("libnative-lib.so");
if (nativeModule === null) {
    console.log("[-] Module not found: libnative-lib.so");
} else {
    var targetAddr = Module.findExportByName("libnative-lib.so", "open");
    if (targetAddr === null) {
        console.log("[-] Export not found: open");
    } else {
        Interceptor.attach(targetAddr, {
            onEnter: function(args) { console.log("[+] Native open called"); },
            onLeave: function(retval) { console.log("[+] Native open returned: " + retval); }
        });
    }
}
```

## 注入目标设备

```bash
# 将生成的脚本注入目标设备
frida -U -f com.example.myapp -l hooks.js --no-pause
```

## 下一步

- 阅读 [plan.md](./plan.md) 了解技术架构
- 阅读 [contracts/codegen-api.md](./contracts/codegen-api.md) 了解 Go API
- 等待 M4: MCP Server 集成（让大模型自动生成 Hook 脚本）
