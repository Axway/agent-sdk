package traceability

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
)

type bodyEncoder interface {
	bulkBodyEncoder
	Reader() io.Reader
	Marshal(doc interface{}) error
}

type bulkBodyEncoder interface {
	AddHeader(header map[string]string)
	Reset()
}

type jsonEncoder struct {
	buf *bytes.Buffer
}

type gzipEncoder struct {
	buf  *bytes.Buffer
	gzip *gzip.Writer
}

func newJSONEncoder(buf *bytes.Buffer) *jsonEncoder {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	return &jsonEncoder{buf}
}

func (b *jsonEncoder) Reset() {
	b.buf.Reset()
}

func (b *jsonEncoder) AddHeader(header map[string]string) {
	header["Content-Type"] = "application/json; charset=UTF-8"
}

func (b *jsonEncoder) Reader() io.Reader {
	return b.buf
}

func (b *jsonEncoder) Marshal(obj interface{}) error {
	b.Reset()
	enc := json.NewEncoder(b.buf)
	return enc.Encode(obj)
}

func newGzipEncoder(level int, buf *bytes.Buffer) (*gzipEncoder, error) {
	if buf == nil {
		buf = bytes.NewBuffer(nil)
	}
	w, err := gzip.NewWriterLevel(buf, level)
	if err != nil {
		return nil, err
	}

	return &gzipEncoder{buf, w}, nil
}

func (b *gzipEncoder) Reset() {
	b.buf.Reset()
	b.gzip.Reset(b.buf)
}

func (b *gzipEncoder) Reader() io.Reader {
	b.gzip.Close()
	return b.buf
}

func (b *gzipEncoder) AddHeader(header map[string]string) {
	header["Content-Type"] = "application/json; charset=UTF-8"
	header["Content-Encoding"] = "gzip"
}

func (b *gzipEncoder) Marshal(obj interface{}) error {
	b.Reset()
	enc := json.NewEncoder(b.gzip)
	err := enc.Encode(obj)
	return err
}
