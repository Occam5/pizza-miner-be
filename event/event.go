package event

import "sync"

// PoolEventType 奖池事件类型
type PoolEventType string

const (
	// PoolBecameActive 奖池变为活跃状态
	PoolBecameActive PoolEventType = "pool_became_active"
)

// PoolEvent 奖池事件
type PoolEvent struct {
	Type   PoolEventType
	PoolID uint
}

// PoolEventHandler 奖池事件处理函数类型
type PoolEventHandler func(event PoolEvent)

// EventManager 事件管理器
type EventManager struct {
	handlers map[PoolEventType][]PoolEventHandler
	mu       sync.RWMutex
}

var (
	defaultManager = &EventManager{
		handlers: make(map[PoolEventType][]PoolEventHandler),
	}
)

// Subscribe 订阅事件
func Subscribe(eventType PoolEventType, handler PoolEventHandler) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()

	defaultManager.handlers[eventType] = append(defaultManager.handlers[eventType], handler)
}

// Publish 发布事件
func Publish(event PoolEvent) {
	defaultManager.mu.RLock()
	handlers := defaultManager.handlers[event.Type]
	defaultManager.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}
