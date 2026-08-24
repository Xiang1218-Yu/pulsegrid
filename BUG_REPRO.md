# Bug 复现说明

## Bug 是什么
更新联系人时，遇到不存在的记录，接口返回了笼统的服务器错误，客户端无法按资源不存在处理，相关流程也因此中断。请先诊断原因并说明影响范围，不要修改代码。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为HTTP 联系人更新 → 服务层错误包装 → HTTP 状态分类。

## 根因
1、文件：internal/service/service.go、internal/httpapi/server.go、internal/httpapi/http_errors.go。2、符号：App.UpdateContact、contactsUpdate、StatusForError。3、失效机制：服务层返回的资源不存在错误经过包装后没有保留可识别的错误链，HTTP 状态分类把它当成了服务器故障。

## 修复状态
Claude 主轨迹完成了诊断，未修改生产代码；文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/httpapi -run '^TestBug002WrappedNotFoundReachesHTTP$' -count=1
```

## 错误信息
bug002_wrapped_not_found_test.go:16: HTTP status = 500, want 404 for wrapped not-found error

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：wrapped_not_found
$ go test -v ./internal/httpapi -run '^TestBug002WrappedNotFoundReachesHTTP$' -count=1
=== RUN   TestBug002WrappedNotFoundReachesHTTP
    bug002_wrapped_not_found_test.go:16: HTTP status = 500, want 404 for wrapped not-found error
--- FAIL: TestBug002WrappedNotFoundReachesHTTP (0.00s)
FAIL
FAIL	pulsegrid/internal/httpapi	1.803s
FAIL
```
