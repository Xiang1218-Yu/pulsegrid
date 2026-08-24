# Bug 复现说明

## Bug 是什么
实时事件页面关闭后，原来的请求偶尔还会继续占着连接，订阅数量和资源占用随时间累积，服务重启前都难以恢复；这边希望修复上下文结束后的生命周期处理，并保留正常事件投递。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP event stream → request context → events.Bus publish。

## 根因
1、文件：internal/httpapi/server.go、internal/events/bus.go、internal/service/service.go。2、符号：Server.events、Bus.Publish、App.publish。3、失效机制：事件流使用的发布上下文没有正确绑定请求生命周期，请求取消后发布和订阅仍等待，长连接资源无法及时释放。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/httpapi -run '^TestBug028EventStreamCancelStopsAfterCancel$' -count=1
```

## 错误信息
--- FAIL: TestBug028EventStreamCancelStopsAfterCancel (0.11s)

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：event_stream_cancel
$ go test -v ./internal/httpapi -run '^TestBug028EventStreamCancelStopsAfterCancel$' -count=1
=== RUN   TestBug028EventStreamCancelStopsAfterCancel
    bug028_event_stream_cancel_test.go:32: event stream did not stop after request cancellation
--- FAIL: TestBug028EventStreamCancelStopsAfterCancel (0.11s)
FAIL
FAIL	pulsegrid/internal/httpapi	1.139s
FAIL
```
