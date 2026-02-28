//go:build !(darwin || linux)

package output

import "errors"

func isTerminal(fd uintptr) bool { return false }

func enableRawMode(fd int) (func(), error) {
	return nil, errors.New("raw mode not supported on this platform")
}
