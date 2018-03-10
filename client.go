package go_i2cp

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"io"
	"strings"
	"sync"
)

const I2CP_CLIENT_VERSION = "0.9.11"
const TAG = CLIENT
const I2CP_PROTOCOL_INIT uint8 = 0x2a
const I2CP_MESSAGE_SIZE = 0xffff
const I2CP_MAX_SESSIONS = 0xffff
const I2CP_MAX_SESSIONS_PER_CLIENT = 32

const I2CP_MSG_ANY uint8 = 0
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
const ROUTER_CAN_HOST_LOOKUP uint32 = 1

type ClientProperty int

const (
	CLIENT_PROP_ROUTER_ADDRESS ClientProperty = iota
	CLIENT_PROP_ROUTER_PORT
	CLIENT_PROP_ROUTER_USE_TLS
	CLIENT_PROP_USERNAME
	CLIENT_PROP_PASSWORD
	NR_OF_I2CP_CLIENT_PROPERTIES
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
	session *Session
}
type RouterInfo struct {
	date         uint64
	version      Version
	capabilities uint32
}

type Client struct {
	logger          *LoggerCallbacks // TODO idk wat this is for
	callbacks       ClientCallBacks
	properties      [NR_OF_I2CP_CLIENT_PROPERTIES]string
	tcp             *Tcp
	outputStream    *Stream
	messageStream   *Stream
	router          RouterInfo
	outputQueue     []*Stream
	sessions        map[uint16]*Session
	n_sessions      int
	lookup          map[string]uint32
	lookupReq       map[uint32]LookupEntry
	lock            sync.Mutex
	connected       bool
	currentSession  *Session // *opaque in the C lib
	lookupRequestId uint32
}

// NewClient creates a new i2p client with the specified callbacks
func NewClient(callbacks ClientCallBacks) (c *Client) {
	c = new(Client)
	c.callbacks = callbacks
	LogInit(nil, ERROR)
	c.outputStream = bytes.NewBuffer(make([]byte, 0, I2CP_MESSAGE_SIZE))
	c.messageStream = bytes.NewBuffer(make([]byte, 0, I2CP_MESSAGE_SIZE))
	c.setDefaultProperties()
	c.lookup = make(map[string]uint32, 1000)
	c.lookupReq = make(map[uint32]LookupEntry, 1000)
	c.sessions = make(map[uint16]*Session)
	c.outputQueue = make([]*Stream, 0)
	return
}

func (c *Client) setDefaultProperties() {
	c.properties[CLIENT_PROP_ROUTER_ADDRESS] = "127.0.0.1"
	c.properties[CLIENT_PROP_ROUTER_PORT] = "7654"
	// TODO PARSE I2CP config file
}

