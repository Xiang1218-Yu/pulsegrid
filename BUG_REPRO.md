# Bug 复现说明

## Bug 是什么
联系人列表做过排序或编辑后，后续查询偶尔会带着上一次操作留下的顺序和标签，页面刷新后看到的内容会被悄悄改写；请修复这种列表状态串联问题，保证正常查询仍能返回完整联系人。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为repository ListContacts → service.SortContactsByEngagement → domain.Contact.Clone。

## 根因
1、文件：internal/store/memory.go、internal/service/contact_usecase.go、internal/domain/model.go。2、符号：Contact.Clone、SortContactsByEngagement、ListContacts。3、失效机制：联系人对象和 Tags slice 在仓储、服务排序与返回值之间共享底层存储，调用方修改结果后污染了仓储状态。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/store -run '^TestBug030ContactSliceConsistencyKeepsState$' -count=1
```

## 错误信息
bug030_contact_slice_consistency_test.go:38: stored contact tag = "mutated", want customer

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：contact_slice_state
$ go test -v ./internal/store -run '^TestBug030ContactSliceConsistencyKeepsState$' -count=1
=== RUN   TestBug030ContactSliceConsistencyKeepsState
    bug030_contact_slice_consistency_test.go:38: stored contact tag = "mutated", want customer
--- FAIL: TestBug030ContactSliceConsistencyKeepsState (0.00s)
FAIL
FAIL	pulsegrid/internal/store	1.974s
FAIL
```
