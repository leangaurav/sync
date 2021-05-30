package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func returnTrue() bool  { return true }
func returnFalse() bool { return false }
func doPanic() bool     { panic(1); return false }
func returnTrueWithDelay(t time.Duration) func() bool {
	return func() bool { time.Sleep(t); return true }
}
func returnFalseWithDelay(t time.Duration) func() bool {
	return func() bool { time.Sleep(t); return false }
}

func TestDefaults(t *testing.T) {
	o, err := NewOnce(false, false, VerifyNone, returnTrue)
	assert.Equal(t, err, nil)
	assert.Equal(t, o.Done(false), false)
	assert.Equal(t, o.Do(), true)
	assert.Equal(t, o.Done(false), true)
	assert.Equal(t, o.Do(), false)
}

func TestOnceValidArgumentCombinations(t *testing.T) {
	var err error
	_, err = NewOnce(false, false, VerifyNone, returnTrue)
	assert.Equal(t, err, nil)

	_, err = NewOnce(false, false, VerifyAll, returnTrue)
	assert.NotEqual(t, err, nil)

	_, err = NewOnce(false, false, VerifyFirstExit, returnTrue)
	assert.NotEqual(t, err, nil)

	_, err = NewOnce(false, false, VerifyFirstRunAll, returnTrue)
	assert.NotEqual(t, err, nil)

	_, err = NewOnce(true, false, VerifyNone, returnTrue)
	assert.Equal(t, err, nil)

	_, err = NewOnce(true, false, VerifyAll, returnTrue)
	assert.Equal(t, err, nil)

	_, err = NewOnce(true, false, VerifyFirstExit, returnTrue)
	assert.Equal(t, err, nil)

	_, err = NewOnce(true, false, VerifyFirstRunAll, returnTrue)
	assert.Equal(t, err, nil)
}

func TestLazyDone(t *testing.T) {
	var (
		err error
		o   *Once
	)

	o, err = NewOnce(false, false, VerifyNone, returnTrueWithDelay(time.Millisecond*4))
	assert.Equal(t, err, nil)
	go func() { assert.Equal(t, true, o.Do()) }()
	time.Sleep(time.Millisecond)
	assert.Equal(t, o.Done(false), true)

	o, err = NewOnce(true, false, VerifyNone, returnTrueWithDelay(time.Millisecond*1))
	assert.Equal(t, err, nil)
	go func() { assert.Equal(t, true, o.Do()) }()
	assert.Equal(t, false, o.Done(false))
	time.Sleep(time.Millisecond)
	assert.Equal(t, o.Done(false), true)
}

func TestPanicOption(t *testing.T) {
	var (
		err error
		o   *Once
	)

	o, err = NewOnce(false, false, VerifyNone, doPanic)
	assert.Equal(t, err, nil)
	assert.Panics(t, func() { o.Do() })

	o, err = NewOnce(false, true, VerifyNone, doPanic)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, true, o.Do()) })
}

func TestDoneAfterPanic(t *testing.T) {
	var (
		err error
		o   *Once
	)

	// eager done with panic
	o, err = NewOnce(false, false, VerifyNone, doPanic)
	assert.Equal(t, err, nil)
	assert.Panics(t, func() { o.Do() })
	assert.Equal(t, o.Done(false), true)

	// eager done with suppressed panic
	o, err = NewOnce(false, true, VerifyNone, doPanic)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, true, o.Do()) })
	assert.Equal(t, o.Done(false), true)

	// lazy done with panic
	o, err = NewOnce(true, false, VerifyNone, doPanic)
	assert.Equal(t, err, nil)
	assert.Panics(t, func() { o.Do() })
	assert.Equal(t, o.Done(false), false)

	// lazy done with suppressed panic
	o, err = NewOnce(true, true, VerifyNone, doPanic)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, true, o.Do()) })
	assert.Equal(t, o.Done(false), false)
}

func TestVerifyNone(t *testing.T) {
	var (
		err error
		o   *Once
	)

	executed := false
	f := func() bool { executed = true; return false }
	o, err = NewOnce(true, false, VerifyNone, returnFalse, f)
	assert.Equal(t, err, nil)
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))
	assert.Equal(t, true, executed)
}

func TestLazyPanicVerifyNone(t *testing.T) {
	var (
		err error
		o   *Once
	)

	executed := false
	f := func() bool { executed = true; return false }
	o, err = NewOnce(false, false, VerifyNone, returnFalse, doPanic, f)
	assert.Equal(t, err, nil)
	assert.Panics(t, func() { o.Do() })
	assert.Equal(t, true, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, true, VerifyNone, doPanic)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, true, o.Do()) })
	assert.Equal(t, o.Done(false), false)
	assert.Equal(t, false, executed)
}

func TestVerifyAll(t *testing.T) {
	var (
		err error
		o   *Once
	)

	executed := false
	f := func() bool { executed = true; return false }
	o, err = NewOnce(true, false, VerifyAll, returnTrue, f)
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Do())
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, true, executed)

	o, err = NewOnce(true, false, VerifyAll, returnFalse, returnTrue)
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Do())
	assert.Equal(t, false, o.Done(false))

	o, err = NewOnce(true, false, VerifyAll, returnTrue, returnFalse)
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Do())
	assert.Equal(t, false, o.Done(false))

	o, err = NewOnce(true, false, VerifyAll, returnTrue, returnTrue)
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))
}

