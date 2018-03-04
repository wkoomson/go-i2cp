package go_i2cp
import (
	sll "github.com/emirpasic/gods/lists/singlylinkedlist"
	"net/rpc"
	"bytes"
	"sync"
)
const I2P_CLIENT_VERSION string = "0.9.11"
const TAG = CLIENT
const I2CP_PROTOCOL_INIT uint8 = 0x2a
const I2CP_MESSAGE_SIZE int = 0xffff
const I2CP_MAX_SESSIONS int = 0xffff
const I2CP_MAX_SESSIONS_PER_CLIENT int = 32

const I2CP_MSG_ANY int = 0
const I2CP_MSG_BANDWIDTH_LIMITS uint8 = 23
const I2CP_MSG_CREATE_LEASE_SET uint8 = 4
const I2CP_MSG_CREATE_SESSION uint8 = 1
const I2CP_MSG_DEST_LOOKUP uint8 = 34
const I2CP_MSG_DEST_REPLY uint8 = 35
const I2CP_MSG_DESTROY_SESSION uint8 = 3
const I2CP_MSG_DISCONNECT uint8 = 30
const I2CP_MSG_GET_BANDWIDTH_LIMITS uint8 = 8
const I2CP_MSG_GET_DATE uint8 = 32
const I2CP_MSG_HOST_LOOKUP uint8 = 38
const I2CP_MSG_HOST_REPLY uint8 = 39
const I2CP_MSG_MESSAGE_STATUS uint8 = 22
const I2CP_MSG_PAYLOAD_MESSAGE uint8 = 31
const I2CP_MSG_REQUEST_LEASESET uint8 = 21
const I2CP_MSG_REQUEST_VARIABLE_LEASESET uint8 = 37
const I2CP_MSG_SEND_MESSAGE uint8 = 5
const I2CP_MSG_SESSION_STATUS uint8 = 20
const I2CP_MSG_SET_DATE uint8 = 33

/* Router capabilities */
const ROUTER_CAN_HOST_LOOKUP int = 1

const (
	CLIENT_PROP_ROUTER_ADDRESS   = iota
	CLIENT_PROP_ROUTER_PORT      = iota
	CLIENT_PROP_ROUTER_USE_TLS   = iota
	CLIENT_PROP_USERNAME         = iota
	CLIENT_PROP_PASSWORD         = iota
	NR_OF_I2CP_CLIENT_PROPERTIES = iota
)
const (
	PROTOCOL_STREAMING    = 6
	PROTOCOL_DATAGRAM     = 17
	PROTOCOL_RAW_DATAGRAM = 18
)
const (
	HOST_LOOKUP_TYPE_HASH = iota
	HOST_LOOKUP_TYPE_HOST = iota
)

type ClientCallBacks struct {
	opaque       *interface{}
	onDisconnect func(*Client, string, *interface{})
	onLog        func(*Client, LoggerTags, string)
}
type LookupEntry struct {
	address string
	session Session
}
type RouterInfo struct {
	date uint64
	version string
	capabilities uint32
}

type Client struct {
	logger *LoggerCallbacks // TODO idk wat this is for
	callbacks ClientCallBacks
	properties [NR_OF_I2CP_CLIENT_PROPERTIES]string
	tcp *Tcp
	outputStream *Stream
	messageStream *Stream
	router RouterInfo
	outputQueue *sll.List
	sessions []*Session
	n_sessions int
	lookup map[string]LookupEntry
	lookupReq map[int]string
	lock sync.Mutex
}

// NewClient creates a new i2p client with the specified callbacks
func NewClient(callbacks ClientCallBacks) (c *Client) {
	c = new(Client)
	c.callbacks = callbacks
	LogInit(nil, ERROR)
	c.outputStream = bytes.NewBuffer(make([]byte, 0, I2CP_MESSAGE_SIZE))
	c.messageStream = bytes.NewBuffer(make([]byte, 0, I2CP_MESSAGE_SIZE))
	c.setDefaultProperties()
	c.lookup = make(map[string]LookupEntry, 1000)
	c.lookupReq = make(map[int]string, 1000)
	c.outputQueue = sll.New()
	return
}

func (c *Client) setDefaultProperties() {
	c.properties[CLIENT_PROP_ROUTER_ADDRESS] = "127.0.0.1"
	c.properties[CLIENT_PROP_ROUTER_PORT] = "7654"
	c.properties[CLIENT_PROP_ROUTER_PORT] = "7654"
	// TODO PARSE I2CP config file
}
func (c *Client) Connect() {
	Info(0, "Client connecting to i2cp at %s:%s", c.properties[CLIENT_PROP_ROUTER_ADDRESS], c.properties[CLIENT_PROP_ROUTER_PORT]);
	err := c.tcp.Connect()
	if err != nil {
		panic(err)
	}
	c.outputStream.Reset()
	c.outputStream.WriteByte(I2CP_PROTOCOL_INIT)
	_, err = c.tcp.Send(c.outputStream)
	Debug(PROTOCOL, "Sending protocol byte message")
	c.messageGetDate(false)
	c.recvMessage(I2CP_MSG_SET_DATE, c.messageStream, true)
}

