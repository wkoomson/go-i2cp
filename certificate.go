package go_i2cp

const (
	CERTIFICATE_NULL     uint8 = iota
	CERTIFICATE_HASHCASH uint8 = iota
	CERTIFICATE_SIGNED   uint8 = iota
	CERTIFICATE_MULTIPLE uint8 = iota
)

type Certificate struct {
	certType uint8
	data     []byte
	length   uint16
}

func NewCertificate(typ uint8) (cert Certificate) {
	cert.certType = typ
	return
}

func NewCertificateFromMessage(stream *Stream) (cert Certificate, err error) {
	cert.certType, err = stream.ReadByte()
	cert.length, err = stream.ReadUint16()
	if err != nil {
		return
	}
	if (cert.certType != CERTIFICATE_NULL) && (cert.length != 0) {
		Fatal(CERTIFICATE|PROTOCOL, "Only null certificates are allowed to have zero length.")
		return
	} else if cert.certType == CERTIFICATE_NULL {
		return
	}
	cert.data = make([]byte, cert.length)
	_, err = stream.Read(cert.data)
	return
}

func NewCertificateFromStream(stream *Stream) (Certificate, error) {
	return NewCertificateFromMessage(stream)
}

func (cert *Certificate) Copy() (newCert Certificate) {
	newCert.certType = cert.certType
	newCert.length = cert.length
	newCert.data = make([]byte, len(cert.data))
	copy(newCert.data, cert.data)
	return
}

func (cert *Certificate) WriteToMessage(stream *Stream) (err error) {
	err = stream.WriteByte(cert.certType)
	err = stream.WriteUint16(cert.length)
	if cert.length > 0 {
		_, err = stream.Write(cert.data)
	}
	return
}

func (cert *Certificate) WriteToStream(stream *Stream) error {
	return cert.WriteToMessage(stream)
}
