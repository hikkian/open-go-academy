//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd

package apilock

import "errors"

type Lock struct{}

func Acquire(string) (*Lock, error) {
	return nil, errors.New("database process locking is not supported on this OS; use the documented Linux environment")
}
func (*Lock) Close() error { return nil }
