package relay

import (
	"net"
	"time"
)

type TcpKeepAliveListener struct {
	*net.TCPListener
}

func (listener TcpKeepAliveListener) Accept() (net.Conn, error) {
	tcpConn, err := listener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tcpConn.SetKeepAlive(true)
	tcpConn.SetKeepAlivePeriod(30 * time.Second)
	return tcpConn, nil
}
