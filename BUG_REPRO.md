# Bug 复现说明

## Bug 是什么
运营人员按组织查看概览时，页面里的组织数和投递统计可能混入其他组织的数据，单组织页面与全局概览的口径不一致，请把这条查询链路修好并确保组织范围和未筛选概览都正确。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为overview HTTP handler → service aggregation → organization and delivery stores。

## 根因
1、文件：internal/store/memory.go、internal/service/service.go、internal/httpapi/server.go。2、符号：App.Overview、ListOrganizations、ListDeliveries。3、失效机制：概览查询的组织范围只应用到部分统计维度，组织列表或投递汇总仍使用全局数据，响应内各项统计口径不一致。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/httpapi -run '^TestBug010OverviewScopeIsolation$' -count=1
```

## 错误信息
--- FAIL: TestBug010OverviewScopeIsolation (0.00s)

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：overview_scope_isolation
$ go test -v ./internal/httpapi -run '^TestBug010OverviewScopeIsolation$' -count=1
=== RUN   TestBug010OverviewScopeIsolation
2026/08/24 09:17:35 INFO http request method=GET path=/v1/overview status=200 duration=160.417µs
    bug010_overview_scope_test.go:33: overview leaked another organization: {GeneratedAt:2026-08-24 01:17:35.915398 +0000 UTC Organizations:2 Contacts:1 Subscribed:1 Campaigns:1 RunningCampaigns:0 Deliveries:1 Delivered:1 Opened:0 Clicked:0 OpenRate:0 ClickRate:0}
--- FAIL: TestBug010OverviewScopeIsolation (0.00s)
FAIL
FAIL	pulsegrid/internal/httpapi	1.758s
FAIL
```
