package gx

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ------------------------------------------------------------
// Stop behavior
// ------------------------------------------------------------

type StopMode int

const (
	StopNoop             StopMode = iota
	StopFlush                     // flush pending (respecting trailing / accumulator)
	StopCallbackOnly              // only call OnStop
	StopFlushAndCallback          // flush then OnStop
)

// ------------------------------------------------------------
// Throttler (token bucket)
// ------------------------------------------------------------

type Throttler interface {
	Acquire(ctx context.Context) error
	TryAcquire() bool
	Stop()
}

type ThrottlerOpts struct {
	Interval time.Duration
	Burst    int
	OnStop   func() // optional
}

func NewThrottler(ctx context.Context, opts ThrottlerOpts) (Throttler, error) {
	if opts.Interval <= 0 {
		return nil, errors.New("gx.Throttler: Interval must be > 0")
	}
	if opts.Burst < 1 {
		opts.Burst = 1
	}
	t := &throttler{
		interval: opts.Interval,
		tokens:   make(chan struct{}, opts.Burst),
		stop:     make(chan struct{}),
		onStop:   opts.OnStop,
	}
	// warm bucket
	for i := 0; i < opts.Burst; i++ {
		t.tokens <- struct{}{}
	}
	go t.refill(ctx)
	return t, nil
}

type throttler struct {
	interval time.Duration
	tokens   chan struct{}
	stop     chan struct{}
	stopOnce sync.Once
	onStop   func()
}

func (t *throttler) refill(ctx context.Context) {
	tk := time.NewTicker(t.interval)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stop:
			return
		case <-tk.C:
			select {
			case t.tokens <- struct{}{}:
			default:
			}
		}
	}
}

func (t *throttler) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.stop:
		return errors.New("gx.Throttler: stopped")
	case <-t.tokens:
		return nil
	}
}

func (t *throttler) TryAcquire() bool {
	select {
	case <-t.tokens:
		return true
	default:
		return false
	}
}

func (t *throttler) Stop() {
	t.stopOnce.Do(func() {
		if t.onStop != nil {
			t.onStop()
		}
		close(t.stop)
	})
}

// ------------------------------------------------------------
// Debouncer (single-key)
// ------------------------------------------------------------

type Debouncer[T any] interface {
	Trigger(v T)
	Flush()
	Stop()
}

type DebounceOpts[T any] struct {
	Wait     time.Duration
	Leading  bool
	Trailing bool
	MaxWait  time.Duration

	StopMode StopMode
	OnStop   func(last T) // optional, gets last pending value (pre-flush)
}

func NewDebouncer[T any](ctx context.Context, opts DebounceOpts[T], cb func(T)) (Debouncer[T], error) {
	if opts.Wait <= 0 {
		return nil, errors.New("gx.Debouncer: Wait must be > 0")
	}
	if !opts.Leading && !opts.Trailing {
		opts.Trailing = true
	}
	d := &debouncer[T]{opts: opts, cb: cb, ctx: ctx}
	return d, nil
}

type debouncer[T any] struct {
	mu       sync.Mutex
	opts     DebounceOpts[T]
	cb       func(T)
	ctx      context.Context
	timer    *time.Timer
	maxTimer *time.Timer
	last     T
	pending  bool
	stopped  bool

	// ensure loops are started exactly once when timers are first created
	timerLoopStarted bool
	maxLoopStarted   bool
}

func (d *debouncer[T]) Trigger(v T) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stopped {
		return
	}
	first := !d.pending
	d.pending = true
	d.last = v

	if d.opts.Leading && first {
		d.cb(v)
		d.resetTimerLocked(d.opts.Wait)
		d.resetMaxLocked(d.opts.MaxWait)
		return
	}
	d.resetTimerLocked(d.opts.Wait)
	if d.opts.MaxWait > 0 && first {
		d.resetMaxLocked(d.opts.MaxWait)
	}
}

func (d *debouncer[T]) Flush() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stopped || !d.pending {
		return
	}
	if d.opts.Trailing {
		d.cb(d.last)
	}
	d.pending = false
	d.stopTimerLocked(d.timer)
	d.stopTimerLocked(d.maxTimer)
}

