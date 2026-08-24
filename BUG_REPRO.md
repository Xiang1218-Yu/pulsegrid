# Bug 复现说明

## Bug 是什么
系统用标准事件创建器触发联系人自动化时，事件已经进入处理流程却没有给联系人打上应有标签，直接传入完整事件反而正常；请先诊断原因，不要修改代码。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为标准事件创建器 → 事件总线 → 联系人自动化服务。

## 根因
1、文件：internal/domain/event_helpers.go、internal/events/bus.go、internal/service/service.go。2、符号：events.NewEvent、ProcessEvent、App.publish。3、失效机制：事件创建器生成的数据字段与直接构造事件的字段口径不一致，跨事件总线进入自动化后关键联系人信息没有传到动作执行层。

## 修复状态
Claude 主轨迹完成了诊断，未修改生产代码；文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/service -run '^TestBug018EventContactFlowReachesAutomation$' -count=1
```

## 错误信息
2026/08/24 09:18:01 WARN automation action failed automation_id=automation-018 error="resource not found"

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：event_contact_flow
$ go test -v ./internal/service -run '^TestBug018EventContactFlowReachesAutomation$' -count=1
=== RUN   TestBug018EventContactFlowReachesAutomation
2026/08/24 09:18:01 WARN automation action failed automation_id=automation-018 error="resource not found"
    bug018_event_contact_flow_test.go:31: contact tags after events.NewEvent -> ProcessEvent = []string(nil); want vip
--- FAIL: TestBug018EventContactFlowReachesAutomation (0.00s)
FAIL
FAIL	pulsegrid/internal/service	1.002s
FAIL
```
