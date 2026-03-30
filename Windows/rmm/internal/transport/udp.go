package transport

import (
"fmt"
"net"
"time"
)

// SendUDP sends a UDP packet to targetIP:port with data
func SendUDP(targetIP string, port int, data []byte) error {
addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", targetIP, port))
if err != nil {
return fmt.Errorf("resolve address: %w", err)
}

conn, err := net.DialUDP("udp", nil, addr)
if err != nil {
return fmt.Errorf("dial UDP: %w", err)
}
defer conn.Close()

_, err = conn.Write(data)
return err
}

// ListenUDP returns a UDP listener on 0.0.0.0:port
func ListenUDP(port int) (*net.UDPConn, error) {
addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", port))
if err != nil {
return nil, fmt.Errorf("resolve: %w", err)
}

return net.ListenUDP("udp", addr)
}

// RecvUDP receives a single UDP packet and returns data + source address
func RecvUDP(conn *net.UDPConn) ([]byte, *net.UDPAddr, error) {
buf := make([]byte, 4096)
n, remoteAddr, err := conn.ReadFromUDP(buf)
if err != nil {
return nil, nil, err
}
return buf[:n], remoteAddr, nil
}

// SendUDPToAddr sends UDP data to a specific UDPAddr
func SendUDPToAddr(conn *net.UDPConn, addr *net.UDPAddr, data []byte) error {
_, err := conn.WriteToUDP(data, addr)
return err
}

// RecvUDPWithTimeout receives with a timeout
func RecvUDPWithTimeout(conn *net.UDPConn, timeout time.Duration) ([]byte, *net.UDPAddr, error) {
conn.SetReadDeadline(time.Now().Add(timeout))
defer conn.SetReadDeadline(time.Time{})
return RecvUDP(conn)
}
