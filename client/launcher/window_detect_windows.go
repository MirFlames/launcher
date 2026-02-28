//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                      = syscall.NewLazyDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
)

// processHasVisibleWindow проверяет, есть ли у процесса pid видимое окно.
func processHasVisibleWindow(pid int) bool {
	var found bool
	cb := syscall.NewCallback(func(hwnd syscall.Handle, lParam uintptr) uintptr {
		var winPid uint32
		procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&winPid)))
		if uint32(pid) != winPid {
			return 1 // continue
		}
		visible, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
		if visible != 0 {
			found = true
			return 0 // stop
		}
		return 1 // continue
	})
	procEnumWindows.Call(cb, 0)
	return found
}

// WaitForProcessWindow опрашивает появление окна процесса и вызывает onWindowFound при первом видимом окне.
// Параллельно ждёт завершения процесса; при выходе вызывает onProcessExit.
// onWaiting вызывается на каждой итерации опроса (для отображения прогресс-бара).
func WaitForProcessWindow(cmd *exec.Cmd, onWindowFound, onProcessExit func(), onWaiting func()) {
	pid := cmd.Process.Pid
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
		onProcessExit()
	}()
	for {
		select {
		case <-done:
			return
		default:
		}
		if processHasVisibleWindow(pid) {
			onWindowFound()
			<-done
			return
		}
		if onWaiting != nil {
			onWaiting()
		}
		time.Sleep(300 * time.Millisecond)
	}
}