func TestPanicVerifyAll(t *testing.T) {
	var (
		err error
		o   *Once
	)

	executed := false
	f := func() bool { executed = true; return false }
	o, err = NewOnce(true, false, VerifyAll, doPanic, f)
	assert.Equal(t, err, nil)
	assert.Panics(t, func() { o.Do() })
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, true, VerifyAll, doPanic, f)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, false, o.Do()) })
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, executed)
}

func TestVerifyFirstRunAll(t *testing.T) {
	var (
		err error
		o   *Once
	)

	executed := false
	f := func() bool { executed = true; return false }
	o, err = NewOnce(true, false, VerifyFirstRunAll, returnFalse, f)
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, o.Do())
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, true, executed)

	o, err = NewOnce(true, false, VerifyFirstRunAll, returnFalse, returnFalse)
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Do())
	assert.Equal(t, false, o.Done(false))

	o, err = NewOnce(true, false, VerifyFirstRunAll, returnTrue, returnFalse)
	assert.Equal(t, err, nil)
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))

	o, err = NewOnce(true, false, VerifyFirstRunAll, returnFalse, returnTrue)
	assert.Equal(t, err, nil)
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))

	o, err = NewOnce(true, false, VerifyFirstRunAll, returnTrue, returnTrue)
	assert.Equal(t, err, nil)
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))
}

func TestPanicVerifyFirstRunAll(t *testing.T) {
	var (
		err error
		o   *Once
	)

	executed := false
	f := func() bool { executed = true; return false }
	o, err = NewOnce(true, false, VerifyFirstRunAll, doPanic, f)
	assert.Equal(t, err, nil)
	assert.Panics(t, func() { o.Do() })
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, true, VerifyFirstRunAll, doPanic, f)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, false, o.Do()) })
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, false, VerifyFirstRunAll, returnTrue, doPanic, f)
	assert.Equal(t, err, nil)
	assert.Panics(t, func() { o.Do() })
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, true, VerifyFirstRunAll, returnTrue, doPanic, f)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, true, o.Do()) })
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, executed)
}

func TestVerifyFirstExit(t *testing.T) {
	var (
		err error
		o   *Once
	)

	executed := false
	f := func() bool { executed = true; return false }

	o, err = NewOnce(true, false, VerifyFirstExit, returnTrue, f)
	assert.Equal(t, err, nil)
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, false, VerifyFirstExit, returnFalse, f, returnTrue)
	assert.Equal(t, err, nil)
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))
	assert.Equal(t, true, executed)

	executed = false
	o, err = NewOnce(true, false, VerifyFirstExit, returnFalse, f)
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Do())
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, true, executed)
}

func TestPanicVerifyFirstExit(t *testing.T) {
	var (
		err error
		o   *Once
	)

	executed := false
	f := func() bool { executed = true; return false }
	o, err = NewOnce(true, false, VerifyFirstExit, doPanic, f)
	assert.Equal(t, err, nil)
	assert.Panics(t, func() { o.Do() })
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, true, VerifyFirstExit, doPanic, f)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, false, o.Do()) })
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, false, VerifyFirstExit, returnTrue, doPanic, f)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, true, o.Do()) })
	assert.Equal(t, true, o.Done(false))
	assert.Equal(t, false, executed)

	executed = false
	o, err = NewOnce(true, true, VerifyFirstExit, returnTrue, doPanic, f)
	assert.Equal(t, err, nil)
	assert.NotPanics(t, func() { assert.Equal(t, true, o.Do()) })
	assert.Equal(t, true, o.Done(false))
	assert.Equal(t, false, executed)
}

func TestBlockingGoroutines(t *testing.T) {
	var (
		err error
		o   *Once
	)

	ts := time.Now()
	o, err = NewOnce(true, false, VerifyNone, returnTrueWithDelay(time.Millisecond*10))
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Done(false))
	go func() { assert.Equal(t, true, o.Do()) }()
	time.Sleep(time.Millisecond)
	assert.Equal(t, false, o.Done(false))
	go func() { assert.Equal(t, false, o.Do()) }()
	assert.True(t, time.Now().Sub(ts) < time.Millisecond*5)
	assert.Equal(t, false, o.Do())
	assert.True(t, time.Now().Sub(ts) > time.Millisecond*10)
	assert.Equal(t, true, o.Done(false))
}

func TestReset(t *testing.T) {
	var (
		err error
		o   *Once
	)

	o, err = NewOnce(true, false, VerifyNone, returnTrue)
	assert.Equal(t, err, nil)
	assert.Equal(t, false, o.Reset())
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))
	assert.Equal(t, false, o.Do())

	assert.Equal(t, true, o.Reset())
	assert.Equal(t, false, o.Reset())
	assert.Equal(t, false, o.Done(false))
	assert.Equal(t, true, o.Do())
	assert.Equal(t, true, o.Done(false))
}

func TestResetConcurrentBlocks(t *testing.T) {
	var (
		err error
		o   *Once
	)

	ts := time.Now()
	o, err = NewOnce(true, false, VerifyNone, returnTrueWithDelay(time.Millisecond*4))
	assert.Equal(t, err, nil)
	go func() { assert.Equal(t, true, o.Do()) }()
	assert.True(t, time.Now().Sub(ts) < time.Millisecond)
	time.Sleep(time.Millisecond) // let Do() lock the mutex
	go func() { assert.Equal(t, true, o.Reset()) }()
	assert.True(t, time.Now().Sub(ts) < time.Millisecond*2)
	time.Sleep(time.Millisecond * 4)
	assert.Equal(t, false, o.Done(false))
}
