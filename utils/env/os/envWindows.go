//go:build windows

package os

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func IsInIDE() bool {
	h, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var p windows.ProcessEntry32
	p.Size = uint32(unsafe.Sizeof(p))
	if e = windows.Process32First(h, &p); e == nil {
		for {
			s := windows.UTF16ToString(p.ExeFile[:])
			// println(s)
			if strings.Contains(s, "goland") {
				return true
			}
			if e = windows.Process32Next(h, &p); e != nil {
				break
			}
		}
	}
	return false
}
