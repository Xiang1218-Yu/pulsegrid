# Bug 复现说明

## Bug 是什么
高峰期多个任务同时记录投递指标时，指标总数偶尔少于实际完成数，并且服务日志出现不稳定的并发异常，影响运营统计。请修复这个问题，同时保证单条指标记录和读取仍然正常。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP 指标入口 → 服务指标记录 → 并发指标存储。

## 根因
1、文件：internal/analytics/metrics.go、internal/service/service.go、internal/httpapi/server.go。2、符号：Store.Record、Store.RecordValue、Server.metrics。3、失效机制：多个 goroutine 同时更新共享指标 slice 和计数，没有同步保护，导致数据竞态、追加丢失和统计结果不完整。并发验证使用 -race -count=20，并通过固定 goroutine 协调、channel 或 WaitGroup 同步屏障保证复现稳定。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v -race ./internal/analytics -run '^TestBug003ConcurrentMetricWritesAreCoordinated$' -count=20
```

## 错误信息
WARNING: DATA RACE

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：concurrent_metric_writes
$ go test -v -race ./internal/analytics -run '^TestBug003ConcurrentMetricWritesAreCoordinated$' -count=20
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
==================
WARNING: DATA RACE
Read at 0x00c000091038 by goroutine 8:
  pulsegrid/internal/analytics.(*Store).Record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:45 +0x1c8
  pulsegrid/internal/analytics.(*Store).RecordValue()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:53 +0x288
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:18 +0xd4
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.gowrap1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:20 +0x38

Previous write at 0x00c000091038 by goroutine 22:
  pulsegrid/internal/analytics.(*Store).Record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:45 +0x270
  pulsegrid/internal/analytics.(*Store).RecordValue()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:53 +0x288
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:18 +0xd4
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.gowrap1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:20 +0x38

Goroutine 8 (running) created at:
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:15 +0xe4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2036 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2101 +0x34

Goroutine 22 (running) created at:
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:15 +0xe4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2036 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2101 +0x34
==================
==================
WARNING: DATA RACE
Write at 0x00c000091038 by goroutine 8:
  pulsegrid/internal/analytics.(*Store).Record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:45 +0x270
  pulsegrid/internal/analytics.(*Store).RecordValue()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:53 +0x288
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:18 +0xd4
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.gowrap1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:20 +0x38

Previous write at 0x00c000091038 by goroutine 22:
  pulsegrid/internal/analytics.(*Store).Record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:45 +0x270
  pulsegrid/internal/analytics.(*Store).RecordValue()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:53 +0x288
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:18 +0xd4
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.gowrap1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:20 +0x38

Goroutine 8 (running) created at:
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:15 +0xe4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2036 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2101 +0x34

Goroutine 22 (running) created at:
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:15 +0xe4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2036 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2101 +0x34
==================
==================
WARNING: DATA RACE
Write at 0x00c0000f34a8 by goroutine 8:
  pulsegrid/internal/analytics.(*Store).Record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:45 +0x220
  pulsegrid/internal/analytics.(*Store).RecordValue()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:53 +0x288
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:18 +0xd4
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.gowrap1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:20 +0x38

Previous write at 0x00c0000f34a8 by goroutine 19:
  pulsegrid/internal/analytics.(*Store).Record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:45 +0x220
  pulsegrid/internal/analytics.(*Store).RecordValue()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:53 +0x288
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:18 +0xd4
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.gowrap1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:20 +0x38

Goroutine 8 (running) created at:
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:15 +0xe4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2036 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2101 +0x34

Goroutine 19 (running) created at:
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:15 +0xe4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2036 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2101 +0x34
==================
==================
WARNING: DATA RACE
Read at 0x00c0000f2bb8 by goroutine 20:
  runtime.growslice()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/runtime/slice.go:178 +0x0
  pulsegrid/internal/analytics.(*Store).Record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:45 +0x1f8
  pulsegrid/internal/analytics.(*Store).RecordValue()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:53 +0x288
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:18 +0xd4
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.gowrap1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:20 +0x38

Previous write at 0x00c0000f2bb8 by goroutine 22:
  pulsegrid/internal/analytics.(*Store).Record()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:45 +0x220
  pulsegrid/internal/analytics.(*Store).RecordValue()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/metrics.go:53 +0x288
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.func1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:18 +0xd4
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated.gowrap1()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:20 +0x38

Goroutine 20 (running) created at:
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:15 +0xe4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2036 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2101 +0x34

Goroutine 22 (finished) created at:
  pulsegrid/internal/analytics.TestBug003ConcurrentMetricWritesAreCoordinated()
      /private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-f158xx3j/env/internal/analytics/bug003_concurrent_metric_writes_test.go:15 +0xe4
  testing.tRunner()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2036 +0x164
  testing.(*T).Run.gowrap1()
      /opt/homebrew/Cellar/go/1.26.5/libexec/src/testing/testing.go:2101 +0x34
==================
    bug003_concurrent_metric_writes_test.go:24: metric count = 154, want 640
    testing.go:1712: race detected during execution of test
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 131, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 121, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 141, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 129, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 99, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 83, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 107, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 123, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 129, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 137, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 126, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 116, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 92, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 144, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 123, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 112, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 83, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 77, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
=== RUN   TestBug003ConcurrentMetricWritesAreCoordinated
    bug003_concurrent_metric_writes_test.go:24: metric count = 151, want 640
--- FAIL: TestBug003ConcurrentMetricWritesAreCoordinated (0.00s)
FAIL
FAIL	pulsegrid/internal/analytics	3.154s
FAIL
```
