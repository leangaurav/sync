package sync

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type one int

func (o *one) Increment() {
	*o++
}

func (o *one) Compare(value int) bool {
	return int(*o) == value
}

func sleeper(d time.Duration, o *one) func() {
	return func() { time.Sleep(d); o.Increment() }
}

func doPanic() { panic(1) }

func TestDefaults(t *testing.T) {

	o := new(one)
	once := NewOnce(false, sleeper(time.Millisecond*5, o))

	assert.Equal(t, once.Done(false), false)
	assert.True(t, once.Do())
	assert.False(t, o.Compare(0))

	assert.Equal(t, once.Done(false), true)
	assert.False(t, once.Do())
	assert.True(t, o.Compare(1))
}

func TestPanicOption(t *testing.T) {
	var (
		once Once
	)

	once = NewOnce(false, doPanic)
	assert.Panics(t, func() { assert.True(t, once.Do()) })

	once = NewOnce(true, doPanic)
	assert.NotPanics(t, func() { assert.True(t, once.Do()) })
}

func TestDoneAfterPanic(t *testing.T) {
	var o Once

	// done with panic
	o = NewOnce(false, doPanic)
	assert.Panics(t, func() { o.Do() })
	time.Sleep(time.Millisecond)
	assert.Equal(t, o.Done(false), true)

	// done with suppressed panic
	o = NewOnce(true, doPanic)
	assert.NotPanics(t, func() { o.Do() })
	time.Sleep(time.Millisecond)
	assert.Equal(t, o.Done(false), true)
}

func TestBlockingGoroutines(t *testing.T) {
	var once Once
	o := new(one)

	ts := time.Now()
	once = NewOnce(false, sleeper(time.Millisecond*20, o))

	assert.Equal(t, false, once.Done(false))
	go func() { assert.Equal(t, true, once.Do()) }()

	time.Sleep(time.Millisecond)

	go func() { assert.Equal(t, false, once.Do()) }()
	assert.Equal(t, false, once.Done(false))
	assert.True(t, time.Since(ts) < time.Millisecond*20)

	// wait for Do to finish
	assert.Equal(t, false, once.Do())
	assert.True(t, time.Since(ts) > time.Millisecond*20)
	assert.Equal(t, true, once.Done(false))
	assert.True(t, o.Compare(1))
}

func TestBlockingDoneOneGoroutine(t *testing.T) {

	ts := time.Now()
	o := new(one)
	once := NewOnce(false, sleeper(time.Millisecond*10, o))
	go func() { assert.Equal(t, true, once.Do()) }()

	// call Done with block=true
	assert.Equal(t, true, once.Done(true))
	assert.True(t, time.Since(ts) > time.Millisecond*10)
	assert.True(t, o.Compare(1))
}

func TestBlockingDoneMultipleGoroutine(t *testing.T) {

	o := new(one)
	once := NewOnce(false, sleeper(time.Millisecond*4, o))
	ts := time.Now()
	go func() { assert.Equal(t, true, once.Do()) }()

	// call Done with block=true
	var t1, t2 time.Time
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { assert.Equal(t, true, once.Done(true)); t1 = time.Now(); wg.Done() }()
	wg.Add(1)
	go func() { assert.Equal(t, true, once.Done(true)); t2 = time.Now(); wg.Done() }()

	// block for done to be set
	once.Done(true)
	wg.Wait()
	assert.True(t, t2.Sub(ts) > time.Millisecond*4)
	assert.True(t, t1.Sub(ts) > time.Millisecond*4)

	assert.True(t, once.Done(false))
	// once state is DONE, Done(true) should return immediately
	ts = time.Now()
	assert.True(t, once.Done(true))
	assert.True(t, time.Since(ts) < time.Millisecond*10)
	assert.True(t, o.Compare(1))
}

func TestBlockingDoneMultipleGoroutineExplicitClose(t *testing.T) {

	var t1, t2 time.Time
	var wg sync.WaitGroup

	o := new(one)
	once := NewOnce(false, sleeper(time.Millisecond*50, o))

	ts := time.Now()
	go func() { assert.Equal(t, true, once.Do()) }()

	// call Done with block=true
	wg.Add(1)
	go func() { assert.Equal(t, false, once.Done(true)); t1 = time.Now(); wg.Done() }()
	wg.Add(1)
	go func() { assert.Equal(t, false, once.Done(true)); t2 = time.Now(); wg.Done() }()

	// close after 5ms
	wg.Add(1)
	go func() { time.Sleep(time.Millisecond * 5); once.Close(); wg.Done() }()

	wg.Wait()

	// both goroutines should get unblocked immediately after Close()
	assert.True(t, time.Millisecond*5 < t2.Sub(ts) && t2.Sub(ts) < time.Millisecond*40)
	assert.True(t, time.Millisecond*5 < t1.Sub(ts) && t1.Sub(ts) < time.Millisecond*40)
	time.Sleep(time.Millisecond * 50)
}
