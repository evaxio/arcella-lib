//go:build windows

package win

import (
	"os"
	"testing"
)

func TestGetProcessByID(t *testing.T) {
	wp := NewWindowsProcessList()
	pid := os.Getpid()
	p := wp.GetProcessByID(pid)
	if p == nil {
		t.Fatalf("current process %d not found", pid)
	}
	if p.ProcessID != pid {
		t.Errorf("ProcessID = %d, want %d", p.ProcessID, pid)
	}
	if wp.GetProcessByID(-1) != nil {
		t.Error("expected nil for bogus pid")
	}
}

func TestProcessListDoesNotGrow(t *testing.T) {
	wp := NewWindowsProcessList()
	pid := os.Getpid()
	for i := 0; i < 5; i++ {
		_ = wp.GetProcessByID(pid)
	}
	repeated := len(wp.processList)
	fresh := NewWindowsProcessList()
	_ = fresh.GetProcessByID(pid)
	single := len(fresh.processList)
	// allow a few processes really starting on the machine in between
	if repeated > single+10 {
		t.Errorf("processList accumulated across calls: repeated=%d, single=%d", repeated, single)
	}
}
