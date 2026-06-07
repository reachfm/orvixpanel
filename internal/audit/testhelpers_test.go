package audit

import (
	"net"
	"strconv"
	"strings"
	"testing"
)

// listenUDP opens a UDP listener on a random local port. The
// returned *net.UDPConn is closed by the test via t.Cleanup.
func listenUDP(t *testing.T) (*net.UDPConn, error) {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", addr)
}

func splitHostPort(s string) (string, int, error) {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return "", 0, &net.AddrError{Err: "missing port", Addr: s}
	}
	port, err := strconv.Atoi(s[idx+1:])
	if err != nil {
		return "", 0, err
	}
	return s[:idx], port, nil
}
