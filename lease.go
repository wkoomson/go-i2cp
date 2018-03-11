package go_i2cp

type Lease struct {
	tunnelGateway [32]byte // sha256 of the RouterIdentity of the tunnel gateway
	tunnelId      uint32
	endDate       uint64
}

func NewLeaseFromStream(stream *Stream) (l *Lease, err error) {
	l = &Lease{}
	stream.Read(l.tunnelGateway[:])
	l.tunnelId, err = stream.ReadUint32()
	l.endDate, err = stream.ReadUint64()
	return
}

func (l *Lease) WriteToMessage(stream *Stream) (err error) {
	_, err = stream.Write(l.tunnelGateway[:])
	err = stream.WriteUint32(l.tunnelId)
	err = stream.WriteUint64(l.endDate)
	return
}
