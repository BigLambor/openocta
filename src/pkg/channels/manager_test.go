package channels

import (
	"context"
	"fmt"
	"testing"
)

type mockChannel struct {
	*BaseRuntimeImpl
	startErr error
}

func (m *mockChannel) Start(ctx context.Context) error {
	// Simulate what real runtimes do: call base Start, then handle specific startup logic
	if err := m.BaseRuntimeImpl.Start(ctx); err != nil {
		return err
	}
	if m.startErr != nil {
		m.BaseRuntimeImpl.MarkConnectionFailed(m.startErr)
		return m.startErr
	}
	return nil
}

func (m *mockChannel) Send(msg *RuntimeOutboundMessage) error {
	return nil
}

func (m *mockChannel) SendStream(chatID string, stream <-chan *RuntimeStreamChunk) error {
	return nil
}

func TestManagerStartErrors(t *testing.T) {
	mgr := NewManager()

	// 1. Register a successful channel
	cfgSuccess := BaseRuntimeConfig{Enabled: true, AccountID: "acc-ok"}
	chSuccess := &mockChannel{
		BaseRuntimeImpl: NewBaseRuntimeImpl("mock-ok", "acc-ok", cfgSuccess, nil),
	}
	if err := mgr.Register(chSuccess); err != nil {
		t.Fatalf("failed to register successful channel: %v", err)
	}

	// 2. Register a failing channel
	cfgFail := BaseRuntimeConfig{Enabled: true, AccountID: "acc-fail"}
	chFail := &mockChannel{
		BaseRuntimeImpl: NewBaseRuntimeImpl("mock-fail", "acc-fail", cfgFail, nil),
		startErr:        fmt.Errorf("api authentication failed"),
	}
	if err := mgr.Register(chFail); err != nil {
		t.Fatalf("failed to register failing channel: %v", err)
	}

	// 3. Start the manager
	results := mgr.Start(context.Background())

	if len(results) != 2 {
		t.Fatalf("expected 2 start results, got %d", len(results))
	}

	// Verify success/fail flags and errors
	var okCount, failCount int
	for _, res := range results {
		if res.Success {
			okCount++
			if res.ChannelID != "mock-ok" {
				t.Errorf("expected mock-ok to succeed, but got channelID %s", res.ChannelID)
			}
		} else {
			failCount++
			if res.ChannelID != "mock-fail" {
				t.Errorf("expected mock-fail to fail, but got channelID %s", res.ChannelID)
			}
			if res.Error != "api authentication failed" {
				t.Errorf("expected error 'api authentication failed', got %q", res.Error)
			}
		}
	}

	if okCount != 1 || failCount != 1 {
		t.Errorf("expected 1 success and 1 failure, got %d success and %d failures", okCount, failCount)
	}

	// 4. Verify that ListRuntimes reflects the failure
	runtimes := mgr.ListRuntimes()
	failStatus, ok := runtimes["mock-fail"]["acc-fail"]
	if !ok {
		t.Fatalf("runtime status for mock-fail/acc-fail not found")
	}

	if failStatus.Running {
		t.Errorf("expected mock-fail to not be running")
	}
	if failStatus.LastError != "api authentication failed" {
		t.Errorf("expected ListRuntimes to expose lastError 'api authentication failed', got %q", failStatus.LastError)
	}
}
