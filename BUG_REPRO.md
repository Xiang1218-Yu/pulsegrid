# Bug 复现说明

## Bug 是什么
高并发批处理完成后，汇总成功数偶尔少于实际完成项，监控因此会显示错误的处理结果，压力升高时还可能伴随测试中的竞态告警。请调整实现，保证并发执行、结果汇总和单工作线程场景都稳定正确。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为batch.Run -> worker goroutines -> report summary。

## 根因
1、文件：internal/batch/batch_limits.go、internal/batch/batch_status.go、internal/batch/processor.go。2、符号：Processor.Run、runStats.record、NormalizeWorkers。3、失效机制：并行 worker 共同更新运行统计和结果汇总，没有可靠同步，完成数与报告内容在竞态下出现丢失。并发验证使用 -race -count=20，并通过固定 goroutine 协调、channel 或 WaitGroup 同步屏障保证复现稳定。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v -race ./internal/batch -run '^TestBug13ParallelSummaryMatchesResults$' -count=20
```

## 错误信息
WARNING: DATA RACE

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：parallel_summary_integrity
$ go test -v -race ./internal/batch -run '^TestBug13ParallelSummaryMatchesResults$' -count=20
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
==================
WARNING: DATA RACE
Read at 0x00c000115710 by goroutine 12:
  pulsegrid/internal/batch.(*runStats).record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/batch_status.go:10 +0x1a4
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/processor.go:84 +0x194

Previous write at 0x00c000115710 by goroutine 8:
  pulsegrid/internal/batch.(*runStats).record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/batch_status.go:10 +0x1b4
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/processor.go:84 +0x194

Goroutine 12 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/processor.go:79 +0x188
  pulsegrid/internal/batch.TestBug13ParallelSummaryMatchesResults.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/bug13_parallel_summary_test.go:37 +0x68

Goroutine 8 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/processor.go:79 +0x188
  pulsegrid/internal/batch.TestBug13ParallelSummaryMatchesResults.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/bug13_parallel_summary_test.go:37 +0x68
==================
==================
WARNING: DATA RACE
Write at 0x00c000115710 by goroutine 13:
  pulsegrid/internal/batch.(*runStats).record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/batch_status.go:10 +0x1b4
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/processor.go:84 +0x194

Previous write at 0x00c000115710 by goroutine 11:
  pulsegrid/internal/batch.(*runStats).record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/batch_status.go:10 +0x1b4
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/processor.go:84 +0x194

Goroutine 13 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/processor.go:79 +0x188
  pulsegrid/internal/batch.TestBug13ParallelSummaryMatchesResults.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/bug13_parallel_summary_test.go:37 +0x68

Goroutine 11 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/processor.go:79 +0x188
  pulsegrid/internal/batch.TestBug13ParallelSummaryMatchesResults.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ef57gzdl/env/internal/batch/bug13_parallel_summary_test.go:37 +0x68
==================
    bug13_parallel_summary_test.go:44: parallel succeeded = 251, want 256
    testing.go:1712: race detected during execution of test
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 217, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 245, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 215, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 181, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 231, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 208, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 225, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 234, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 241, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 239, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 247, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
--- PASS: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 236, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 234, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 232, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 245, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 238, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 247, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
=== RUN   TestBug13ParallelSummaryMatchesResults
    bug13_parallel_summary_test.go:14: Processor.Run report.Summary
    bug13_parallel_summary_test.go:44: parallel succeeded = 231, want 256
--- FAIL: TestBug13ParallelSummaryMatchesResults (0.00s)
FAIL
FAIL	pulsegrid/internal/batch	1.727s
FAIL
```
