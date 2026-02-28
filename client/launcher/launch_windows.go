//go:build windows

package main

import "syscall"

// CREATE_NO_WINDOW = 0x08000000 — не показывать консоль при запуске дочернего процесса
var sysProcAttrForLaunch = &syscall.SysProcAttr{
	CreationFlags: 0x08000000,
}
