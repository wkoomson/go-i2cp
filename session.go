package go_i2cp

type SessionMessageStatus int

const (
	I2CP_MSG_STATUS_AVAILABLE SessionMessageStatus = iota
	I2CP_MSG_STATUS_ACCEPTED
	I2CP_MSG_STATUS_BEST_EFFORT_SUCCESS
	I2CP_MSG_STATUS_BEST_EFFORT_FAILURE
	I2CP_MSG_STATUS_GUARANTEED_SUCCESS
	I2CP_MSG_STATUS_GUARANTEED_FAILURE
	I2CP_MSG_STATUS_LOCAL_SUCCESS
	I2CP_MSG_STATUS_LOCAL_FAILURE
	I2CP_MSG_STATUS_ROUTER_FAILURE
	I2CP_MSG_STATUS_NETWORK_FAILURE
	I2CP_MSG_STATUS_BAD_SESSION
	I2CP_MSG_STATUS_BAD_MESSAGE
	I2CP_MSG_STATUS_OVERFLOW_FAILURE
	I2CP_MSG_STATUS_MESSAGE_EXPIRED
	I2CP_MSG_STATUS_MESSAGE_BAD_LOCAL_LEASESET
	I2CP_MSG_STATUS_MESSAGE_NO_LOCAL_TUNNELS
	I2CP_MSG_STATUS_MESSAGE_UNSUPPORTED_ENCRYPTION
	I2CP_MSG_STATUS_MESSAGE_BAD_DESTINATION
	I2CP_MSG_STATUS_MESSAGE_BAD_LEASESET
	I2CP_MSG_STATUS_MESSAGE_EXPIRED_LEASESET
	I2CP_MSG_STATUS_MESSAGE_NO_LEASESET
)

type SessionStatus int

const (
	I2CP_SESSION_STATUS_DESTROYED SessionStatus = iota
	I2CP_SESSION_STATUS_CREATED
	I2CP_SESSION_STATUS_UPDATED
	I2CP_SESSION_STATUS_INVALID
)

type SessionCallbacks struct {
	onMessage     func(session *Session, protocol uint8, srcPort, destPort uint16, payload *Stream)
	onStatus      func(session *Session, status SessionStatus)
	onDestination func(session *Session, requestId uint32, address string, dest *Destination)
}

type Session struct {
	id        uint16
	config    *SessionConfig
	client    *Client
	callbacks *SessionCallbacks
}

func NewSession(client *Client, callbacks SessionCallbacks) (sess *Session) {
	sess = &Session{}
	sess.client = client
	dest, _ := NewDestination()
	sess.config = &SessionConfig{destination: dest}
	sess.callbacks = &callbacks
	return
}
func (session *Session) SendMessage(destination *Destination, protocol uint8, srcPort, destPort uint16, payload *Stream, nonce uint32) {
	session.client.msgSendMessage(session, destination, protocol, srcPort, destPort, payload, nonce, true)
}
func (session *Session) Destination() *Destination {
	return session.config.destination
}
func (session *Session) dispatchMessage(protocol uint8, srcPort, destPort uint16, payload *Stream) {
	if session.callbacks == nil || session.callbacks.onMessage == nil {
		return
	}
	session.callbacks.onMessage(session, protocol, srcPort, destPort, payload)
}

func (session *Session) dispatchDestination(requestId uint32, address string, destination *Destination) {
	if session.callbacks == nil || session.callbacks.onDestination == nil {
		return
	}
	session.callbacks.onDestination(session, requestId, address, destination)
}

func (session *Session) dispatchStatus(status SessionStatus) {
	switch status {
	case I2CP_SESSION_STATUS_CREATED:
		Debug(SESSION, "Session %p is created", session)
	case I2CP_SESSION_STATUS_DESTROYED:
		Debug(SESSION, "Session %p is destroyed", session)
	case I2CP_SESSION_STATUS_UPDATED:
		Debug(SESSION, "Session %p is updated", session)
	case I2CP_SESSION_STATUS_INVALID:
		Debug(SESSION, "Session %p is invalid", session)
	}
	if session.callbacks == nil || session.callbacks.onStatus == nil {
		return
	}
	session.callbacks.onStatus(session, status)
}
