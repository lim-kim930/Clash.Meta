package tunnel

import (
	"runtime"
	"testing"
	"time"

	C "github.com/metacubex/mihomo/constant"
)

func TestLockedRouteMatcherAllowsPendingConfigWriter(t *testing.T) {
	oldMode := mode
	mode = Rule
	defer func() { mode = oldMode }()

	configMux.RLock()
	readLocked := true
	writerDone := make(chan struct{})
	go func() {
		configMux.Lock()
		configMux.Unlock()
		close(writerDone)
	}()
	defer func() {
		if readLocked {
			configMux.RUnlock()
		}
		<-writerDone
	}()

	deadline := time.Now().Add(2 * time.Second)
	for configMux.TryRLock() {
		configMux.RUnlock()
		if time.Now().After(deadline) {
			t.Fatal("config writer did not start")
		}
		runtime.Gosched()
	}

	matched := make(chan error, 1)
	go func() {
		_, _, err := resolveMetadataWithMatcher(&C.Metadata{Type: C.INNER}, matchLocked)
		matched <- err
	}()
	select {
	case err := <-matched:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		configMux.RUnlock()
		readLocked = false
		select {
		case <-matched:
		case <-time.After(2 * time.Second):
		}
		t.Fatal("route matcher recursively acquired the config read lock")
	}
}
