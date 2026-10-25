package glass

import "testing"

func TestApplyNeedsAWindow(t *testing.T) {
	if err := Apply(0, true); err == nil {
		t.Fatal("Apply(0) succeeded")
	}
}

func TestRegistryReadsDoNotPanic(t *testing.T) {
	// The values depend on the machine; both calls must just answer.
	_ = Available()
	_ = SystemDark()
}
