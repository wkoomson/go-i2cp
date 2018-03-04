package go_i2cp

const I2P_CLIENT_VERSION string = "0.9.11"
const TAG string = "CLIENT"
const I2CP_PROTOCOL_INIT int = 0x2a
const I2CP_MESSAGE_SIZE int = 0xffff
const I2CP_MAX_SESSIONS int = 0xffff
const I2CP_MAX_SESSIONS_PER_CLIENT int = 32

const I2CP_MSG_ANY int = 0
const I2CP_MSG_BANDWIDTH_LIMITS int = 23
const I2CP_MSG_CREATE_LEASE_SET int = 4
const I2CP_MSG_CREATE_SESSION int = 1
const I2CP_MSG_DEST_LOOKUP int = 34
const I2CP_MSG_DEST_REPLY int = 35
const I2CP_MSG_DESTROY_SESSION int = 3
const I2CP_MSG_DISCONNECT int = 30
const I2CP_MSG_GET_BANDWIDTH_LIMITS int = 8
const I2CP_MSG_GET_DATE int = 32
const I2CP_MSG_HOST_LOOKUP int = 38
const I2CP_MSG_HOST_REPLY int = 39
const I2CP_MSG_MESSAGE_STATUS int = 22
const I2CP_MSG_PAYLOAD_MESSAGE int = 31
const I2CP_MSG_REQUEST_LEASESET int = 21
const I2CP_MSG_REQUEST_VARIABLE_LEASESET int = 37
const I2CP_MSG_SEND_MESSAGE int = 5
const I2CP_MSG_SESSION_STATUS int = 20
const I2CP_MSG_SET_DATE int = 33

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
	onDisconnect *func(*Client, string, *interface{})
	onLog        func(*Client, LoggerTags, string)
}
type Client struct {
}
