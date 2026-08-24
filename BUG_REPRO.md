# Bug 复现说明

## Bug 是什么
服务启动后有一类异步任务偶尔会让进程直接退出，任务统计也来不及记录，正在处理的请求会一起中断；请修复这条任务执行链路，让异常任务被安全处理且不影响正常任务。。

## 如何触发
按题面所述业务流程准备输入，执行目标验证场景即可触发该问题；该题涉及的生产链路为main worker registration → jobs.Queue → worker execution。

## 根因
1、文件：cmd/pulsegridd/main.go、internal/jobs/jobs.go、internal/service/service.go。2、符号：Queue.Register、Queue.execute、App.publish。3、失效机制：启动阶段把 nil handler 注册进真实任务队列，worker 执行时直接调用空函数，触发进程级 panic 并中断统计。

## 修复状态
Claude 主轨迹已完成实际修复，文档记录的是修复前的真实失败证据。

## 运行指令
以下命令来自本题验证契约中的 target 场景：

```bash
go test -v ./internal/jobs -run '^TestBug029NilWorkerDoesNotCrash$' -count=1
```

## 错误信息
panic: runtime error: invalid memory address or nil pointer dereference

## 错误堆栈
下面保留该题修复前稳定性检查首轮 target 的完整原始输出，未对交付文档中的测试路径、测试方法名或堆栈内容做脱敏：

```text
[stability/before_fix] 第 1/5 轮：nil_worker
$ go test -v ./internal/jobs -run '^TestBug029NilWorkerDoesNotCrash$' -count=1
=== RUN   TestBug029NilWorkerDoesNotCrash
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x2 addr=0x0 pc=0x1028d7c24]

goroutine 35 [running]:
pulsegrid/internal/jobs.(*Queue).execute(0x13799eb58080, {0x13799eb04308, 0x8}, {{0x13799eb26018, 0x17}, {0x1028e3d28, 0xa}, 0x0, 0x0, 0x1, ...})
	/private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-st9d7aas/env/internal/jobs/jobs.go:153 +0x184
pulsegrid/internal/jobs.(*Queue).worker(0x13799eb58080, 0x102a5ae18?)
	/private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-st9d7aas/env/internal/jobs/jobs.go:134 +0xd4
created by pulsegrid/internal/jobs.(*Queue).Start in goroutine 34
	/private/var/folders/xt/smfvg4l12sxg2p93jgz9qfh80000gn/T/go-an-stability-st9d7aas/env/internal/jobs/jobs.go:83 +0x94
FAIL	pulsegrid/internal/jobs	1.328s
FAIL
```