func (d *debouncer[T]) Stop() {
	// snapshot state BEFORE any flush/callback
	d.mu.Lock()
	if d.stopped {
		d.mu.Unlock()
		return
	}
	pending := d.pending
	last := d.last
	shouldFlush := (d.opts.StopMode == StopFlush || d.opts.StopMode == StopFlushAndCallback) &&
		pending && d.opts.Trailing
	shouldCallback := (d.opts.StopMode == StopCallbackOnly || d.opts.StopMode == StopFlushAndCallback) &&
		d.opts.OnStop != nil

	d.stopped = true
	// stop timers; clear pending inside lock
	d.stopTimerLocked(d.timer)
	d.stopTimerLocked(d.maxTimer)
	d.pending = false
	d.mu.Unlock()

	// flush first (pre-flush value)
	if shouldFlush {
		d.cb(last)
	}
	// then callback with the same pre-flush value
	if shouldCallback {
		d.opts.OnStop(last)
	}
}

func (d *debouncer[T]) timerLoop() {
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.timer.C:
			d.mu.Lock()
			if d.stopped {
				d.mu.Unlock()
				return
			}
			if d.opts.Trailing && d.pending {
				d.cb(d.last)
			}
			d.pending = false
			d.stopTimerLocked(d.timer)
			d.stopTimerLocked(d.maxTimer)
			d.mu.Unlock()
		}
	}
}

func (d *debouncer[T]) maxLoop() {
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.maxTimer.C:
			d.mu.Lock()
			if d.stopped {
				d.mu.Unlock()
				return
			}
			if d.pending && d.opts.Trailing {
				d.cb(d.last)
			}
			d.pending = false
			d.stopTimerLocked(d.timer)
			d.stopTimerLocked(d.maxTimer)
			d.mu.Unlock()
		}
	}
}

func (d *debouncer[T]) resetTimerLocked(dur time.Duration) {
	if dur <= 0 {
		return
	}
	if d.timer == nil {
		d.timer = time.NewTimer(dur)
		if !d.timerLoopStarted {
			d.timerLoopStarted = true
			go d.timerLoop()
		}
		return
	}
	if !d.timer.Stop() {
		select {
		case <-d.timer.C:
		default:
		}
	}
	d.timer.Reset(dur)
}

func (d *debouncer[T]) resetMaxLocked(dur time.Duration) {
	if dur <= 0 {
		return
	}
	if d.maxTimer == nil {
		d.maxTimer = time.NewTimer(dur)
		if !d.maxLoopStarted {
			d.maxLoopStarted = true
			go d.maxLoop()
		}
		return
	}
	if !d.maxTimer.Stop() {
		select {
		case <-d.maxTimer.C:
		default:
		}
	}
	d.maxTimer.Reset(dur)
}

func (d *debouncer[T]) stopTimerLocked(t *time.Timer) {
	if t == nil {
		return
	}
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	// IMPORTANT: do not nil the timer here; loops may still reference it safely after Stop()
}

// ------------------------------------------------------------
// DebouncerByKey
// ------------------------------------------------------------

type DebouncerByKey[K comparable, V any] interface {
	Trigger(k K, v V)
	FlushKey(k K)
	FlushAll()
	Stop()
}

type DebounceKeyOpts[K comparable, V any] struct {
	Wait     time.Duration
	Leading  bool
	Trailing bool
	MaxWait  time.Duration
	IdleTTL  time.Duration // optional eviction of idle keys

	StopMode StopMode
	OnStop   func(key K, last V) // callback gets pre-flush value
}

func NewDebouncerByKey[K comparable, V any](
	ctx context.Context,
	opts DebounceKeyOpts[K, V],
	cb func(K, V),
) (DebouncerByKey[K, V], error) {
	if opts.Wait <= 0 {
		return nil, errors.New("gx.DebouncerByKey: Wait must be > 0")
	}
	if !opts.Leading && !opts.Trailing {
		opts.Trailing = true
	}
	m := &debouncerByKey[K, V]{
		ctx:   ctx,
		opts:  opts,
		cb:    cb,
		nodes: make(map[K]*debouncerNode[V]),
	}
	if opts.IdleTTL > 0 {
		go m.evictor()
	}
	return m, nil
}

type debouncerNode[V any] struct {
	db   *debouncer[V]
	last time.Time
}

type debouncerByKey[K comparable, V any] struct {
	mu    sync.Mutex
	ctx   context.Context
	opts  DebounceKeyOpts[K, V]
	cb    func(K, V)
	nodes map[K]*debouncerNode[V]
	stop  bool
}

func (m *debouncerByKey[K, V]) Trigger(k K, v V) {
	m.mu.Lock()
	if m.stop {
		m.mu.Unlock()
		return
	}
	n := m.nodes[k]
	if n == nil {
		db, _ := NewDebouncer[V](m.ctx, DebounceOpts[V]{
			Wait:     m.opts.Wait,
			Leading:  m.opts.Leading,
			Trailing: m.opts.Trailing,
			MaxWait:  m.opts.MaxWait,
			StopMode: m.opts.StopMode,
			OnStop: func(last V) {
				if m.opts.OnStop != nil {
					m.opts.OnStop(k, last)
				}
			},
		}, func(val V) { m.cb(k, val) })
		n = &debouncerNode[V]{db: db.(*debouncer[V])}
		m.nodes[k] = n
	}
	n.last = time.Now()
	m.mu.Unlock()
	n.db.Trigger(v)
}

