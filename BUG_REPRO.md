# Bug 复现说明

## Bug 是什么
运营人员在批量整理联系人并查看分页结果时，修改当前结果会连带改变后续批次的数据，重新处理后列表内容出现覆盖，影响了后面的发送准备。请先说明问题原因和影响范围，不要修改代码。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为批处理 → pagination → store。

## 根因
1、文件：internal/batch/processor.go、internal/pagination/page.go、internal/store/filters.go。2、符号：store.FilterSubscribed、pagination.Apply、batch.Chunk。3、失效机制：筛选、分页和分块结果暴露了调用方 slice 的底层数组，后续编辑返回值会反向覆盖原始联系人集合。

## 修复状态
Claude 主轨迹完成了诊断，未修改生产代码；文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test ./internal/pagination -run '^TestBug22SliceViewIsolation$' -v -count=1
```

## 错误信息
--- FAIL: TestBug22SliceViewIsolation (0.00s)

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：slice_target
$ go test ./internal/pagination -run '^TestBug22SliceViewIsolation$' -v -count=1
=== RUN   TestBug22SliceViewIsolation
    bug22_slice_view_test.go:21: filtered contacts must not mutate the caller's slice
    bug22_slice_view_test.go:28: page items must not mutate the caller's slice
    bug22_slice_view_test.go:35: chunks must not expose the caller's backing array
--- FAIL: TestBug22SliceViewIsolation (0.00s)
FAIL
FAIL	pulsegrid/internal/pagination	1.060s
FAIL
```
