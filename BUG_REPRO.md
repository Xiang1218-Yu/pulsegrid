# Bug 复现说明

## Bug 是什么
用户取消创建联系人请求后，后台自动化仍可能继续处理这次事件并改写联系人状态，导致取消后的请求留下难以追踪的副作用，请修复这条上下文生命周期并让正常请求继续按原流程完成。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为request context → event bus → automation service → contact repository。

## 根因
1、文件：internal/service/service.go、internal/events/bus.go、internal/jobs/jobs.go。2、符号：App.publish、Events.Publish、ProcessEvent。3、失效机制：异步自动化沿用了调用方的 context 生命周期，取消请求后后台事件仍可能继续执行并产生联系人状态副作用。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/service -run '^TestBug007ContextLifecycleCancelledEvent$' -count=1
```

## 错误信息
--- FAIL: TestBug007ContextLifecycleCancelledEvent (0.05s)

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：context_cancelled_event
$ go test -v ./internal/service -run '^TestBug007ContextLifecycleCancelledEvent$' -count=1
=== RUN   TestBug007ContextLifecycleCancelledEvent
    bug007_context_lifecycle_test.go:35: cancelled event changed contact status to "unsubscribed"
--- FAIL: TestBug007ContextLifecycleCancelledEvent (0.05s)
FAIL
FAIL	pulsegrid/internal/service	1.398s
FAIL
```
