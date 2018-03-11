package go_i2cp

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"time"
)

type TcpProperty int

const (
	TCP_PROP_ADDRESS TcpProperty = iota
	TCP_PROP_PORT
	TCP_PROP_USE_TLS
	TCP_PROP_TLS_CLIENT_CERTIFICATE
	NR_OF_TCP_PROPERTIES
)

var CAFile = "/etc/ssl/certs/ca-certificates.crt"
var defaultRouterAddress = "127.0.0.1:7654"

const USE_TLS = true

func (tcp *Tcp) Init() (err error) {
	tcp.address, err = net.ResolveTCPAddr("tcp", defaultRouterAddress)
	return
}

func (tcp *Tcp) Connect() (err error) {
	if USE_TLS {
		roots, _ := x509.SystemCertPool()
		tcp.tlsConn, err = tls.Dial("tcp", tcp.address.String(), &tls.Config{RootCAs: roots})
	} else {
		tcp.conn, err = net.DialTCP("tcp", nil, tcp.address)
		if err == nil {
			err = tcp.conn.SetKeepAlive(true)
		}
	}
	_ = err // currently unused
	return
}

func (tcp *Tcp) Send(buf *Stream) (i int, err error) {
	if USE_TLS {
		i, err = tcp.tlsConn.Write(buf.Bytes())
	} else {
		i, err = tcp.conn.Write(buf.Bytes())
	}
	return
}

func (tcp *Tcp) Receive(buf *Stream) (i int, err error) {
	if USE_TLS {
		i, err = tcp.tlsConn.Read(buf.Bytes())
	} else {
		i, err = tcp.conn.Read(buf.Bytes())
	}
	return
}

func (tcp *Tcp) CanRead() bool {
	var one []byte
	if USE_TLS {
		tcp.tlsConn.SetReadDeadline(time.Now())
		if _, err := tcp.tlsConn.Read(one); err == io.EOF {
			Debug(TCP, "%s detected closed LAN connection", tcp.address.String())
			defer tcp.Disconnect()
			return false
		} else {
			var zero time.Time
			tcp.tlsConn.SetReadDeadline(zero)
			return tcp.tlsConn.ConnectionState().HandshakeComplete
		}
	} else {
		tcp.conn.SetReadDeadline(time.Now())
		if _, err := tcp.conn.Read(one); err == io.EOF {
			Debug(TCP, "%s detected closed LAN connection", tcp.address.String())
			defer tcp.Disconnect()
			return false
		} else {
			var zero time.Time
			tcp.conn.SetReadDeadline(zero)
			return true
		}
	}
}

func (tcp *Tcp) Disconnect() {
	if USE_TLS {
		tcp.tlsConn.Close()
	} else {
		tcp.conn.Close()
	}
}

func (tcp *Tcp) IsConnected() bool {
	return tcp.CanRead()
}

func (tcp *Tcp) SetProperty(property TcpProperty, value string) {
	tcp.properties[property] = value
}
func (tcp *Tcp) GetProperty(property TcpProperty) string {
	return tcp.properties[property]
}

type Tcp struct {
	address    *net.TCPAddr
	conn       *net.TCPConn
	tlsConn    *tls.Conn
	properties [NR_OF_TCP_PROPERTIES]string
}
