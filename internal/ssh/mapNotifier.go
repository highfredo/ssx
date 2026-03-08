package ssh

import "sync"

type MapNotifier struct {
	mu       sync.RWMutex
	active   map[string]int
	onChange func()
}

func NewMapNotifier() *MapNotifier {
	return &MapNotifier{
		active: make(map[string]int),
	}
}

func (m *MapNotifier) Set(key string, value int) {
	m.mu.Lock()
	prev, ok := m.active[key]
	if ok && prev == value {
		m.mu.Unlock()
		return
	}
	m.active[key] = value
	m.mu.Unlock()
	m.notify()
}

func (m *MapNotifier) Get(key string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.active[key]
	return value, ok
}

func (m *MapNotifier) Unset(key string) {
	m.mu.Lock()
	_, ok := m.active[key]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(m.active, key)
	m.mu.Unlock()
	m.notify()
}

func (m *MapNotifier) Snapshot() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]int, len(m.active))
	for key, value := range m.active {
		out[key] = value
	}
	return out
}

func (m *MapNotifier) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.active)
}

func (m *MapNotifier) Replace(values map[string]int) {
	next := make(map[string]int, len(values))
	for key, value := range values {
		next[key] = value
	}

	m.mu.Lock()
	m.active = next
	m.mu.Unlock()
	m.notify()
}

func (m *MapNotifier) notify() {
	if m.onChange == nil {
		return
	}
	m.onChange()
}
