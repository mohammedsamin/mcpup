//go:build darwin

package output

import "syscall"

const (
	ioctlReadTermios  uintptr = syscall.TIOCGETA
	ioctlWriteTermios uintptr = syscall.TIOCSETA
)
