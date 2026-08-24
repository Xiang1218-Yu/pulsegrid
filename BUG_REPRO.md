# Bug 复现说明

## Bug 是什么
联系人完成退订后，列表页面仍把这条记录显示在可订阅联系人视图中，后续筛选和发送准备可能继续选中它，造成状态与实际操作不一致。请修复这个问题，并保留仍处于订阅状态的联系人在正常视图中的行为。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为退订 HTTP 入口 → 联系人状态模型 → 联系人列表查询。

## 根因
1、文件：internal/domain/model.go、internal/service/service.go、internal/httpapi/server.go。2、符号：Contact.Unsubscribe、App.UnsubscribeContact、ListContacts。3、失效机制：退订状态在领域对象、服务持久化和列表筛选之间没有保持同一状态事实，成功响应后的查询仍可见旧订阅状态。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/httpapi -run '^TestBug004UnsubscribeStateAcrossViews$' -count=1
```

## 错误信息
--- FAIL: TestBug004UnsubscribeStateAcrossViews (0.00s)

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：unsubscribe_view
$ go test -v ./internal/httpapi -run '^TestBug004UnsubscribeStateAcrossViews$' -count=1
=== RUN   TestBug004UnsubscribeStateAcrossViews
2026/08/24 09:17:09 INFO http request method=POST path=/v1/contacts/contact-1787534229904991000-1/unsubscribe status=200 duration=166.542µs
    bug004_unsubscribe_state_across_views_test.go:37: unsubscribe response still reports subscribed: {"id":"contact-1787534229904991000-1","organization_id":"org-1","email":"person@example.com","name":"Person","status":"subscribed","locale":"en-US","timezone":"UTC","created_at":"2026-08-24T01:17:09.904991Z","updated_at":"2026-08-24T01:17:09.905275Z"}
--- FAIL: TestBug004UnsubscribeStateAcrossViews (0.00s)
FAIL
FAIL	pulsegrid/internal/httpapi	2.726s
FAIL
```
