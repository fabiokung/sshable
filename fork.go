package main

import (
	"runtime"
	"syscall"
)

func fork() (pid int, err syscall.Errno) {
	darwin := runtime.GOOS == "darwin"

	r1, r2, errno := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
	if errno != 0 {
		return 0, errno
	}

	if darwin && r2 == 1 {
		r1 = 0
	}

	return int(r1), 0
}
