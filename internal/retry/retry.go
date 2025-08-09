package retry

import (
	"context"
	"time"
)

type Fn func(context.Context) error

// Первая попытка сразу + последующие, пока retryIf(err) == true.
func DoIf(ctx context.Context, delays []time.Duration, fn Fn, retryIf func(error) bool) error {
	err := fn(ctx)
	if err == nil {
		return nil
	}
	for _, d := range delays {
		if !retryIf(err) {
			return err
		}
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
		if err = fn(ctx); err == nil {
			return nil
		}
	}
	return err
}