// TODO Write messageGetDate
func (c *Client) messageGetDate(queue bool) {
	Debug(PROTOCOL, "Sending GetDateMessage")
	c.messageStream.Reset()
	c.messageStream.WriteString(I2CP_CLIENT_VERSION)
	/* write new 0.9.10 auth mapping if username property is set */
	if c.properties[CLIENT_PROP_USERNAME] != "" {
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
	if err := c.sendMessage(I2CP_MSG_GET_DATE, c.messageStream, queue); err != nil {
		Error(0, "%s", "error while sending GetDateMessage.")
	}
}

func (c *Client) sendMessage(typ uint8, stream *Stream, queue bool) (err error) {
	send := NewStream(make([]byte, stream.Len()+4+1))
	err = send.WriteUint32(uint32(stream.Len()))
	err = send.WriteByte(typ)
	_, err = send.Write(stream.Bytes())
	if queue {
		Debug(PROTOCOL, "Putting %d bytes message on the output queue.", send.Len())
		c.lock.Lock()
		c.outputQueue = append(c.outputQueue, send)
		c.lock.Unlock()
	} else {
		_, err = c.tcp.Send(send)
	}
	return
}

func (c *Client) recvMessage(typ uint8, stream *Stream, dispatch bool) (err error) {
	length := uint32(0)
	msgType := uint8(0)
	var i int
	firstFive := NewStream(make([]byte, 5))
	i, err = c.tcp.Receive(firstFive)
	if i == 0 {
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
	case I2CP_MSG_SET_DATE:
		c.onMsgSetDate(stream)
	case I2CP_MSG_DISCONNECT:
		c.onMsgDisconnect(stream)
	case I2CP_MSG_PAYLOAD_MESSAGE:
		c.onMsgPayload(stream)
	case I2CP_MSG_MESSAGE_STATUS:
		c.onMsgStatus(stream)
	case I2CP_MSG_DEST_REPLY:
		c.onMsgDestReply(stream)
	case I2CP_MSG_BANDWIDTH_LIMITS:
		c.onMsgBandwithLimit(stream)
	case I2CP_MSG_SESSION_STATUS:
		c.onMsgSessionStatus(stream)
	case I2CP_MSG_REQUEST_VARIABLE_LEASESET:
		c.onMsgReqVariableLease(stream)
	case I2CP_MSG_HOST_REPLY:
		c.onMsgHostReply(stream)
	default:
		Info(TAG, "%s", "recieved unhandled i2cp message.")
	}
}
func (c *Client) onMsgSetDate(stream *Stream) {
	Debug(TAG|PROTOCOL, "Received SetDate message.")
	var err error
	c.router.date, err = stream.ReadUint64()
	var verLength uint8
	verLength, err = stream.ReadByte()
	version := make([]byte, verLength)
	_, err = stream.Read(version)
	c.router.version = parseVersion(string(version))
	Debug(TAG|PROTOCOL, "Router version %s, date %ld", string(version), c.router.date)
	if err != nil {
		Error(TAG|PROTOCOL, "Could not read SetDate correctly data")
	}
	if c.router.version.compare(Version{major: 0, minor: 9, micro: 10, qualifier: 0}) >= 0 {
		c.router.capabilities |= ROUTER_CAN_HOST_LOOKUP
	}
}
func (c *Client) onMsgDisconnect(stream *Stream) {
	var size uint8
	var err error
	Debug(TAG|PROTOCOL, "Received Disconnect message")
	size, err = stream.ReadByte()
	strbuf := make([]byte, size)
	_, err = stream.Read(strbuf)
	Debug(TAG|PROTOCOL, "Received Disconnect message with reason %s", string(strbuf))
	if err != nil {
		Error(TAG|PROTOCOL, "Could not read msgDisconnect correctly data")
	}
	c.callbacks.onDisconnect(c, string(strbuf), nil)
}
func (c *Client) onMsgPayload(stream *Stream) {
	var gzipHeader = [3]byte{0x1f, 0x8b, 0x08}
	var testHeader [3]byte
	var protocol uint8
	var sessionId, srcPort, destPort uint16
	var messageId, payloadSize uint32
	var out Stream
	var err error
	var ret int
	Debug(TAG|PROTOCOL, "Received PayloadMessage message")
	sessionId, err = stream.ReadUint16()
	messageId, err = stream.ReadUint32()
	session, ok := c.sessions[sessionId]
	if !ok {
		Fatal(TAG|FATAL, "Session id %d does not match any of our currently initiated sessions by %p", sessionId, c)
	}
	payloadSize, err = stream.ReadUint32()
	// validate gzip header
	var msgStream = bytes.NewBuffer(stream.Bytes())
	ret, err = stream.Read(testHeader[:])
	if testHeader != gzipHeader {
		Warning(TAG, "Payload validation failed, skipping payload")
		return
	}
	var payload = bytes.NewBuffer(make([]byte, 0xffff))
	var decompress io.ReadCloser
	decompress, err = zlib.NewReader(msgStream)
	io.Copy(payload, decompress)
	decompress.Close()
	if payload.Len() > 0 {
		// finish reading header
		// skip gzip flags
		_, err = stream.ReadByte()
		srcPort, err = stream.ReadUint16()
		destPort, err = stream.ReadUint16()
		_, err = stream.ReadByte()
		protocol, err = stream.ReadByte()
		session.dispatchMessage(protocol, srcPort, destPort, payload)
	}

}
func (c *Client) onMsgStatus(stream *Stream) {
	var status uint8
	var sessionId uint16
	var messageId, size, nonce uint32
	var err error
	Debug(TAG|PROTOCOL, "Received MessageStatus message")
	sessionId, err = stream.ReadUint16()
	messageId, err = stream.ReadUint32()
	status, err = stream.ReadByte()
	size, err = stream.ReadUint32()
	nonce, err = stream.ReadUint32()
	Debug(TAG|PROTOCOL, "Message status; session id %d, message id %d, status %d, size %d, nonce %d", sessionId, messageId, status, size, nonce)
}
func (c *Client) onMsgDestReply(stream *Stream) {
	var b32 string
	var destination Destination
	var lup LookupEntry
	var err error
	var requestId uint32
	Debug(TAG|PROTOCOL, "Received DestReply message.")
	if stream.Len() != 32 {
		destination, err = NewDestinationFromMessage(stream)
		if err != nil {
			Fatal(TAG|FATAL, "Failed to construct destination from stream")
		}
		b32 = destination.b32
	} else {
		bits := GetCryptoInstance().EncodeStream(CODEC_BASE32, stream)
		b32 = string(bits.Bytes()) + ".b32.i2p"
		Debug(TAG, "Could not resolve destination")
	}
	requestId = c.lookup[b32]
	delete(c.lookup, b32)
	lup = c.lookupReq[requestId]
	delete(c.lookupReq, requestId)
	if lup == (LookupEntry{}) {
		Warning(TAG, "No sesssion for destination lookup of address '%s'", b32)
	} else {
		lup.session.dispatchDestination(requestId, b32, &destination)
	}
}
func (c *Client) onMsgBandwithLimit(stream *Stream) {
	Debug(TAG|PROTOCOL, "Received BandwidthLimits message.")
}
func (c *Client) onMsgSessionStatus(stream *Stream) {
	var sess *Session
	var sessionID uint16
	var sessionStatus uint8
	var err error
	Debug(TAG|PROTOCOL, "Received SessionStatus message.")
	sessionID, err = stream.ReadUint16()
	sessionStatus, err = stream.ReadByte()
	if sessionStatus == I2CP_SESSION_STATUS_CREATED {
		if c.currentSession == nil {
			Error(TAG, "Received session status created without waiting for it %p", c)
			return
		}
		c.currentSession.id = sessionID
		c.sessions[sessionID] = c.currentSession
		c.currentSession = nil
	}
	sess = c.sessions[sessionID]
	if sess == nil {
		Fatal(TAG|FATAL, "Session with id %d doesn't exists in client instance %p.", sessionID, c)
	} else {
		sess.dispatchStatus(sessionStatus)
	}
}
func (c *Client) onMsgReqVariableLease(stream *Stream) {
	var t int
	var sessionId uint16
	var tunnels uint8
	var sess *Session
	var leases []*Lease
	var err error
	Debug(TAG|PROTOCOL, "Received RequestVariableLeaseSet message.")
	sessionId, err = stream.ReadUint16()
	tunnels, err = stream.ReadByte()
	sess = c.sessions[sessionId]
	if sess == nil {
		Fatal(TAG|FATAL, "Session with id %d doesn't exist in client instance %p.", sessionId, c)
	}
	leases = make([]*Lease, tunnels)
	for i := uint8(0); i < tunnels; i++ {
		leases[i] = NewLeaseFromStream(stream)
	}
	c.msgCreateLeaseSet(sess, tunnels, leases, true)
}
func (c *Client) onMsgHostReply(stream *Stream) {
	var result uint8
	var sessionId uint16
	var requestId uint32
	var sess *Session
	var dest *Destination
	var lup LookupEntry
	var err error
	Debug(TAG|PROTOCOL, "Received HostReply message.")
	sessionId, err = stream.ReadUint16()
	requestId, err = stream.ReadUint32()
	result, err = stream.ReadByte()
	if result == 0 {
		dst, err := NewDestinationFromMessage(stream)
		if err != nil {
			Fatal(TAG|FATAL, "Failed to construct destination from stream.")
		}
	}
	sess = c.sessions[sessionId]
	if sess == nil {
		Fatal(TAG|FATAL, "Session with id %d doesn't exist in client instance %p.", sessionId, c)
	}
	lup = c.lookupReq[requestId]
	delete(c.lookupReq, requestId)
	sess.dispatchDestination(requestId, lup.address, dest)
}

func (c *Client) configFileParseCallback(name, value string) {
	switch name {
	case "i2cp.tcp.host":
		c.properties[CLIENT_PROP_ROUTER_ADDRESS] = value
	case "i2cp.tcp.port":
		c.properties[CLIENT_PROP_ROUTER_PORT] = value
	case "i2cp.tcp.SSL":
		c.properties[CLIENT_PROP_ROUTER_USE_TLS] = value
	case "i2cp.tcp.username":
		c.properties[CLIENT_PROP_USERNAME] = value
	case "i2cp.tcp.password":
		c.properties[CLIENT_PROP_PASSWORD] = value
	default:
		break
	}
}
func (c *Client) msgCreateLeaseSet(session *Session, tunnels uint8, leases []*Lease, queue bool) {
	var err error
	var nullbytes [256]byte
	var leaseSet *Stream
	var config *SessionConfig
	var dest *Destination
	var sgk *SignatureKeyPair
	Debug(TAG|PROTOCOL, "Sending CreateLeaseSetMessage")
	leaseSet = NewStream(make([]byte, 4096))
	config = session.config
	dest = config.destination
	sgk = &dest.sgk
	// memset 0 nullbytes
	for i := 0; i < len(nullbytes); i++ {
		nullbytes[i] = 0
	}
	// construct the message
	c.messageStream.WriteUint16(session.id)
	c.messageStream.Write(nullbytes[:20])
	c.messageStream.Write(nullbytes[:256])
	//Build leaseset stream and sign it
	dest.WriteToMessage(leaseSet)
	leaseSet.Write(nullbytes[:256])
	GetCryptoInstance().WritePublicSignatureToStream(sgk, leaseSet)
	leaseSet.WriteByte(tunnels)
	for i := uint8(0); i < tunnels; i++ {
		leases[i].WriteToMessage(leaseSet)
	}
	GetCryptoInstance().SignStream(sgk, leaseSet)
	c.messageStream.Write(leaseSet.Bytes())
	if err = c.sendMessage(I2CP_MSG_CREATE_LEASE_SET, c.messageStream, queue); err != nil {
		Error(TAG, "Error while sending CreateLeaseSet")
	}
}
func (c *Client) msgGetDate(queue bool) {
	var err error
	var auth *Stream
	Debug(TAG|PROTOCOL, "Sending GetDateMessage")
	c.messageStream.Reset()
	c.messageStream.Write([]byte(I2CP_CLIENT_VERSION))
	if len(c.properties[CLIENT_PROP_USERNAME]) > 0 {
		auth = NewStream(make([]byte, 0, 512))
		auth.Write([]byte("i2cp.password="))
		auth.Write([]byte(c.properties[CLIENT_PROP_PASSWORD]))
		auth.Write([]byte(";"))
		auth.Write([]byte("i2cp.username="))
		auth.Write([]byte(c.properties[CLIENT_PROP_USERNAME]))
		auth.Write([]byte(";"))
		c.messageStream.WriteUint16(uint16(auth.Len()))
		c.messageStream.Write(auth.Bytes())
	}
	if err = c.sendMessage(I2CP_MSG_SET_DATE, c.messageStream, queue); err != nil {
		Error(TAG, "Error while sending GetDateMessage")
	}
}
func (c *Client) msgCreateSession(config *SessionConfig, queue bool) {
	var err error
	Debug(TAG|PROTOCOL, "Sending CreateSessionMessage")
	c.messageStream.Reset()
	config.writeToMessage(c.messageStream)
	if err = c.sendMessage(I2CP_MSG_CREATE_SESSION, c.messageStream, queue); err != nil {
		Error(TAG, "Error while sending CreateSessionMessage.")
	}
}
func (c *Client) msgDestLookup(hash []byte, queue bool) {
	Debug(TAG|PROTOCOL, "Sending DestLookupMessage.")
	c.messageStream.Reset()
	c.messageStream.Write(hash)
	if err := c.sendMessage(I2CP_MSG_DEST_LOOKUP, c.messageStream, queue); err != nil {
		Error(TAG, "Error while sending DestLookupMessage.")
	}
}
func (c *Client) msgHostLookup(sess *Session, requestId, timeout uint32, typ uint8, data []byte, queue bool) {
	var sessionId uint16
	Debug(TAG|PROTOCOL, "Sending HostLookupMessage.")
	c.messageStream.Reset()
	sessionId = sess.id
	c.messageStream.WriteUint16(sessionId)
	c.messageStream.WriteUint32(requestId)
	c.messageStream.WriteUint32(timeout)
	c.messageStream.WriteByte(typ)
	if typ == HOST_LOOKUP_TYPE_HASH {
		c.messageStream.Write(data)
	}
	if err := c.sendMessage(I2CP_MSG_HOST_LOOKUP, c.messageStream, queue); err != nil {
		Error(TAG, "Error while sending HostLookupMessage")
	}
}
func (c *Client) msgGetBandwidthLimits(queue bool) {
	Debug(TAG|PROTOCOL, "Sending GetBandwidthLimitsMessage.")
	c.messageStream.Reset()
	if err := c.sendMessage(I2CP_MSG_GET_BANDWIDTH_LIMITS, c.messageStream, queue); err != nil {
		Error(TAG, "Error while sending GetBandwidthLimitsMessage")
	}
}
func (c *Client) msgDestroySession(sess *Session, queue bool) {
	Debug(TAG|PROTOCOL, "Sending DestroySessionMessage")
	c.messageStream.Reset()
	c.messageStream.WriteUint16(sess.id)
	if err := c.sendMessage(I2CP_MSG_DESTROY_SESSION, c.messageStream, queue); err != nil {
		Error(TAG, "Error while sending DestroySessionMessage")
	}
}
func (c *Client) msgSendMessage(sess *Session, dest *Destination, protocol uint8, srcPort, destPort uint16, payload *Stream, nonce uint32, queue bool) {
	Debug(TAG|PROTOCOL, "Sending SendMessageMessage")
	out := bytes.NewBuffer(make([]byte, 0xffff))
	c.messageStream.Reset()
	c.messageStream.WriteUint16(sess.id)
	dest.WriteToMessage(c.messageStream)
	compress := gzip.NewWriter(out)
	compress.Write(payload.Bytes())
	compress.Close()
	header := out.Bytes()[:10]
	binary.LittleEndian.PutUint16(header[4:6], srcPort)
	binary.LittleEndian.PutUint16(header[6:8], destPort)
	header[9] = protocol
	c.messageStream.WriteUint32(uint32(out.Len()))
	c.messageStream.Write(out.Bytes())
	c.messageStream.WriteUint32(nonce)
	if err := c.sendMessage(I2CP_MSG_SEND_MESSAGE, c.messageStream, queue); err != nil {
		Error(TAG, "Error while sending SendMessageMessage")
	}
}
func (c *Client) Connect() {
	Info(0, "Client connecting to i2cp at %s:%s", c.properties[CLIENT_PROP_ROUTER_ADDRESS], c.properties[CLIENT_PROP_ROUTER_PORT])
	err := c.tcp.Connect()
	if err != nil {
		panic(err)
	}
	c.outputStream.Reset()
	c.outputStream.WriteByte(I2CP_PROTOCOL_INIT)
	_, err = c.tcp.Send(c.outputStream)
	Debug(PROTOCOL, "Sending protocol byte message")
	c.msgGetDate(false)
	c.recvMessage(I2CP_MSG_SET_DATE, c.messageStream, true)
}

func (c *Client) CreateSession(sess *Session) {
	var config *SessionConfig
	if c.n_sessions == I2CP_MAX_SESSIONS_PER_CLIENT {
		Warning(TAG, "Maximum number of session per client connection reached.")
		return
	}
	sess.config.SetProperty(SESSION_CONFIG_PROP_I2CP_FAST_RECEIVE, "true")
	sess.config.SetProperty(SESSION_CONFIG_PROP_I2CP_MESSAGE_RELIABILITY, "none")
	c.msgCreateSession(sess.config, false)
	c.currentSession = sess
	c.recvMessage(I2CP_MSG_ANY, c.messageStream, true)
}

func (c *Client) ProcessIO() error {
	c.lock.Lock()
	for _, stream := range c.outputQueue {
		Debug(TAG|PROTOCOL, "Sending %d bytes message", stream.Len())
		ret, err := c.tcp.Send(stream)
		if ret < 0 {
			c.lock.Unlock()
			return err
		}
		if ret == 0 {
			break
		}
	}
	c.lock.Unlock()
	for c.tcp.CanRead() {
		if err := c.recvMessage(I2CP_MSG_ANY, c.messageStream, true); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) DestinationLookup(session *Session, address string) (requestId uint32) {
	var out *Stream
	var lup LookupEntry
	b32Len := 52 + 8
	defaultTimeout := uint32(30000)
	routerCanHostLookup := (c.router.capabilities & ROUTER_CAN_HOST_LOOKUP) == ROUTER_CAN_HOST_LOOKUP
	if !routerCanHostLookup && len(address) != b32Len {
		Warning(TAG, "Address '%s' is not a b32 address %d.", address, len(address))
		return
	}
	in := NewStream(make([]byte, 512))
	if len(address) == b32Len {
		Debug(TAG, "Lookup of b32 address detected, decode and use hash for faster lookup.")
		host := address[:strings.Index(address, ".")]
		in.Write([]byte(host))
		dout, _ := GetCryptoInstance().DecodeStream(CODEC_BASE32, in)
		out = &dout
		if out.Len() == 0 {
			Warning(TAG, "Failed to decode hash of address '%s'", address)
		}
	}
	lup = LookupEntry{address: address, session: session}
	c.lookupRequestId += 1
	requestId = c.lookupRequestId
	c.lookupReq[requestId] = lup
	if routerCanHostLookup {
		if out == nil || out.Len() == 0 {
			c.msgHostLookup(session, requestId, defaultTimeout, HOST_LOOKUP_TYPE_HOST, []byte(address), true)
		} else {
			c.msgHostLookup(session, requestId, defaultTimeout, HOST_LOOKUP_TYPE_HASH, out.Bytes(), true)
		}
	} else {
		c.lookup[address] = requestId
		c.msgDestLookup(out.Bytes(), true)
	}
	return requestId
}

func (c *Client) Disconnect() {
	Info(TAG, "Disconnection client %p", c)
	c.tcp.Disconnect()
}

func (c *Client) SetProperty(property ClientProperty, value string) {
	c.properties[property] = value
	switch property {
	case CLIENT_PROP_ROUTER_ADDRESS:
		c.tcp.SetProperty(TCP_PROP_ADDRESS, c.properties[CLIENT_PROP_ROUTER_ADDRESS])
	case CLIENT_PROP_ROUTER_PORT:
		c.tcp.SetProperty(TCP_PROP_PORT, c.properties[CLIENT_PROP_ROUTER_PORT])
	case CLIENT_PROP_ROUTER_USE_TLS:
		c.tcp.SetProperty(TCP_PROP_USE_TLS, c.properties[CLIENT_PROP_ROUTER_USE_TLS])
	}
}

func (c *Client) GetProperty(property ClientProperty) string {
	return c.properties[property]
}

func (c *Client) IsConnected() bool {
	return c.tcp.IsConnected()
}
