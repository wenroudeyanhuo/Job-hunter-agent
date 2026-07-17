package crawl

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
)

var DefaultScheduleSpecs = []string{"0 9 * * *", "0 12 * * *", "0 18 * * *"}

func StartScheduler(ctx context.Context, runner *Runner, specs []string) (func(), error) {
	c := cron.New()
	for _, spec := range specs {
		spec := spec
		if _, err := c.AddFunc(spec, func() {
			if runner == nil {
				return
			}
			_, _ = runner.Run(ctx, "scheduled")
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
