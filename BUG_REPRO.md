# Bug 复现说明

## Bug 是什么
客户端频繁断开实时事件流后，服务运行一段时间出现事件推送延迟、资源占用持续上升，重连也会变得不稳定，连接有效时的事件接收不能受到影响。请修复连接结束后的资源生命周期处理，并保留有效连接的实时推送能力。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP event stream -> subscription -> event bus。

## 根因
1、文件：internal/events/event_helpers.go、internal/events/bus.go、internal/httpapi/server.go。2、符号：Subscription.Close、Bus.Publish、events handler。3、失效机制：实时事件流断开后的订阅注销与发布链路没有协调好，失效订阅持续留在总线中，造成资源生命周期泄漏。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/httpapi -run '^TestBug14EventSubscriptionReleasedAfterDisconnect$' -count=1
```

## 错误信息
bug14_event_subscription_test.go:48: subscriber count = 1, want 0

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：event_subscription_release
$ go test -v ./internal/httpapi -run '^TestBug14EventSubscriptionReleasedAfterDisconnect$' -count=1
=== RUN   TestBug14EventSubscriptionReleasedAfterDisconnect
    bug14_event_subscription_test.go:30: GET /v1/events
2026/08/24 09:17:48 INFO http request method=GET path=/v1/events status=200 duration=1.127ms
    bug14_event_subscription_test.go:48: subscriber count = 1, want 0
--- FAIL: TestBug14EventSubscriptionReleasedAfterDisconnect (1.00s)
FAIL
FAIL	pulsegrid/internal/httpapi	2.892s
FAIL
```
