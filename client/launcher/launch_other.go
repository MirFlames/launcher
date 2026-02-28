//go:build !windows

package main

import "syscall"

var sysProcAttrForLaunch *syscall.SysProcAttr = nil
