package codec

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"io"
)

type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

func Encode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func Decode(data []byte, to interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(to)
}
func Uint32(b []byte) uint32 {
	_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}
func ReadUint32(r io.Reader) (uint32, error) {
	var bytes = make([]byte, 4)
	if _, err := io.ReadFull(r, bytes); err != nil {
		return 0, err
	}
	return Uint32(bytes), nil
}
func (c *GobCodec) ReadHeader(h *Header) error {
	bufLen, _ := ReadUint32(c.conn)
	buf := make([]byte, bufLen)
	_, err := io.ReadFull(c.conn, buf)
	//fmt.Println("head:", len(buf), buf)
	if err != nil {
		return err
	}
	Decode(buf, h)
	return nil
}

func (c *GobCodec) ReadBody(body interface{}) error {
	bufLen, _ := ReadUint32(c.conn)
	buf := make([]byte, bufLen)
	_, err := io.ReadFull(c.conn, buf)
	//fmt.Println("body:", len(buf), buf)
	if err != nil {
		return err
	}
	Decode(buf, body)
	return nil
}
func WriteUint32(w io.Writer, val uint32) error {
	buf := make([]byte, 4)
	PutUint32(buf, val)
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return nil
}
func PutUint32(b []byte, v uint32) {
	_ = b[3] // early bounds check to guarantee safety of writes below
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}
func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	byt, _ := Encode(h)
	buflen := len(byt)
	WriteUint32(c.conn, uint32(buflen))
	c.conn.Write(byt)
	byt, _ = Encode(body)
	buflen = len(byt)
	WriteUint32(c.conn, uint32(buflen))
	c.conn.Write(byt)
	return
}

func (c *GobCodec) Close() error {
	return c.conn.Close()
}
