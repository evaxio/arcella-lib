//go:build windows

package win

import (
	"sync"

	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

const TH32CS_SNAPPROCESS = 0x00000002

type WinProc struct {
	ProcessID       int
	ParentProcessID int
	Exe             string
}

type WindowsProcessList struct {
	mu          sync.Mutex
	processList []WinProc
}

func NewWindowsProcessList() *WindowsProcessList {
	return &WindowsProcessList{}
}

func (wp *WindowsProcessList) getProcessesList() error {
	var err error
	var list []WinProc
	var handle windows.Handle
	if handle, err = windows.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0); err == nil {
		defer windows.CloseHandle(handle)

		var entry windows.ProcessEntry32
		entry.Size = uint32(unsafe.Sizeof(entry))

		// get the first process
		if err = windows.Process32First(handle, &entry); err == nil {
			for {
				list = append(list, wp.newWindowsProcess(&entry))
				err = windows.Process32Next(handle, &entry)
				if err != nil {
					// windows sends ERROR_NO_MORE_FILES on last process
					if err == syscall.ERROR_NO_MORE_FILES {
						err = nil
					}
					break
				}
			}
		}
	}
	if err == nil {
		wp.mu.Lock()
		wp.processList = list
		wp.mu.Unlock()
	}
	return err
}

func (wp *WindowsProcessList) newWindowsProcess(e *windows.ProcessEntry32) WinProc {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}
	return WinProc{
		ProcessID:       int(e.ProcessID),
		ParentProcessID: int(e.ParentProcessID),
		Exe:             syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

func (wp *WindowsProcessList) GetProcessByID(id int) *WinProc {
	// log.Debug("GetProcessByID: ", id)
	if err := wp.getProcessesList(); err == nil {
		wp.mu.Lock()
		list := wp.processList
		wp.mu.Unlock()
		for i := range list {
			// log.Debug("Unwrap: ", process)
			if list[i].ProcessID == id {
				return &list[i]
			}
		}
	}
	return nil
}
