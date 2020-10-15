package observable

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestObservable(t *testing.T) {
	require := require.New(t)
	expected := "Hello, world!"
	observable := New()

	ob := observable.Observe()
	ch := make(chan interface{}, 1)
	done := make(chan struct{}, 1)
	ob.HandleFunc(func(ob *Observer, v interface{}) {
		ch <- v
	}, func(*Observer) {
		close(done)
	})

	observable.Dispatch(expected)
	assertNext(require, ch, expected)

	ob.Dispose()
	observable.Dispatch(expected)
	assertDone(require, ch, done)
}

func assertNext(require *require.Assertions, ch <-chan interface{}, expected string) {
	timeout := time.After(100 * time.Millisecond)
	select {
	case v := <-ch:
		require.Equal(expected, v)
	case <-timeout:
		require.Fail("Timeout while receiving next value")
	}
}

func assertDone(require *require.Assertions, ch <-chan interface{}, done <-chan struct{}) {
	timeout := time.After(100 * time.Millisecond)
	select {
	case <-done:
	case <-ch:
		require.Fail("Expected not to receive any new value")
	case <-timeout:
		require.Fail("Timeout while disposing")
	}
}
