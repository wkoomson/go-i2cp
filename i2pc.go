package go_i2cp

func Init() {
	tcpInit()
}

func Deinit() {
	tcpDeinit()
}

const (
	CLIENT uint32 = (iota +1) << 8

)