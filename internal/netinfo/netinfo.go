package netinfo

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func ServerPort(addr string) int {
	portText := strings.TrimPrefix(addr, ":")
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 {
		return 8080
	}
	return port
}

func DetectLANIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || udpAddr.IP == nil {
		return ""
	}

	ipv4 := udpAddr.IP.To4()
	if ipv4 == nil || ipv4.IsLoopback() {
		return ""
	}

	return ipv4.String()
}

func URLs(addr string) (localURL, lanURL string, hasLAN bool) {
	port := ServerPort(addr)
	localURL = fmt.Sprintf("http://localhost:%d", port)

	lanIP := DetectLANIP()
	if lanIP == "" {
		return localURL, "", false
	}

	return localURL, fmt.Sprintf("http://%s:%d", lanIP, port), true
}
