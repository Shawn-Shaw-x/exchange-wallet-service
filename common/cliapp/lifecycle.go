package cliapp

import (
	"context"
	"errors"
	"fmt"

	"github.com/dapplink-labs/multichain-sync-account/common/opio"
	"github.com/urfave/cli/v2"
)

// Lifecycle 接口定义了应用程序的生命周期管理方法。
// Start: 启动应用程序。
// Stop: 停止应用程序。
// Stopped: 检查应用程序是否已经停止。
type Lifecycle interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Stopped() bool
}

// LifecycleAction 是一个函数类型，用于初始化一个生命周期管理对象。
// 它接收命令行上下文和取消函数，返回一个实现了 Lifecycle 接口的对象。
type LifecycleAction func(ctx *cli.Context, close context.CancelCauseFunc) (Lifecycle, error)

// interruptErr 表示一个中断信号错误，当用户按下 Ctrl+C 或其他中断信号时，将触发该错误。
var interruptErr = errors.New("interrupt signal")

// LifecycleCmd 是一个函数，用于创建一个 cli.ActionFunc。
// 该函数会调用生命周期管理操作，并在中断信号时优雅地关闭应用。
func LifecycleCmd(fn LifecycleAction) cli.ActionFunc {
	// 定义一个函数，用于在接收到中断信号时阻塞当前上下文
	blockOnInterrupt := opio.BlockOnInterruptsContext
	return func(ctx *cli.Context) error {
		// 从命令行上下文中获取主上下文，并创建带取消功能的应用上下文
		hostCtx := ctx.Context
		appCtx, appCancel := context.WithCancelCause(hostCtx)
		ctx.Context = appCtx
		// 启动一个 Goroutine，监听中断信号，触发取消操作
		go func() {
			blockOnInterrupt(appCtx)
			// 当接收到中断信号时，调用取消函数，传递中断错误
			appCancel(interruptErr)
		}()
		// 调用传入的 LifecycleAction 函数，获取生命周期管理对象
		appLifecycle, err := fn(ctx, appCancel)
		if err != nil {
			// 如果初始化失败，返回错误信息
			return errors.Join(
				fmt.Errorf("failed to setup: %w", err),
				context.Cause(appCtx),
			)
		}
		// 调用生命周期管理对象的 Start 方法，启动应用
		if err := appLifecycle.Start(appCtx); err != nil {
			// 如果启动失败，返回错误信息
			return errors.Join(
				fmt.Errorf("failed to start: %w", err),
				context.Cause(appCtx),
			)
		}
		// 等待应用程序上下文完成，也就是等待程序结束或接收到中断信号
		<-appCtx.Done()
		// 当应用上下文完成后，创建停止上下文，用于优雅停止应用程序
		stopCtx, stopCancel := context.WithCancelCause(hostCtx)
		go func() {
			blockOnInterrupt(stopCtx)
			// 在停止过程中，如果接收到中断信号，取消停止操作
			stopCancel(interruptErr)
		}()
		// 调用生命周期管理对象的 Stop 方法，停止应用程序
		stopErr := appLifecycle.Stop(stopCtx)
		stopCancel(nil) // 停止取消操作
		if stopErr != nil {
			// 如果停止过程中发生错误，返回错误信息
			return errors.Join(
				fmt.Errorf("failed to stop: %w", stopErr),
				context.Cause(stopCtx),
			)
		}
		return nil
	}
}
