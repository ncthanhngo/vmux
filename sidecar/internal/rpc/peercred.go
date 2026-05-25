package rpc

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// checkPeerUID rejects connections whose peer is not the same UID as the
// sidecar. The 0600 socket already restricts access, but this is defense in
// depth on multi-user machines (the socket lives under the user's home).
func checkPeerUID(c net.Conn) error {
	uc, ok := c.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("rpc: non-unix connection")
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
		// Darwin returns the peer credentials via LOCAL_PEERCRED.
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
		return fmt.Errorf("rpc: peer uid %d != %d", uid, self)
	}
	return nil
}
