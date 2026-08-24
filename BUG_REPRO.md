# Bug 复现说明

## Bug 是什么
客户端结束实时事件订阅后，恰好有事件正在发送时服务可能崩溃，或者发布方一直等不到结果，连接关闭后资源也无法稳定回收，请修复这条关闭流程并保留正常订阅收消息的行为。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP event stream → subscription → event bus publisher。

## 根因
1、文件：internal/events/bus.go、internal/events/topic_names.go、internal/httpapi/server.go。2、符号：Bus.Publish、Subscription.Close、Server.events。3、失效机制：订阅关闭与发布发送之间缺少安全的生命周期协调，发布方可能向已关闭 channel 发送并触发运行时崩溃。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/events -run '^TestBug009SubscriptionLifecycleClose$' -count=1
```

## 错误信息
bug009_subscription_lifecycle_test.go:38: publisher panicked while subscription closed: send on closed channel

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：subscription_lifecycle_close
$ go test -v ./internal/events -run '^TestBug009SubscriptionLifecycleClose$' -count=1
=== RUN   TestBug009SubscriptionLifecycleClose
    bug009_subscription_lifecycle_test.go:38: publisher panicked while subscription closed: send on closed channel
--- FAIL: TestBug009SubscriptionLifecycleClose (0.00s)
FAIL
FAIL	pulsegrid/internal/events	0.989s
FAIL
```
