package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestServeGracefulShutdown(t *testing.T) {
	port := freePort(t)
	started := make(chan struct{})
	release := make(chan struct{})
	srv := newServer(port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		fmt.Fprint(w, "done")
	}))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, zerolog.Nop(), 5*time.Second, srv) }()

	var body string
	reqDone := make(chan error, 1)
	require.Eventually(t, func() bool {
		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)
	go func() {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err == nil {
			b := make([]byte, 4)
			n, _ := resp.Body.Read(b)
			body = string(b[:n])
			resp.Body.Close()
		}
		reqDone <- err
	}()
	<-started

	cancel()
	select {
	case <-done:
		t.Fatal("serve returned before in-flight request finished")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-reqDone)
	require.Equal(t, "done", body)
	require.NoError(t, <-done)
}

func TestServeReturnsListenError(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer l.Close()
	srv := newServer(l.Addr().(*net.TCPAddr).Port, http.NotFoundHandler())
	require.Error(t, serve(context.Background(), zerolog.Nop(), time.Second, srv))
}
