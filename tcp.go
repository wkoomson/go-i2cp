package go_i2cp

import "net"

func tcpInit() {
	address = net.TCPAddr{
		IP: net.ParseIP("127.0.0.1"),
		Port: 7654,
	}
}

func tcpDeinit() {

}

var address net.TCPAddr
var conn *net.TCPConn

func (tcp *Tcp) Connect() (err error) {
	conn, err = net.DialTCP("tcp", nil, &address )
	return
}

func (tcp *Tcp) Send(buf *Stream) (i int, err error) {
	i, err = conn.Write(buf.Bytes())
	return
}

func (tcp *Tcp) Receive(buf *Stream) (i int, err error) {
	i, err = conn.Read(buf.Bytes())
}

type Tcp struct {

}