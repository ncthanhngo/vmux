package pty

import "syscall"

// killGroup SIGKILLs the entire process group led by pid. creack/pty starts the
// child with Setsid, so its pid is also its process-group id; the negative
// argument targets the whole group, reaping any grandchildren.
func killGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}
