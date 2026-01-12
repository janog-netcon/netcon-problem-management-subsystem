package ssh

import (
	"context"
	"testing"
	"time"
)

func TestSetValueConcurrency(t *testing.T) {
	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := newContext(baseCtx, nil)
	defer cancel()

	go func() {
		for { // use a loop to access context.Context functions to make sure they are thread-safe with SetValue
			_, _ = ctx.Deadline()
			_ = ctx.Err()
			_ = ctx.Value("foo")
			select {
			case <-ctx.Done():
				break
			default:
			}
		}
	}()
	ctx.SetValue("bar", -1) // a context value which never changes
	now := time.Now()
	var cnt int64
	go func() {
		for time.Since(now) < 100*time.Millisecond {
			cnt++
			ctx.SetValue("foo", cnt) // a context value which changes a lot
		}
		cancel()
	}()
	<-ctx.Done()
	if ctx.Value("foo") != cnt {
		t.Fatal("context.Value(foo) doesn't match latest SetValue")
	}
	if ctx.Value("bar") != -1 {
		t.Fatal("context.Value(bar) doesn't match latest SetValue")
	}
}
