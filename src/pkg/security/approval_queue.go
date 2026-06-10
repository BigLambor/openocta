// Package security provides shared approval queue management.
package security

import (
	"fmt"
	"sync"
)

// manager 是 ApprovalQueue 的单例管理器
type manager struct {
	mu     sync.RWMutex
	queues map[string]*ApprovalQueue // key: storePath
}

var (
	instance *manager
	once     sync.Once
)

// getManager 返回单例管理器
func getManager() *manager {
	once.Do(func() {
		instance = &manager{
			queues: make(map[string]*ApprovalQueue),
		}
	})
	return instance
}

// GetApprovalQueue 获取或创建指定存储路径的 ApprovalQueue 实例
// 同一个 storePath 会返回同一个实例，确保共享状态
func GetApprovalQueue(storePath string) (*ApprovalQueue, error) {
	if storePath == "" {
		return nil, fmt.Errorf("security: store path required")
	}
	key := approvalStoreKey(storePath)

	m := getManager()

	// 先尝试读锁获取已存在的实例
	m.mu.RLock()
	if q, ok := m.queues[key]; ok {
		m.mu.RUnlock()
		return q, nil
	}
	m.mu.RUnlock()

	// 不存在，创建新实例
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查，防止并发创建
	if q, ok := m.queues[key]; ok {
		return q, nil
	}

	q, err := NewApprovalQueue(storePath)
	if err != nil {
		return nil, err
	}

	m.queues[key] = q
	return q, nil
}

// RemoveApprovalQueue 从管理器中移除指定存储路径的 ApprovalQueue
// 通常在不再需要时调用，比如配置变更时
func RemoveApprovalQueue(storePath string) {
	m := getManager()
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.queues, storePath)
}

// ClearApprovalQueues 清空所有 ApprovalQueue 实例
// 谨慎使用，主要用于测试
func ClearApprovalQueues() {
	m := getManager()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queues = make(map[string]*ApprovalQueue)
}
