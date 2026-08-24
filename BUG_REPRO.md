# Bug 复现说明

## Bug 是什么
供应商重复发送同一条投递回调后，活动的已发送数量会被重复累加，报表和真实投递状态逐渐对不上；请修复重复回调下的状态一致性。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为投递回调 → 投递状态规则 → 活动统计服务。

## 根因
1、文件：internal/domain/delivery_rules.go、internal/service/delivery_plan.go、internal/service/service.go。2、符号：RegisterDeliveryEvent、ApplyDeliveryProgress、DeliveryEventShouldCount。3、失效机制：投递回调没有区分状态是否真正前进，重复回调再次触发活动计数、指标和事件，造成累计状态污染。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/service -run '^TestBug020RepeatedDeliveryEventKeepsCampaignCount$' -count=1
```

## 错误信息
bug020_repeated_delivery_event_test.go:27: campaign sent count after repeated callback = 2; want 1

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：repeated_delivery_event
$ go test -v ./internal/service -run '^TestBug020RepeatedDeliveryEventKeepsCampaignCount$' -count=1
=== RUN   TestBug020RepeatedDeliveryEventKeepsCampaignCount
    bug020_repeated_delivery_event_test.go:27: campaign sent count after repeated callback = 2; want 1
--- FAIL: TestBug020RepeatedDeliveryEventKeepsCampaignCount (0.00s)
FAIL
FAIL	pulsegrid/internal/service	1.134s
FAIL
```
