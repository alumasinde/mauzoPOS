package printing

import (
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// A network printer receives the exact bytes on its raw port.
func TestNetworkSend(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	got := make(chan []byte, 4)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			b, _ := io.ReadAll(c)
			_ = c.Close()
			got <- b
		}
	}()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	p, _ := strconv.Atoi(port)
	cfg := Config{Kind: KindNetwork, Host: "127.0.0.1", Port: p}
	if s := Probe(cfg); s != "ready" {
		t.Fatalf("probe %s", s)
	}
	job := []byte{0x1B, '@', 'h', 'i'}
	if err := Send(cfg, job); err != nil {
		t.Fatal(err)
	}
	for range 2 { // the probe's empty connection, then the job
		if b := <-got; bytes.Equal(b, job) {
			return
		}
	}
	t.Fatal("job not received")
}

func TestFileAndUnconfigured(t *testing.T) {
	if err := Send(Config{Kind: KindNone}, []byte("x")); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("none: %v", err)
	}
	if Probe(Config{Kind: KindNetwork}) != "not_configured" {
		t.Fatal("empty host should be not_configured")
	}
	path := filepath.Join(t.TempDir(), "jobs.bin")
	cfg := Config{Kind: KindFile, Name: path}
	for range 2 {
		if err := Send(cfg, []byte("ab")); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "abab" {
		t.Fatalf("file jobs: %q %v", b, err)
	}
	if Probe(Config{Kind: KindNetwork, Host: "127.0.0.1", Port: 1}) != "offline" {
		t.Fatal("closed port should be offline")
	}
}

// Listing installed printers must not fail (empty is fine on CI).
func TestList(t *testing.T) {
	if _, err := List(); err != nil {
		t.Fatal(err)
	}
}
