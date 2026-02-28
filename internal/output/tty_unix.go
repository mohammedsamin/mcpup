//go:build darwin || linux

package output

import (
	"syscall"
	"unsafe"
)

func isTerminal(fd uintptr) bool {
	var wsz [4]uint16
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&wsz[0])))
	return err == 0
}

func enableRawMode(fd int) (restore func(), err error) {
	var orig syscall.Termios
	if _, _, e := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&orig)), 0, 0, 0); e != 0 {
		return nil, e
	}
	raw := orig
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.ISIG
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	if _, _, e := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&raw)), 0, 0, 0); e != 0 {
		return nil, e
	}
	return func() {
		syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(&orig)), 0, 0, 0)
	}, nil
}
