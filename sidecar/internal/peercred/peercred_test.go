package peercred

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckSameUID(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "vmuxpc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		c, _ := ln.Accept()
		accepted <- c
	}()

	client, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	srvConn := <-accepted
	defer srvConn.Close()

	// The peer is this same process → same UID → must pass.
	if err := Check(srvConn); err != nil {
		t.Errorf("Check rejected same-uid peer: %v", err)
	}
}

func TestCheckRejectsNonUnix(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	if err := Check(c1); err == nil {
		t.Error("expected error for non-unix connection")
	}
}
