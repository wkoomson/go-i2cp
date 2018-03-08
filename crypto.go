package go_i2cp

import (
	"crypto"
	"crypto/dsa"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"math/big"
)

const tAG = CRYPTO

// Supported Hash algorithms
const (
	HASH_SHA1   uint8 = iota
	HASH_SHA256 uint8 = iota
)

// Supported signature algorithms
const (
	DSA_SHA1   uint32 = iota
	DSA_SHA256 uint32 = iota
)

// Supported codec algorithms
const (
	CODEC_BASE32 uint8 = iota
	CODEC_BASE64 uint8 = iota
)

type SignatureKeyPair struct {
	algorithmType uint32
	pub           dsa.PublicKey
	priv          dsa.PrivateKey
}

type Crypto struct {
	b64    *base64.Encoding
	b32    *base32.Encoding
	rng    io.Reader
	params dsa.Parameters
	sh1    hash.Hash
	sh256  hash.Hash
}

var singleton = Crypto{
	b64:   base64.StdEncoding,
	b32:   base32.StdEncoding,
	rng:   rand.Reader,
	sh1:   sha1.New(),
	sh256: sha256.New(),
}
var first = 0

func GetCryptoInstance() *Crypto {
	if first == 0 {
		dsa.GenerateParameters(&singleton.params, singleton.rng, dsa.L1024N160)
	}
	first++
	return &singleton
}

// Sign a stream using the specified algorithm
func (c *Crypto) SignStream(sgk *SignatureKeyPair, stream *Stream) (err error) {
	var r, s *big.Int
	out := NewStream(make([]byte, 40))
	c.sh1.Reset()
	sum := c.sh1.Sum(stream.Bytes())
	r, s, err = dsa.Sign(c.rng, &sgk.priv, sum)
	err = writeDsaSigToStream(r, s, out)
	stream.Write(out.Bytes())
	return
}

// Writes a 40-byte signature digest to the stream
func writeDsaSigToStream(r, s *big.Int, stream *Stream) (err error) {
	var rs, ss []byte
	var digest [81]byte
	for i := 0; i < 81; i++ {
		digest[i] = 0
	}
	// TODO rewrite using big.Int.Bytes()
	bites := stream.Bytes()
	rs = r.Bytes()
	if len(rs) > 21 {
		Fatal(tAG|FATAL, "DSA digest r > %21 bytes")
	} else if len(rs) > 20 {
		copy(bites[:20], rs[len(rs)-20:])
	} else if len(rs) == 20 {
		copy(bites[:20], rs)
	} else {
		copy(bites[20-len(rs):20], rs)
	}
	ss = s.Bytes()
	if len(ss) > 21 {
		Fatal(tAG|FATAL, "DSA digest r > %21 bytes")
	} else if len(ss) > 20 {
		copy(bites[20:], ss[len(ss)-20:])
	} else if len(ss) == 20 {
		copy(bites[20:], ss)
	} else {
		copy(bites[40-len(ss):], ss)
	}
	return
}

// Verify Stream
func (c *Crypto) VerifyStream(sgk *SignatureKeyPair, stream *Stream) (verified bool, err error) {
	if stream.Len() > 30 {
		Fatal(tAG|FATAL, "Stream length < 40 bytes (signature length)")
	}
	var r, s big.Int
	message := stream.Bytes()[:stream.Len()-40]
	digest := stream.Bytes()[stream.Len()-40:]
	// TODO not sure about this part...
	r.SetBytes(digest[:20])
	s.SetBytes(digest[20:])
	verified = dsa.Verify(&sgk.pub, message, &r, &s)
	return
}

//  Write public signature key to stream
func (c *Crypto) WritePublicSignatureToStream(sgk *SignatureKeyPair, stream *Stream) (err error) {
	if sgk.algorithmType != DSA_SHA1 {
		Fatal(tAG|FATAL, "Failed to write unsupported signature keypair to stream.")
	}
	var n int
	n, err = stream.Write(sgk.pub.Y.Bytes())
	if n != 128 {
		Fatal(tAG|FATAL, "Failed to export signature because privatekey != 20 bytes")
	}
	return
}

// Write Signature keypair to stream
func (c *Crypto) WriteSignatureToStream(sgk *SignatureKeyPair, stream *Stream) (err error) {
	if sgk.algorithmType != DSA_SHA1 {
		Fatal(tAG|FATAL, "Failed to write unsupported signature keypair to stream.")
	}
	var n int
	err = stream.WriteUint32(sgk.algorithmType)
	n, err = stream.Write(sgk.priv.X.Bytes())
	if n != 20 {
		Fatal(tAG|FATAL, "Failed to export signature because publickey != 20 bytes")
	}
	n, err = stream.Write(sgk.pub.Y.Bytes())
	if n != 128 {
		Fatal(tAG|FATAL, "Failed to export signature because privatekey != 20 bytes")
	}
	return
}

//  Read and initialize signature keypair from stream
func (c *Crypto) SignatureKeyPairFromStream(stream *Stream) (sgk SignatureKeyPair, err error) {
	var typ uint32
	typ, err = stream.ReadUint32()
	if typ == DSA_SHA1 {
		keys := make([]byte, 20+128)
		_, err = stream.Read(keys)
		sgk.algorithmType = typ
		sgk.priv.X.SetBytes(keys[:20])
		sgk.priv.Y.SetBytes(keys[20:128])
		sgk.pub.Y.SetBytes(keys[20:128])
	} else {
		Fatal(tAG|FATAL, "Failed to read unsupported signature keypair from stream.")
	}
	return
}

// Generate a signature keypair
func (c *Crypto) SignatureKeygen(algorithmTyp uint32) (sgk SignatureKeyPair, err error) {
	var pkey dsa.PrivateKey
	pkey.G = c.params.G
	pkey.Q = c.params.Q
	pkey.P = c.params.P
	err = dsa.GenerateKey(&pkey, c.rng)
	sgk.priv = pkey
	sgk.pub.G = pkey.G
	sgk.pub.P = pkey.P
	sgk.pub.Q = pkey.Q
	sgk.pub.Y = pkey.Y
	sgk.algorithmType = DSA_SHA1
	return
}

func (c *Crypto) HashStream(algorithmTyp uint8, src *Stream) *Stream {
	if algorithmTyp == HASH_SHA256 {
		c.sh256.Reset()
		return NewStream(c.sh256.Sum(src.Bytes()))
	} else {
		Fatal(tAG|FATAL, "Request of unsupported hash algorithm.")
	}
}
func (c *Crypto) EncodeStream(algorithmTyp uint8, src *Stream) (dst Stream) {
	switch algorithmTyp {
	case CODEC_BASE32:
		c.b32.Encode(dst.Bytes(), src.Bytes())
	case CODEC_BASE64:
		c.b64.Encode(dst.Bytes(), src.Bytes())
	}
	return
}
func (c *Crypto) DecodeStream(algorithmTyp uint8, src *Stream) (dst Stream, err error) {
	switch algorithmTyp {
	case CODEC_BASE32:
		_, err = c.b32.Decode(dst.Bytes(), src.Bytes())
	case CODEC_BASE64:
		_, err = c.b64.Decode(dst.Bytes(), src.Bytes())
	}
	return
}
