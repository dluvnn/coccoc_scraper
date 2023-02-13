package monitor

import "sync"

type SafeValue[T any] struct {
	mtx   sync.RWMutex
	value T
}

func (sc *SafeValue[T]) Get() T {
	sc.mtx.RLock()
	defer sc.mtx.RUnlock()
	return sc.value
}

func (sc *SafeValue[T]) Set(x T) {
	sc.mtx.Lock()
	defer sc.mtx.Unlock()
	sc.value = x
}

type SafeCounter SafeValue[int64]

func (sc *SafeCounter) Inc() {
	sc.mtx.Lock()
	defer sc.mtx.Unlock()
	sc.value++
}

func (sc *SafeCounter) Get() int64 {
	sc.mtx.RLock()
	defer sc.mtx.RUnlock()
	return sc.value
}

type SafeStringMap[T any] struct {
	mtx  sync.RWMutex
	data map[string]T
}

func (sm *SafeStringMap[T]) Get(key string) (T, bool) {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()
	x, ok := sm.data[key]
	return x, ok
}

func (sm *SafeStringMap[T]) GetMany(keys []string) map[string]T {
	m := map[string]T{}
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()
	for _, k := range keys {
		x, ok := sm.data[k]
		if ok {
			m[k] = x
		}
	}
	return m
}

func (sm *SafeStringMap[T]) Set(key string, value T) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	sm.data[key] = value
}

func (sm *SafeStringMap[T]) SetMany(m map[string]T) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	for k, v := range m {
		sm.data[k] = v
	}
}

func (sm *SafeStringMap[T]) Clear() map[string]T {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	m := sm.data
	sm.data = make(map[string]T)
	return m
}

func (sm *SafeStringMap[T]) All() map[string]T {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()
	m := make(map[string]T)
	for k, v := range sm.data {
		m[k] = v
	}
	return m
}

type CounterManager struct {
	mtx      sync.Mutex
	counters SafeStringMap[*SafeCounter]
	updated  SafeStringMap[bool]
}

func (cm *CounterManager) Init() {
	cm.counters.Clear()
	cm.updated.Clear()
}

func (cm *CounterManager) Get(user_id string) *SafeCounter {
	p, ok := cm.counters.Get(user_id)
	if !ok {
		cm.mtx.Lock()
		defer cm.mtx.Unlock()
		p, ok = cm.counters.Get(user_id)
		if !ok {
			p = new(SafeCounter)
			cm.counters.Set(user_id, p)
		}
	}
	return p
}

func (cm *CounterManager) Update(user_id string) {
	cm.Get(user_id).Inc()
	cm.updated.Set(user_id, true)
}

func (cm *CounterManager) ChangedInfo() map[string]int64 {
	m := cm.updated.Clear()
	rs := make(map[string]int64)
	for user_id := range m {
		p, ok := cm.counters.Get(user_id)
		if !ok {
			continue
		}
		rs[user_id] = p.Get()
	}
	return rs
}
