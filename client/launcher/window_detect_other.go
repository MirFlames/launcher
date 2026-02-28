//go:build !windows

package main

import "os/exec"

// WaitForProcessWindow на не-Windows: скрывает окно сразу при запуске процесса,
// т.к. EnumWindows недоступен. Показывает при выходе процесса.
func WaitForProcessWindow(cmd *exec.Cmd, onWindowFound, onProcessExit func(), onWaiting func()) {
	onWindowFound()
	go func() {
		_ = cmd.Wait()
		onProcessExit()
	}()
}
