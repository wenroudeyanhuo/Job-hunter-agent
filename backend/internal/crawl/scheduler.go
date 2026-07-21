package crawl

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
)

var DefaultScheduleSpecs = []string{"0 9 * * *", "0 12 * * *", "0 18 * * *"}
var DefaultAutomationSpecs = []string{"* * * * *"}

func StartScheduler(ctx context.Context, runner Runnable, specs []string) (func(), error) {
	return StartScheduledFunc(ctx, specs, func(ctx context.Context) {
		if runner == nil {
			return
		}
		_, _ = runner.Run(ctx, "scheduled")
	})
}

func StartScheduledFunc(ctx context.Context, specs []string, fn func(context.Context)) (func(), error) {
	c := cron.New()
	for _, spec := range specs {
		spec := spec
		if _, err := c.AddFunc(spec, func() {
			if fn == nil {
				return
			}
			fn(ctx)
		}); err != nil {
			return nil, fmt.Errorf("add cron spec %q: %w", spec, err)
		}
	}
	c.Start()
	stop := func() {
		stopCtx := c.Stop()
		<-stopCtx.Done()
	}
	go func() {
		<-ctx.Done()
		stop()
	}()
	return stop, nil
}
