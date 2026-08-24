# Bug 复现说明

## Bug 是什么
运营请求结束后，仍有事件发布操作一直等待，导致后续自动化动作没有及时完成，部分状态更新会延迟。请先定位原因并说明影响范围，不要修改代码。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP 事件流 → 服务事件发布 → 事件总线订阅。

## 根因
1、文件：internal/events/bus.go、internal/service/service.go、internal/httpapi/server.go。2、符号：Bus.Publish、Subscription.Close、Server.events。3、失效机制：请求上下文取消信号没有沿事件发布链路生效，Bus.Publish 仍等待订阅者，HTTP 请求退出后后台发布因此无法及时结束。

## 修复状态
Claude 主轨迹完成了诊断，未修改生产代码；文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/events -run '^TestBug001PublishStopsOnCancel$' -count=1
```

## 错误信息
--- FAIL: TestBug001PublishStopsOnCancel (0.10s)

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：publish_cancel
$ go test -v ./internal/events -run '^TestBug001PublishStopsOnCancel$' -count=1
=== RUN   TestBug001PublishStopsOnCancel
    bug001_publish_stops_on_cancel_test.go:33: publish did not stop after cancellation
--- FAIL: TestBug001PublishStopsOnCancel (0.10s)
FAIL
FAIL	pulsegrid/internal/events	1.559s
FAIL
```
