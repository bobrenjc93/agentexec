//go:build windows

package client

import "syscall"

func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000008}
}
