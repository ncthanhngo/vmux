// Package peercred verifies that the peer of a local Unix socket connection is
// the same OS user as this process. The 0600 socket already restricts access;
// this is defense in depth for multi-user machines, applied to every vmux
// socket that accepts commands.
package peercred

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// Check returns an error if the connection's peer UID differs from this
// process's UID (or if credentials cannot be determined).
func Check(c net.Conn) error {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("peercred: non-unix connection")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return err
	}

	var (
		uid     uint32
		sockErr error
	)
	ctrlErr := raw.Control(func(fd uintptr) {
		// Darwin exposes peer credentials via LOCAL_PEERCRED.
		cred, e := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if e != nil {
			sockErr = e
			return
		}
		uid = cred.Uid
	})
	if ctrlErr != nil {
		return ctrlErr
	}
	if sockErr != nil {
		return sockErr
	}
	if self := uint32(os.Getuid()); uid != self {
		return fmt.Errorf("peercred: peer uid %d != %d", uid, self)
	}
	return nil
}