func (m *debouncerByKey[K, V]) FlushKey(k K) {
	m.mu.Lock()
	n := m.nodes[k]
	m.mu.Unlock()
	if n != nil {
		n.db.Flush()
	}
}

func (m *debouncerByKey[K, V]) FlushAll() {
	m.mu.Lock()
	list := make([]*debouncer[V], 0, len(m.nodes))
	for _, n := range m.nodes {
		list = append(list, n.db)
	}
	m.mu.Unlock()
	for _, db := range list {
		db.Flush()
	}
}

func (m *debouncerByKey[K, V]) Stop() {
	m.mu.Lock()
	if m.stop {
		m.mu.Unlock()
		return
	}
	m.stop = true
	list := make([]struct {
		k  K
		db *debouncer[V]
	}, 0, len(m.nodes))
	for k, n := range m.nodes {
		list = append(list, struct {
			k  K
			db *debouncer[V]
		}{k, n.db})
	}
	m.mu.Unlock()

	// stop each per-key debouncer (they'll handle StopMode + OnStop)
	for _, it := range list {
		it.db.Stop()
	}
}

func (m *debouncerByKey[K, V]) evictor() {
	t := time.NewTicker(m.opts.IdleTTL)
	defer t.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-t.C:
			m.mu.Lock()
			if m.stop {
				m.mu.Unlock()
				return
			}
			cut := time.Now().Add(-m.opts.IdleTTL)
			for k, n := range m.nodes {
				if n.last.Before(cut) {
					n.db.Stop()
					delete(m.nodes, k)
				}
			}
			m.mu.Unlock()
		}
	}
}

// ------------------------------------------------------------
// Coalescer (single-key)
// ------------------------------------------------------------

type Coalescer[T any] interface {
	Add(v T)
	Flush()
	Stop()
}

type CoalesceOpts[T any] struct {
	StopMode StopMode
	OnStop   func(acc T) // callback gets pre-flush accumulator
}

func NewCoalescer[T any](
	ctx context.Context,
	window time.Duration,
	folder func(acc T, next T) T,
	emit func(T),
	opts ...CoalesceOpts[T],
) (Coalescer[T], error) {
	if window <= 0 {
		return nil, errors.New("gx.Coalescer: Window must be > 0")
	}
	var o CoalesceOpts[T]
	if len(opts) > 0 {
		o = opts[0]
	}
	c := &coalescer[T]{ctx: ctx, window: window, folder: folder, emit: emit, stopMode: o.StopMode, onStop: o.OnStop}
	return c, nil
}

type coalescer[T any] struct {
	mu       sync.Mutex
	ctx      context.Context
	window   time.Duration
	folder   func(acc T, next T) T
	emit     func(T)
	timer    *time.Timer
	acc      T
	hasAcc   bool
	stopped  bool
	stopMode StopMode
	onStop   func(T)
}

func (c *coalescer[T]) Add(v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return
	}
	if !c.hasAcc {
		c.acc = v
		c.hasAcc = true
	} else {
		c.acc = c.folder(c.acc, v)
	}
	if c.timer == nil {
		c.timer = time.NewTimer(c.window)
		go c.loop()
	} else {
		if !c.timer.Stop() {
			select {
			case <-c.timer.C:
			default:
			}
		}
		c.timer.Reset(c.window)
	}
}

func (c *coalescer[T]) Flush() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped || !c.hasAcc {
		return
	}
	acc := c.acc
	c.hasAcc = false
	c.stopTimerLocked()
	c.mu.Unlock()
	c.emit(acc)
	c.mu.Lock()
}

func (c *coalescer[T]) Stop() {
	// snapshot state BEFORE any flush/callback
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	hasAcc := c.hasAcc
	acc := c.acc
	shouldFlush := (c.stopMode == StopFlush || c.stopMode == StopFlushAndCallback) && hasAcc
	shouldCallback := (c.stopMode == StopCallbackOnly || c.stopMode == StopFlushAndCallback) && c.onStop != nil

	c.stopped = true
	c.stopTimerLocked()
	c.hasAcc = false
	c.mu.Unlock()

	// flush first (pre-flush acc)
	if shouldFlush {
		c.emit(acc)
	}
	// then callback with the same pre-flush acc
	if shouldCallback {
		c.onStop(acc)
	}
}

