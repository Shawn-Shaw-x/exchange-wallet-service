package tasks

import (
	"fmt"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

type Group struct {
	errGroup   errgroup.Group
	HandleCrit func(err error)
}

func (t *Group) Go(fn func() error) {
	t.errGroup.Go(func() error {
		defer func() {
			// 捕获panic
			if err := recover(); err != nil {
				// 打印堆栈
				debug.PrintStack()
				// 调用错误回调
				t.HandleCrit(fmt.Errorf("panic: %v", err))
			}
		}()
		// 如果有错误就抛出没有就抛出nil
		return fn()
	})
}

func (t *Group) Wait() error {
	// 等待所有任务完成并返回所有的error
	return t.errGroup.Wait()
}
