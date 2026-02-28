//go:build linux

package output

import "syscall"

const (
	ioctlReadTermios  uintptr = syscall.TCGETS
	ioctlWriteTermios uintptr = syscall.TCSETS
)
