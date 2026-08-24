# Bug 复现说明

## Bug 是什么
前端分页查看联系人时，修改当前页返回的数据会意外影响后续页面或原始列表，刷新后出现联系人顺序和内容被改写的情况，请修复分页结果的一致性并保留正常分页边界。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为pagination request → page window helper → caller-owned slice。

## 根因
1、文件：internal/pagination/limits.go、internal/pagination/page.go、internal/pagination/cursor.go。2、符号：pagination.Apply、pagination.ApplyCursor、cloneWindow。3、失效机制：分页、游标和窗口辅助逻辑直接复用了输入 slice 的底层数组，调用方修改返回值后污染了后续查询状态。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/pagination -run '^TestBug008PageWindowIsolation$' -count=1
```

## 错误信息
--- FAIL: TestBug008PageWindowIsolation (0.00s)

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：page_window_isolation
$ go test -v ./internal/pagination -run '^TestBug008PageWindowIsolation$' -count=1
=== RUN   TestBug008PageWindowIsolation
    bug008_page_window_test.go:14: page result shares source slice storage: source=[changed second third]
--- FAIL: TestBug008PageWindowIsolation (0.00s)
FAIL
FAIL	pulsegrid/internal/pagination	1.747s
FAIL
```
