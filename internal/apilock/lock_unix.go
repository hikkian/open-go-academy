//go:build linux || darwin || freebsd || openbsd || netbsd

// Package apilock prevents two API processes from recovering or grading the same database.
package apilock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

type Lock struct{ file *os.File }

func Acquire(path string) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("database must be a regular file")
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("database already used by another API process: %w", err)
	}
	return &Lock{f}, nil
}
func (l *Lock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	err := l.file.Close()
	l.file = nil
	return err
}
