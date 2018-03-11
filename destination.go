package go_i2cp

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
)

const tag = DESTINATION
const PUB_KEY_SIZE = 256
const DIGEST_SIZE = 40
const DEST_SIZE = 4096

type Destination struct {
	cert       *Certificate
	sgk        SignatureKeyPair
	signPubKey *big.Int
	pubKey     [PUB_KEY_SIZE]byte
	digest     [DIGEST_SIZE]byte
	b32        string
	b64        string
}

func NewDestination() (dest Destination, err error) {
	nullCert := NewCertificate(CERTIFICATE_NULL)
	dest.cert = &nullCert
	dest.sgk, err = GetCryptoInstance().SignatureKeygen(DSA_SHA1)
	dest.signPubKey = dest.sgk.pub.Y
	dest.generateB32()
	dest.generateB64()
	return
}

func NewDestinationFromMessage(stream *Stream) (dest Destination, err error) {
	_, err = stream.Read(dest.pubKey[:])
	if err != nil {
		return
	}
	dest.signPubKey, err = GetCryptoInstance().PublicKeyFromStream(DSA_SHA1, stream)
	if err != nil {
		return
	}
	dest.sgk = SignatureKeyPair{}
	dest.sgk.priv.Y = dest.signPubKey
	dest.sgk.pub.Y = dest.signPubKey
	var cert Certificate
	cert, err = NewCertificateFromMessage(stream)
	if err != nil {
		return
	}
	dest.cert = &cert
	dest.generateB32()
	dest.generateB64()
	return
}

func NewDestinationFromStream(stream *Stream) (dest Destination, err error) {
	var cert Certificate
	var pubKeyLen uint16
	cert, err = NewCertificateFromStream(stream)
	dest.cert = &cert
	dest.sgk, err = GetCryptoInstance().SignatureKeyPairFromStream(stream)
	pubKeyLen, err = stream.ReadUint16()
	if pubKeyLen != PUB_KEY_SIZE {
		Fatal(tag, "Failed to load pub key len, %d != %d", pubKeyLen, PUB_KEY_SIZE)
	}
	_, err = stream.Read(dest.pubKey[:])
	dest.generateB32()
	dest.generateB64()
	return
}

func NewDestinationFromBase64(base64 string) (dest Destination, err error) {
	/* Same as decode, except from a filesystem / URL friendly set of characters,
	*  replacing / with ~, and + with -
	 */
	// see https://javadoc.freenetproject.org/freenet/support/Base64.html
	if len(base64) == 0 {
		err = errors.New("empty string")
		return
	}
	var replaced string
	// Convert from freenet to standard
	replaced = strings.Replace(base64, "~", "/", -1)
	replaced = strings.Replace(replaced, "-", "+", -1)
	stream := NewStream([]byte(replaced))
	var decoded *Stream
	decoded, err = GetCryptoInstance().DecodeStream(CODEC_BASE64, stream)
	return NewDestinationFromMessage(decoded)
}

func NewDestinationFromFile(file *os.File) (dest Destination, err error) {
	var stream Stream
	stream.loadFile(file)
	return NewDestinationFromStream(&stream)
}
func (dest *Destination) Copy() (newDest Destination) {
	newDest.cert = dest.cert
	newDest.signPubKey = dest.signPubKey
	newDest.pubKey = dest.pubKey
	newDest.sgk = dest.sgk
	newDest.b32 = dest.b32
	newDest.b64 = dest.b64
	newDest.digest = dest.digest
	return
}
func (dest *Destination) WriteToFile(filename string) (err error) {
	stream := NewStream(make([]byte, 0, DEST_SIZE))
	dest.WriteToStream(stream)
	var file *os.File
	file, err = os.Open(filename)
	stream.WriteTo(file)
	file.Close()
	return
}
func (dest *Destination) WriteToMessage(stream *Stream) (err error) {
	lena := len(dest.pubKey)
	_ = lena
	_, err = stream.Write(dest.pubKey[:])
	_, err = stream.Write(dest.signPubKey.Bytes()) //GetCryptoInstance().WriteSignatureToStream(&dest.sgk, stream)
	err = dest.cert.WriteToMessage(stream)
	lenb := stream.Len()
	_ = lenb
	return
}
func (dest *Destination) WriteToStream(stream *Stream) (err error) {
	err = dest.cert.WriteToStream(stream)
	err = GetCryptoInstance().WriteSignatureToStream(&dest.sgk, stream)
	err = stream.WriteUint16(PUB_KEY_SIZE)
	_, err = stream.Write(dest.pubKey[:])
	return
}

//Doesn't seem to be used anywhere??
func (dest *Destination) Verify() (verified bool, err error) {
	stream := NewStream(make([]byte, 0, DEST_SIZE))
	dest.WriteToMessage(stream)
	stream.Write(dest.digest[:])
	return GetCryptoInstance().VerifyStream(&dest.sgk, stream)
}

func (dest *Destination) generateB32() {
	stream := NewStream(make([]byte, 0, DEST_SIZE))
	dest.WriteToMessage(stream)
	cpt := GetCryptoInstance()
	hash := cpt.HashStream(HASH_SHA256, stream)
	b32 := cpt.EncodeStream(CODEC_BASE32, hash)
	length := b32.Len()
	_ = length
	dest.b32 = string(b32.Bytes())
	dest.b32 += ".b32.i2p"
	Debug(tag, "New destination %s", dest.b32)
}
func (dest *Destination) generateB64() {
	stream := NewStream(make([]byte, 0, DEST_SIZE))
	dest.WriteToMessage(stream)
	cpt := GetCryptoInstance()
	if stream.Len() > 0 {
		fmt.Printf("Stream len %d \n", stream.Len())
	}
	b64B := cpt.EncodeStream(CODEC_BASE64, stream)
	replaced := strings.Replace(string(b64B.Bytes()), "/", "~", -1)
	replaced = strings.Replace(replaced, "/", "~", -1)
	dest.b64 = replaced
}
