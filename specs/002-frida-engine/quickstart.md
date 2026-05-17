# Quickstart: Frida 并发调度引擎

**功能**: 002-frida-engine | **日期**: 2026-05-12

## 前置环境

1. **安装 frida-core-devkit**:
   ```bash
   # 从 Frida releases 下载对应平台的 frida-core-devkit
   # https://github.com/frida/frida/releases
   # 解压后:
   sudo cp frida-core.h /usr/local/include/
   sudo cp libfrida-core.a /usr/local/lib/
   ```

2. **确保 frida-server 在目标设备运行**:
   ```bash
   adb push frida-server /data/local/tmp/
   adb shell chmod 755 /data/local/tmp/frida-server
   adb shell /data/local/tmp/frida-server &
   ```

## 编译

```bash
go build ./cmd/fridaforge/
```

## 基本使用

### 枚举设备
```bash
./fridaforge device list
# ID                    NAME                  TYPE
# emulator-5554         Android Emulator      emulator
# R5CT1234ABCD          Samsung Galaxy S21    usb
```

### Attach 并注入脚本
```go
import "github.com/bigsmater/fridaforge/pkg/fridaengine"

func main() {
    engine, err := fridaengine.NewEngineWithDefaults()
    if err != nil {
        log.Fatal(err)
    }
    defer engine.Close()

    ctx := context.Background()

    // 1. 枚举设备
    devices, err := engine.ListDevices(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 2. Attach
    session, err := engine.Attach(ctx, devices[0].ID, "com.example.app")
    if err != nil {
        log.Fatal(err)
    }
    defer session.Detach()

    // 3. 注入脚本
    err = session.CreateScript(`
        Java.perform(function() {
            var MainActivity = Java.use("com.example.MainActivity");
            MainActivity.onCreate.implementation = function() {
                console.log("onCreate called!");
                this.onCreate();
            };
        });
    `)
    if err != nil {
        log.Fatal(err)
    }

    // 4. 接收消息
    for msg := range session.Messages() {
        fmt.Printf("[%s] %s\n", msg.Type, msg.Payload)
    }
}
```

## 测试

```bash
# 单元测试（无需 frida-server）
go test ./pkg/fridaengine/ -v

# 集成测试（需要 frida-server + Android 设备）
go test -tags=integration ./pkg/fridaengine/ -v
```

## 下一步

- M3: 声明式代码生成器 (`pkg/codegen/`) — 从 YAML spec 自动生成 Frida JS 脚本
- M4: MCP Server 集成 — 让 LLM 通过 MCP 协议调用 FridaForge
