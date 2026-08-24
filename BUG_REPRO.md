# Bug 复现说明

## Bug 是什么
联系人事件缺少一项数据时，自动化处理会直接崩溃，导致事件后的动作和后续记录都没有完成；请把这类异常输入处理好。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为联系人事件解析 → 事件总线 → 自动化动作执行。

## 根因
1、文件：internal/domain/event_helpers.go、internal/events/bus.go、internal/service/service.go。2、符号：EventDataString、RequiredEventContactID、App.ProcessEvent。3、失效机制：事件缺少可选联系人数据时，字段读取和动作执行没有安全处理非字符串或缺失值，运行时异常中断了后续自动化。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/service -run '^TestBug019EventContactDataDoesNotCrashAutomation$' -count=1
```

## 错误信息
bug019_event_contact_data_test.go:27: automation panicked while handling contact data: interface conversion: interface {} is nil, not string

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：event_contact_nil
$ go test -v ./internal/service -run '^TestBug019EventContactDataDoesNotCrashAutomation$' -count=1
=== RUN   TestBug019EventContactDataDoesNotCrashAutomation
    bug019_event_contact_data_test.go:27: automation panicked while handling contact data: interface conversion: interface {} is nil, not string
--- FAIL: TestBug019EventContactDataDoesNotCrashAutomation (0.00s)
FAIL
FAIL	pulsegrid/internal/service	1.079s
FAIL
```
