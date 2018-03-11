package go_i2cp

import (
	"bytes"
	"encoding/binary"
	"os"
)

type Stream struct {
	*bytes.Buffer
}

func NewStream(buf []byte) (s *Stream) {
	return &Stream{bytes.NewBuffer(buf)}
}
func (s *Stream) ReadUint16() (r uint16, err error) {
	bts := make([]byte, 2)
	_, err = s.Read(bts)
	r = binary.LittleEndian.Uint16(bts)
	return
}
func (s *Stream) ReadUint32() (r uint32, err error) {
	bts := make([]byte, 4)
	_, err = s.Read(bts)
	r = binary.LittleEndian.Uint32(bts)
	return
}
func (s *Stream) ReadUint64() (r uint64, err error) {
	bts := make([]byte, 8)
	_, err = s.Read(bts)
	r = binary.LittleEndian.Uint64(bts)
	return
}

func (s *Stream) WriteUint16(i uint16) (err error) {
	bts := make([]byte, 2)
	binary.LittleEndian.PutUint16(bts, i)
	_, err = s.Write(bts)
	return
}
func (s *Stream) WriteUint32(i uint32) (err error) {
	bts := make([]byte, 4)
	binary.LittleEndian.PutUint32(bts, i)
	_, err = s.Write(bts)
	return
}
func (s *Stream) WriteUint64(i uint64) (err error) {
	bts := make([]byte, 8)
	binary.LittleEndian.PutUint64(bts, i)
	_, err = s.Write(bts)
	return
}
func (s *Stream) loadFile(f *os.File) (err error) {
	_, err = f.Read(s.Bytes())
	return
}

func (s *Stream) ChLen(len int) {
	byt := s.Bytes()
	byt = byt[:len]
}

/*type Stream struct {
	data []byte
	size uint32
	p uint32
	end uint32
}

func (s *Stream) Init(len uint32) {
	data := make([]byte, len)
	s.data = data
	s.size = len
	s.Reset()
}

func (s *Stream) Reset() {
	s.end = s.size - 1
	s.p = 0
}

func (s *Stream) Seek(a uint32) {
	s.p = a
}
func (s *Stream) Advance() {
	s.p += 1
}
func (s *Stream) Tell() uint32 { return s.p }
func (s *Stream) MarkEnd() { s.end = s.p}
func (s *Stream) Eof() bool { return s.end < s.p}
func (s *Stream) Debug() {fmt.Printf("STREAM: data %p size %d p %d end %d", s.data, s.size, s.p, s.end)}
func (s *Stream) Check(len uint32) {
	if (s.p + len) > s.size {
		s.Debug()
		// TODO better error message
		os.Exit(2)
	}
}
func (s *Stream) Dump(file os.File) {
	file.Write(s.data)
	defer file.Close()
}
func (s *Stream) Skip(n uint32) {
	s.Check(n)
	s.p += n
}
func (s *Stream) ReadUint8() uint8 {
	s.Check(1)
	defer s.Advance()
	return s.data[s.p]
}
func (s *Stream) ReadUint8p(len uint32) []uint8 {
	s.Check(len)
	defer s.Skip(len)
	return s.data[s.p:len]
}
*/
