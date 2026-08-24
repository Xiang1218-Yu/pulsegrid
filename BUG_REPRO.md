# Bug 复现说明

## Bug 是什么
运营人员通过自动化处理请求触发联系人动作时，资源已经不存在的场景被返回成笼统的服务异常，调用方无法区分可重试失败和数据问题，正常的自动化指标动作也需要保持可用，后续监控也会把这类请求归为系统故障。请先定位错误为什么没有按调用链传到接口层，说明影响范围和判断依据，代码不要修改。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP handler -> service.ProcessEvent -> domain error。

## 根因
1、文件：internal/domain/model.go、internal/service/service.go、internal/httpapi/server.go。2、符号：App.ProcessEvent、executeAction、StatusForError。3、失效机制：自动化动作产生的资源不存在错误跨服务层和 HTTP 层传播时被泛化，StatusForError 无法依据原始错误选择资源状态。

## 修复状态
Claude 主轨迹完成了诊断，未修改生产代码；文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/httpapi -run '^TestBug11ErrorPropagationReachesHTTPStatus$' -count=1
```

## 错误信息
bug11_error_propagation_test.go:34: POST /v1/automations/process

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：automation_error_status
$ go test -v ./internal/httpapi -run '^TestBug11ErrorPropagationReachesHTTPStatus$' -count=1
=== RUN   TestBug11ErrorPropagationReachesHTTPStatus
    bug11_error_propagation_test.go:34: POST /v1/automations/process
2026/08/24 09:17:36 WARN automation action failed automation_id=automation-11 error="resource not found"
2026/08/24 09:17:36 INFO http request method=POST path=/v1/automations/process status=500 duration=546.292µs
    bug11_error_propagation_test.go:53: automation error status = 500, want 404
--- FAIL: TestBug11ErrorPropagationReachesHTTPStatus (0.00s)
FAIL
FAIL	pulsegrid/internal/httpapi	2.037s
FAIL
```
