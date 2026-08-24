# Bug 复现说明

## Bug 是什么
后台任务第一次执行失败后进入重试，上一轮的请求上下文一直没有结束，任务多了以后会积累资源压力；请修复重试生命周期，同时保留单次执行的正常行为。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为任务执行 → 重试调度 → 单次尝试 Context。

## 根因
1、文件：internal/jobs/job_payload.go、internal/jobs/jobs.go、internal/jobs/worker_config.go。2、符号：Queue.execute、runAttempt、normalizeAttemptTimeout。3、失效机制：重试循环把每轮尝试的 context 生命周期拖到整个任务结束，超时计时器和相关资源不能在下一轮开始前释放。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/jobs -run '^TestBug017RetryReleasesAttemptContext$' -count=1
```

## 错误信息
--- FAIL: TestBug017RetryReleasesAttemptContext (0.01s)

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：retry_context
$ go test -v ./internal/jobs -run '^TestBug017RetryReleasesAttemptContext$' -count=1
=== RUN   TestBug017RetryReleasesAttemptContext
    bug017_retry_releases_attempt_context_test.go:46: previous attempt context was not released
--- FAIL: TestBug017RetryReleasesAttemptContext (0.01s)
FAIL
FAIL	pulsegrid/internal/jobs	1.390s
FAIL
```