func (c *coalescer[T]) loop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.timer.C:
			c.mu.Lock()
			if c.stopped {
				c.mu.Unlock()
				return
			}
			if c.hasAcc {
				acc := c.acc
				c.hasAcc = false
				c.stopTimerLocked()
				c.mu.Unlock()
				c.emit(acc)
				continue
			}
			c.stopTimerLocked()
			c.mu.Unlock()
			return
		}
	}
}

func (c *coalescer[T]) stopTimerLocked() {
	if c.timer != nil {
		if !c.timer.Stop() {
			select {
			case <-c.timer.C:
			default:
			}
		}
		// IMPORTANT: do not nil the timer here; loop may still reference it safely after Stop()
	}
}

// ------------------------------------------------------------
// CoalescerByKey
// ------------------------------------------------------------

type CoalescerByKey[K comparable, V any] interface {
	Add(k K, v V)
	FlushKey(k K)
	FlushAll()
	Stop()
}

type CoalesceKeyOpts[K comparable, V any] struct {
	Window  time.Duration
	IdleTTL time.Duration // optional eviction

	StopMode StopMode
	OnStop   func(key K, acc V) // callback gets pre-flush accumulator
}

func NewCoalescerByKey[K comparable, V any](
	ctx context.Context,
	opts CoalesceKeyOpts[K, V],
	folder func(acc V, next V) V,
	emit func(K, V),
) (CoalescerByKey[K, V], error) {
	if opts.Window <= 0 {
		return nil, errors.New("gx.CoalescerByKey: Window must be > 0")
	}
	c := &coalescerByKey[K, V]{
		ctx:    ctx,
		opts:   opts,
		folder: folder,
		emit:   emit,
		nodes:  make(map[K]*coalesceNode[V]),
	}
	if opts.IdleTTL > 0 {
		go c.evictor()
	}
	return c, nil
}

type coalesceNode[V any] struct {
	c    *coalescer[V]
	last time.Time
}

type coalescerByKey[K comparable, V any] struct {
	mu     sync.Mutex
	ctx    context.Context
	opts   CoalesceKeyOpts[K, V]
	folder func(acc V, next V) V
	emit   func(K, V)
	nodes  map[K]*coalesceNode[V]
	stop   bool
}

func (m *coalescerByKey[K, V]) Add(k K, v V) {
	m.mu.Lock()
	if m.stop {
		m.mu.Unlock()
		return
	}
	n := m.nodes[k]
	if n == nil {
		cc, _ := NewCoalescer[V](m.ctx, m.opts.Window, m.folder, func(acc V) { m.emit(k, acc) },
			CoalesceOpts[V]{StopMode: m.opts.StopMode, OnStop: func(a V) {
				if m.opts.OnStop != nil {
					m.opts.OnStop(k, a)
				}
			}},
		)
		n = &coalesceNode[V]{c: cc.(*coalescer[V])}
		m.nodes[k] = n
	}
	n.last = time.Now()
	m.mu.Unlock()
	n.c.Add(v)
}

func (m *coalescerByKey[K, V]) FlushKey(k K) {
	m.mu.Lock()
	n := m.nodes[k]
	m.mu.Unlock()
	if n != nil {
		n.c.Flush()
	}
}

func (m *coalescerByKey[K, V]) FlushAll() {
	m.mu.Lock()
	list := make([]*coalescer[V], 0, len(m.nodes))
	for _, n := range m.nodes {
		list = append(list, n.c)
	}
	m.mu.Unlock()
	for _, c := range list {
		c.Flush()
	}
}

func (m *coalescerByKey[K, V]) Stop() {
	m.mu.Lock()
	if m.stop {
		m.mu.Unlock()
		return
	}
	m.stop = true
	list := make([]struct {
		k K
		c *coalescer[V]
	}, 0, len(m.nodes))
	for k, n := range m.nodes {
		list = append(list, struct {
			k K
			c *coalescer[V]
		}{k, n.c})
	}
	m.mu.Unlock()

	// stop each per-key coalescer (honors StopMode + OnStop per key)
	for _, it := range list {
		it.c.Stop()
	}
}

func (m *coalescerByKey[K, V]) evictor() {
	t := time.NewTicker(m.opts.IdleTTL)
	defer t.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-t.C:
			m.mu.Lock()
			if m.stop {
				m.mu.Unlock()
				return
			}
			cut := time.Now().Add(-m.opts.IdleTTL)
			for k, n := range m.nodes {
				if n.last.Before(cut) {
					n.c.Stop()
					delete(m.nodes, k)
				}
			}
			m.mu.Unlock()
		}
	}
}