// TODO Write messageGetDate
func (c *Client) messageGetDate(queue bool) {
	Debug(PROTOCOL, "Sending GetDateMessage")
	c.messageStream.Reset()
	c.messageStream.WriteString(I2P_CLIENT_VERSION)
	/* write new 0.9.10 auth mapping if username property is set */
	if c.properties[CLIENT_PROP_USERNAME] != ""	{
		auth := NewStream(make([]byte, 0, 512))
		auth.WriteString("i2cp.password")
		auth.WriteByte('=')
		auth.WriteString(c.properties[CLIENT_PROP_PASSWORD])
		auth.WriteByte(';')
		auth.WriteString("i2cp.username")
		auth.WriteByte('=')
		auth.WriteString(c.properties[CLIENT_PROP_USERNAME])
		auth.WriteByte(';')
		c.messageStream.WriteUint16(uint16(auth.Len()))
		c.messageStream.Write(auth.Bytes())
		auth.Reset()
	}
	if err := c.sendMessage(I2CP_MSG_GET_DATE, c.messageStream, queue);
		err != nil {
		Error(0, "%s", "error while sending GetDateMessage.");
	}
}

func (c *Client) sendMessage(typ uint8, stream *Stream, queue bool) (err error) {
	send := NewStream(make([]byte, stream.Len() + 4 + 1))
	err = send.WriteUint32(uint32(stream.Len()))
	err = send.WriteByte(typ)
	_, err = send.Write(stream.Bytes())
	if queue {
		Debug(PROTOCOL, "Putting %d bytes message on the output queue.", send.Len())
		c.lock.Lock()
		c.outputQueue.Add(send)
		c.lock.Unlock()
	} else {
		err = c.tcp.Send(send)
	}
	return
}

func (c *Client) recvMessage(typ uint8, stream *Stream, dispatch bool) (err error) {
	length := uint32(0)
	msgType := uint8(0)
	var i int
	firstFive := NewStream(make([]byte, 5))
	i, err = c.tcp.Receive(firstFive)
	if i ==0 {
		c.callbacks.onDisconnect(c, "Didn't receive anything", nil)
	}
	length, err = firstFive.ReadUint32()
	msgType, err = firstFive.ReadByte()
	if (typ == I2CP_MSG_SET_DATE) && (length > 0xffff) {
		Fatal(PROTOCOL, "Unexpected response, your router is probably configured to use SSL")
	}
	if length > 0xffff {
		Fatal(PROTOCOL, "unexpected message length, length > 0xffff")
	}
	if (typ != 0) && (msgType != typ) {
		Error(PROTOCOL, "expected message type %d, received %d", typ, msgType)
	}
	// receive rest
	stream.ChLen(int(length))
	i, err = c.tcp.Receive(stream)
	
	if dispatch {
		c.onMessage(msgType, stream)
	}
	return
}
func (c *Client) onMessage(msgType uint8, stream *Stream) {
	switch msgType {
	case I2CP_MSG_SET_DATE:	c.onMsgSetDate(stream)
	case I2CP_MSG_DISCONNECT:	c.onMsgDisconnect(stream)
	case I2CP_MSG_PAYLOAD_MESSAGE: c.onMsgPayload(stream)
	case I2CP_MSG_MESSAGE_STATUS: c.onMsgStatus(stream)
	case I2CP_MSG_DEST_REPLY: c.onMsgDestReply(stream)
	case I2CP_MSG_BANDWIDTH_LIMITS: c.onMsgBandwithLimit(stream)
	case I2CP_MSG_SESSION_STATUS: c.onMsgSessionStatus(stream)
	case I2CP_MSG_REQUEST_VARIABLE_LEASESET: c.onMsgReqVariableLease(stream)
	case I2CP_MSG_HOST_REPLY: c.onMsgHostReply(stream)
	default: Info(TAG, "%s", "recieved unhandled i2cp message.")
	}
}
func (c *Client) onMsgSetDate(stream *Stream) {
	Debug(TAG|PROTOCOL, "Received SetDate message.")
	var err error
	c.router.date, err = stream.ReadUint64()
	c.router.version, err = stream.R
	if err != nil {
		Error(TAG|PROTOCOL, "Could not read SetDate correctly data")
	}
}
func (c *Client) onMsgDisconnect(stream *Stream) {

}
func (c *Client) onMsgPayload(stream *Stream) {

}
func (c *Client) onMsgStatus(stream *Stream) {

}
func (c *Client) onMsgDestReply(stream *Stream) {

}
func (c *Client) onMsgBandwithLimit(stream *Stream) {

}
func (c *Client) onMsgSessionStatus(stream *Stream) {

}
func (c *Client) onMsgReqVariableLease(stream *Stream) {

}
func (c *Client) onMsgHostReply(stream *Stream) {

}