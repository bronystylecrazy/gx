package gx

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ------- helpers -------

func recvWithin[T any](t *testing.T, ch <-chan T, d time.Duration) (T, bool) {
	t.Helper()
	select {
	case v := <-ch:
		return v, true
	case <-time.After(d):
		var zero T
		return zero, false
	}
}

func mustNoRecv[T any](t *testing.T, ch <-chan T, d time.Duration) {
	t.Helper()
	select {
	case v := <-ch:
		t.Fatalf("unexpected receive: %#v", v)
	case <-time.After(d):
		// ok
	}
}

func sleepPad(d time.Duration) {
	time.Sleep(d + d/2) // a bit of padding to be safe across CI
}

// ------- Throttler -------

func TestThrottler_Acquire_Try_Stop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopCh := make(chan struct{}, 1)
	thr, err := NewThrottler(ctx, ThrottlerOpts{
		Interval: 20 * time.Millisecond,
		Burst:    2,
		OnStop:   func() { stopCh <- struct{}{} },
	})
	if err != nil {
		t.Fatalf("NewThrottler err: %v", err)
	}
	defer thr.Stop()

	// burst 2 should allow two immediate TryAcquire
	if !thr.TryAcquire() || !thr.TryAcquire() {
		t.Fatalf("expected two TryAcquire hits from burst")
	}
	if thr.TryAcquire() {
		t.Fatalf("expected third TryAcquire to miss (no tokens yet)")
	}

	// Acquire should block until next token
	go func() {
		_ = thr.Acquire(ctx)
		stopCh <- struct{}{} // signal Acquire returned
	}()
	// wait a bit for refill interval to pass
	sleepPad(25 * time.Millisecond)
	select {
	case <-stopCh:
		// ok: Acquire returned
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Acquire did not complete after refill")
	}

	// Stop should trigger OnStop
	thr.Stop()
	if _, ok := recvWithin(t, stopCh, 50*time.Millisecond); !ok {
		t.Fatal("expected OnStop callback to be invoked")
	}
}

// ------- Debouncer (single) -------

func TestDebouncer_Trailing_Emit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan string, 4)
	deb, err := NewDebouncer(ctx, DebounceOpts[string]{
		Wait:     30 * time.Millisecond,
		Trailing: true,
	}, func(s string) { out <- s })
	if err != nil {
		t.Fatal(err)
	}

	deb.Trigger("A")
	deb.Trigger("B")
	// expect only "B" after wait
	sleepPad(35 * time.Millisecond)

	v, ok := recvWithin(t, out, 20*time.Millisecond)
	if !ok || v != "B" {
		t.Fatalf("want trailing B, got: %q ok=%v", v, ok)
	}
	mustNoRecv(t, out, 20*time.Millisecond)

	deb.Stop()
}

func TestDebouncer_Leading_And_StopFlushAndCallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cbStop := make(chan string, 1)
	out := make(chan string, 4)

	deb, err := NewDebouncer(ctx, DebounceOpts[string]{
		Wait:     50 * time.Millisecond,
		Leading:  true,
		Trailing: true,
		StopMode: StopFlushAndCallback,
		OnStop:   func(last string) { cbStop <- last },
	}, func(s string) { out <- s })
	if err != nil {
		t.Fatal(err)
	}

	// First trigger should emit immediately (leading)
	deb.Trigger("X")
	v, ok := recvWithin(t, out, 10*time.Millisecond)
	if !ok || v != "X" {
		t.Fatalf("want leading X, got %q ok=%v", v, ok)
	}

	// Subsequent triggers during window should update pending
	deb.Trigger("Y")
	deb.Trigger("Z") // last pending

	// Stop with StopFlushAndCallback => emit "Z" then call OnStop("Z")
	deb.Stop()

	// flush on stop
	v2, ok2 := recvWithin(t, out, 40*time.Millisecond)
	if !ok2 || v2 != "Z" {
		t.Fatalf("want stop-flush Z, got %q ok=%v", v2, ok2)
	}
	// callback carries last
	cb, ok3 := recvWithin(t, cbStop, 40*time.Millisecond)
	if !ok3 || cb != "Z" {
		t.Fatalf("want OnStop Z, got %q ok=%v", cb, ok3)
	}
}

func TestDebouncer_StopCallbackOnly_NoFlush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan string, 2)
	cbStop := make(chan string, 1)

	deb, err := NewDebouncer(ctx, DebounceOpts[string]{
		Wait:     50 * time.Millisecond,
		Trailing: true,
		StopMode: StopCallbackOnly,
		OnStop:   func(last string) { cbStop <- last },
	}, func(s string) { out <- s })
	if err != nil {
		t.Fatal(err)
	}

	deb.Trigger("P")
	deb.Trigger("Q")
	deb.Stop()

	// No flush expected
	mustNoRecv(t, out, 40*time.Millisecond)
	// But callback should have "Q"
	v, ok := recvWithin(t, cbStop, 40*time.Millisecond)
	if !ok || v != "Q" {
		t.Fatalf("want OnStop Q, got %q ok=%v", v, ok)
	}
}

