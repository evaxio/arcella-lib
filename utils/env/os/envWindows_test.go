//go:build windows

package os

import (
	"testing"
)

func TestIsInIDESmoke(t *testing.T) {
	// light smoke test: must complete without panic or hang
	t.Logf("IsInIDE = %v", IsInIDE())
}
