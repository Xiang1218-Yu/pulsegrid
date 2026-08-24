# Bug 复现说明

## Bug 是什么
监控页面持续订阅事件时，客户端结束连接后系统仍会在后续事件到达时崩溃，影响同一进程中的其他监控请求。请修复这条资源生命周期问题，并保留正常订阅接收事件的能力。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP event stream → events bus → subscription lifecycle。

## 根因
1、文件：internal/events/bus.go、internal/httpapi/server.go、internal/service/service.go。2、符号：Subscription.Close、Bus.Publish、Server.events。3、失效机制：事件订阅关闭、总线发送和 HTTP 连接退出的时序没有形成互斥关系，关闭后的订阅仍可能被发布方访问。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test ./internal/events -run '^TestBug24SubscriptionLifecycle$' -v -count=1
```

## 错误信息
bug24_subscription_lifecycle_test.go:29: publishing after a closed subscription panicked

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：lifecycle_target
$ go test ./internal/events -run '^TestBug24SubscriptionLifecycle$' -v -count=1
=== RUN   TestBug24SubscriptionLifecycle
    bug24_subscription_lifecycle_test.go:29: publishing after a closed subscription panicked
--- FAIL: TestBug24SubscriptionLifecycle (0.00s)
FAIL
FAIL	pulsegrid/internal/events	0.985s
FAIL
```
