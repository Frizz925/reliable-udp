package observable

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestObservable(t *testing.T) {
	require := require.New(t)
	observable := New()
	expected := "Hello, world!"

	obs := []*Observer{
		observable.Observe(),
		observable.Observe(),
	}

	go observable.Dispatch(expected)
	for _, ob := range obs {
		assertNext(require, ob.Next(), expected)
		assertTimeout(require, ob.Next())
	}

	ob1, ob2 := obs[0], obs[1]
	ob2.Close()

	go observable.Dispatch(expected)
	assertNext(require, ob1.Next(), expected)
	assertDone(require, ob2.Next(), ob2.Done())

	ch := make(chan interface{})
	dispose := observable.ObserveFunc(func(ob *Observer, v interface{}) {
		ch <- v
	})

	go observable.Dispatch(expected)
	assertNext(require, ch, expected)

	dispose()
	go observable.Dispatch(expected)
	assertTimeout(require, ch)
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

func assertTimeout(require *require.Assertions, ch <-chan interface{}) {
	timeout := time.After(100 * time.Millisecond)
	select {
	case v := <-ch:
		require.Nil(v, "Expected not to receive the next value")
	case <-timeout:
	}
}

func assertDone(require *require.Assertions, ch <-chan interface{}, done <-chan struct{}) {
	timeout := time.After(100 * time.Millisecond)
	select {
	case v := <-ch:
		require.Nil(v, "Expected not to receive the next value")
	case <-timeout:
		require.Fail("Timeout while receiving next value")
	case <-done:
	}
}
