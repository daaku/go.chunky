// Package chunky provides a writer that writes chunks of up to the configured
// size to it's underlying writer. It is very useful when you want to write UDP
// packets of up to or smaller than a certain size, while also ensuring the
// writes are not split across client defined semantic.
package chunky

import (
	"bytes"
	"errors"
	"io"
)

var (
	errBiggerThanMaxLen = errors.New("chunky: chunk was bigger MaxWriteLength")
	errUnexpectedLen    = errors.New("chunky: write returned unexpected length")
	errFlushBeforeMark  = errors.New("chunky: Flush called before Mark")
)

// Writer provides the chunky writer functionality that allows for aggregating
// chunks as best possible while preventing splitting of chunks. This Writer is
// NOT safe for concurrent use.
type Writer struct {
	Writer         io.Writer
	MaxWriteLength int
	mark           int
	two            bool
	buf1           bytes.Buffer
	buf2           bytes.Buffer
}

// Buffers data for future writes. Writes do not happen until Mark or Flush is
// called.
func (w *Writer) Write(d []byte) (int, error) {
	var buf *bytes.Buffer
	if w.two {
		buf = &w.buf2
	} else {
		buf = &w.buf1
	}

	n, err := buf.Write(d)
	if buf.Len()-w.mark > w.MaxWriteLength {
		return 0, errBiggerThanMaxLen
	}
	return n, err
}

// Flushes the pending data if any.
func (w *Writer) Flush() error {
	var pri *bytes.Buffer
	if w.two {
		pri = &w.buf2
	} else {
		pri = &w.buf1
	}

	prilen := pri.Len()
	if w.mark != prilen {
		return errFlushBeforeMark
	}

	contents := pri.Bytes()
	n, err := w.Writer.Write(contents[:prilen])
	if err != nil {
		return err
	}
	if n != w.mark {
		return errUnexpectedLen
	}
	pri.Reset()
	w.mark = 0
	return nil
}

// Marks the current point as a safe termination point and writes pending data
// if necessary. If the length of the data from the previous mark to this one
// is larger than the MaxWriteLength, it is considered an error.
func (w *Writer) Mark() error {
	var pri, sec *bytes.Buffer
	if w.two {
		pri = &w.buf2
		sec = &w.buf1
	} else {
		pri = &w.buf1
		sec = &w.buf2
	}

	prilen := pri.Len()
	if prilen > w.MaxWriteLength {
		// we need to flush upto the previous mark and swith buffers
		contents := pri.Bytes()
		n, err := w.Writer.Write(contents[:w.mark])
		if err != nil {
			return err
		}
		if n != w.mark {
			return errUnexpectedLen
		}
		n, err = sec.Write(contents[w.mark:])
		if err != nil {
			return err
		}
		if n != prilen-w.mark {
			return errUnexpectedLen
		}
		w.mark = n
		pri.Reset()
		w.two = !w.two
	} else {
		// we can move the mark and delay writing
		w.mark = prilen
	}

	return nil
}
