package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	target := "http://127.0.0.1:8080/healthz"
	if len(os.Args) > 1 && os.Args[1] != "" {
		target = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if strings.HasPrefix(target, "tcp://") {
		dialer := net.Dialer{}
		conn, err := dialer.DialContext(ctx, "tcp", strings.TrimPrefix(target, "tcp://"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = conn.Close()
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintf(os.Stderr, "unhealthy status: %d\n", resp.StatusCode)
		os.Exit(1)
	}
}
