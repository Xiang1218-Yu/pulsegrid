# Bug 复现说明

## Bug 是什么
运营批量并行处理消息后，结果报告偶尔出现重复或缺少记录，成功数量也和实际完成数对不上，影响后续重试判断。请修复并行处理中的一致性问题；验证时使用带 race 检查的固定 20 次运行，借助同步屏障保证稳定复现。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为batch processor → worker goroutines → report aggregation。

## 根因
1、文件：internal/batch/processor.go、internal/batch/batch_status.go、internal/batch/batch_limits.go。2、符号：Processor.Run、runStats.record、NormalizeWorkers。3、失效机制：多个 worker 并行追加共享结果 slice 并更新统计，没有同步保护，报告内容和成功计数会在竞态下损坏。并发验证使用 -race -count=20，并通过固定 goroutine 协调、channel 或 WaitGroup 同步屏障保证复现稳定。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test ./internal/batch -run '^TestBug25ParallelReportIntegrity$' -v -race -count=20
```

## 错误信息
WARNING: DATA RACE

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：concurrency_target
$ go test ./internal/batch -run '^TestBug25ParallelReportIntegrity$' -v -race -count=20
=== RUN   TestBug25ParallelReportIntegrity
==================
WARNING: DATA RACE
Write at 0x00c0001dc210 by goroutine 14:
  pulsegrid/internal/batch.recordResult()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/batch_status.go:4 +0x1e0
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:86 +0x11c

Previous read at 0x00c0001dc210 by goroutine 13:
  pulsegrid/internal/batch.recordResult()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/batch_status.go:4 +0x144
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:86 +0x11c

Goroutine 14 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:82 +0x19c
  pulsegrid/internal/batch.TestBug25ParallelReportIntegrity.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/bug25_parallel_report_test.go:30 +0x6c

Goroutine 13 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:82 +0x19c
  pulsegrid/internal/batch.TestBug25ParallelReportIntegrity.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/bug25_parallel_report_test.go:30 +0x6c
==================
==================
WARNING: DATA RACE
Write at 0x00c0001dc210 by goroutine 12:
  pulsegrid/internal/batch.recordResult()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/batch_status.go:4 +0x1e0
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:86 +0x11c

Previous write at 0x00c0001dc210 by goroutine 10:
  pulsegrid/internal/batch.recordResult()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/batch_status.go:4 +0x1e0
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:86 +0x11c

Goroutine 12 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:82 +0x19c
  pulsegrid/internal/batch.TestBug25ParallelReportIntegrity.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/bug25_parallel_report_test.go:30 +0x6c

Goroutine 10 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:82 +0x19c
  pulsegrid/internal/batch.TestBug25ParallelReportIntegrity.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/bug25_parallel_report_test.go:30 +0x6c
==================
==================
WARNING: DATA RACE
Write at 0x00c000281ec8 by goroutine 15:
  pulsegrid/internal/batch.recordResult()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/batch_status.go:4 +0x198
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:86 +0x11c

Previous write at 0x00c000281ec8 by goroutine 9:
  pulsegrid/internal/batch.recordResult()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/batch_status.go:4 +0x198
  pulsegrid/internal/batch.(*Processor).Run.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:86 +0x11c

Goroutine 15 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:82 +0x19c
  pulsegrid/internal/batch.TestBug25ParallelReportIntegrity.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/bug25_parallel_report_test.go:30 +0x6c

Goroutine 9 (running) created at:
  pulsegrid/internal/batch.(*Processor).Run()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/processor.go:82 +0x19c
  pulsegrid/internal/batch.TestBug25ParallelReportIntegrity.func2()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-ukhibjxu/env/internal/batch/bug25_parallel_report_test.go:30 +0x6c
==================
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 107
    testing.go:1712: race detected during execution of test
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 111
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 110
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 101
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 121
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 106
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 114
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 113
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 112
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 118
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 110
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 114
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 117
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 108
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 100
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 117
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 118
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 117
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 118
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
=== RUN   TestBug25ParallelReportIntegrity
    bug25_parallel_report_test.go:39: parallel report should contain 64 results, got 121
--- FAIL: TestBug25ParallelReportIntegrity (0.00s)
FAIL
FAIL	pulsegrid/internal/batch	1.295s
FAIL
```