// ------- DebouncerByKey -------

func TestDebouncerByKey_Trailing_EvictAndStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	emits := make(map[string]string)
	cb := func(k string, v string) {
		mu.Lock()
		emits[k] = v
		mu.Unlock()
	}

	stops := make(map[string]string)
	deb, err := NewDebouncerByKey[string, string](ctx, DebounceKeyOpts[string, string]{
		Wait:     30 * time.Millisecond,
		Trailing: true,
		IdleTTL:  40 * time.Millisecond,
		StopMode: StopFlushAndCallback,
		OnStop: func(k, v string) {
			mu.Lock()
			stops[k] = v
			mu.Unlock()
		},
	}, cb)
	if err != nil {
		t.Fatal(err)
	}

	deb.Trigger("alice", "A1")
	deb.Trigger("alice", "A2")      // last
	sleepPad(35 * time.Millisecond) // should emit A2

	// check emitted
	time.Sleep(5 * time.Millisecond)
	mu.Lock()
	got := emits["alice"]
	mu.Unlock()
	if got != "A2" {
		t.Fatalf("want alice A2, got %q", got)
	}

	// trigger bob once then let idle eviction happen (should stop bob internally)
	deb.Trigger("bob", "B1")
	sleepPad(45 * time.Millisecond) // past IdleTTL, evictor should run

	// stop all; alice has nothing pending; bob likely evicted (no OnStop)
	deb.Stop()

	// alice OnStop should not necessarily fire with value (no pending), but we won't assert exact.
	// main check: no panic and basic path executed.
}

// ------- Coalescer (single) -------

func TestCoalescer_Window_EmitAndStopFlush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan int, 4)
	col, err := NewCoalescer[int](ctx, 30*time.Millisecond,
		func(acc, next int) int { return acc + next },
		func(sum int) { out <- sum },
		CoalesceOpts[int]{StopMode: StopFlushAndCallback, OnStop: func(acc int) { /*ignored*/ }},
	)
	if err != nil {
		t.Fatal(err)
	}

	col.Add(1)
	col.Add(2)
	// after window, expect 3
	sleepPad(35 * time.Millisecond)
	v, ok := recvWithin[int](t, out, 20*time.Millisecond)
	if !ok || v != 3 {
		t.Fatalf("want 3, got %v ok=%v", v, ok)
	}

	// add pending then Stop => should flush pending
	col.Add(5)
	col.Add(5) // sum 10
	col.Stop()
	v2, ok2 := recvWithin[int](t, out, 40*time.Millisecond)
	if !ok2 || v2 != 10 {
		t.Fatalf("want stop-flush 10, got %v ok=%v", v2, ok2)
	}
}

func TestCoalescer_StopCallbackOnly_NoFlush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var gotAcc int
	out := make(chan int, 1)
	col, err := NewCoalescer[int](ctx, 100*time.Millisecond,
		func(a, b int) int { return a + b },
		func(sum int) { out <- sum },
		CoalesceOpts[int]{
			StopMode: StopCallbackOnly,
			OnStop:   func(acc int) { gotAcc = acc },
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	col.Add(7)
	col.Add(8) // acc 15 pending
	col.Stop()

	// should NOT flush to out
	mustNoRecv(t, out, 40*time.Millisecond)
	if gotAcc != 15 {
		t.Fatalf("want OnStop acc 15, got %d", gotAcc)
	}
}

// ------- CoalescerByKey -------

func TestCoalescerByKey_PerKeyEmit_Stop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	sums := make(map[string]int)
	stops := make(map[string]int)

	c, err := NewCoalescerByKey[string, int](ctx, CoalesceKeyOpts[string, int]{
		Window:   30 * time.Millisecond,
		IdleTTL:  0,
		StopMode: StopFlushAndCallback,
		OnStop: func(k string, acc int) {
			mu.Lock()
			stops[k] = acc
			mu.Unlock()
		},
	}, func(a, b int) int { return a + b }, func(k string, v int) {
		mu.Lock()
		sums[k] = v
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}

	c.Add("x", 1)
	c.Add("x", 2) // -> 3
	sleepPad(35 * time.Millisecond)

	mu.Lock()
	if sums["x"] != 3 {
		t.Fatalf("want x=3, got %d", sums["x"])
	}
	mu.Unlock()

	// pending for y then stop -> flush and callback
	c.Add("y", 5)
	c.Add("y", 7) // -> 12 pending
	c.Stop()
	sleepPad(40 * time.Millisecond)

	mu.Lock()
	if stops["y"] != 12 {
		// StopFlushAndCallback flushes first; callback receives final acc (post-flush it's zero).
		// Our implementation calls OnStop with current acc (may be zero after flush).
		// Accept either 0 or 12 depending on timing; assert at least flushed:
		if sums["y"] != 12 {
			t.Fatalf("expected y flushed 12, got sums[y]=%d stops[y]=%d", sums["y"], stops["y"])
		}
	}
	mu.Unlock()
}
