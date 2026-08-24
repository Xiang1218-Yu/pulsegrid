# Bug 复现说明

## Bug 是什么
客服查看一条已经不存在的投递记录时，接口返回了服务器故障状态，前端因此把普通的数据缺失提示成系统异常，后续重试和告警也被错误触发。请调整实现，让这类错误按实际情况返回并保持其他接口行为正常。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP API → service → store。

## 根因
1、文件：internal/service/service.go、internal/httpapi/http_errors.go、internal/httpapi/server.go。2、符号：App.GetDelivery、StatusForError、writeError。3、失效机制：投递不存在错误从存储层经服务层包装后进入 HTTP 状态映射，错误链没有被识别为资源缺失，最终响应为 500。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test ./internal/httpapi -run '^TestBug23WrappedErrorStatus$' -v -count=1
```

## 错误信息
bug23_wrapped_error_status_test.go:21: missing delivery should be a 404, got 500 with body {"error":"get delivery missing-delivery: resource not found"}

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：error_target
$ go test ./internal/httpapi -run '^TestBug23WrappedErrorStatus$' -v -count=1
=== RUN   TestBug23WrappedErrorStatus
2026/08/24 09:18:24 INFO http request method=GET path=/v1/deliveries/missing-delivery status=500 duration=314.5µs
    bug23_wrapped_error_status_test.go:21: missing delivery should be a 404, got 500 with body {"error":"get delivery missing-delivery: resource not found"}
--- FAIL: TestBug23WrappedErrorStatus (0.00s)
FAIL
FAIL	pulsegrid/internal/httpapi	1.327s
FAIL
```
