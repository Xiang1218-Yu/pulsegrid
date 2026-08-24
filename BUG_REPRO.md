# Bug 复现说明

## Bug 是什么
运营人员启动一场定向活动时，活动记录已经进入运行状态，但后台处理暂时不可用会让接口只返回笼统的服务器错误，调用方无法判断是否需要稍后重试，麻烦把这条失败链路修好并保留正常启动行为。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP handler → campaign service → delivery queue → error response。

## 根因
1、文件：internal/jobs/jobs.go、internal/service/service.go、internal/httpapi/http_errors.go、internal/httpapi/server.go。2、符号：Queue.Enqueue、StartCampaign、StatusForError、writeError。3、失效机制：队列入队失败从后台处理层返回到活动启动接口时被重新包装或遮蔽，错误映射层无法识别服务不可用语义。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/httpapi -run '^TestBug006CampaignErrorQueueFailure$' -count=1
```

## 错误信息
bug006_campaign_error_test.go:25: expected queue failure to reach HTTP as 503, got 500: {"error":"queue delivery delivery-1787534229433186000-6: queue is not started"}

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：campaign_error_queue_failure
$ go test -v ./internal/httpapi -run '^TestBug006CampaignErrorQueueFailure$' -count=1
=== RUN   TestBug006CampaignErrorQueueFailure
2026/08/24 09:17:09 INFO http request method=POST path=/v1/campaigns/campaign-1787534229433160000-5/start status=500 duration=143.208µs
    bug006_campaign_error_test.go:25: expected queue failure to reach HTTP as 503, got 500: {"error":"queue delivery delivery-1787534229433186000-6: queue is not started"}
--- FAIL: TestBug006CampaignErrorQueueFailure (0.00s)
FAIL
FAIL	pulsegrid/internal/httpapi	2.270s
FAIL
```
