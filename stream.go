// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package edn

import (
	"io"
)

var encodeStatePool = make(chan *encodeState, 8)

func newEncodeState() *encodeState {
	select {
	case e := <-encodeStatePool:
		e.Reset()
		return e
	default:
		return new(encodeState)
	}
}

func putEncodeState(e *encodeState) {
	select {
	case encodeStatePool <- e:
	default:
	}
}

// An Encoder writes EDN objects to an output stream.
type Encoder struct {
	w   io.Writer
	err error
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes the EDN encoding of v to the stream.
//
// See the documentation for Marshal for details about the
// conversion of Go values to EDN.
func (enc *Encoder) Encode(v interface{}) error {
	if enc.err != nil {
		return enc.err
	}
	e := newEncodeState()
	err := e.marshal(v)
	if err != nil {
		return err
	}

	// Terminate each value with a newline.
	// This makes the output look a little nicer
	// when debugging, and some kind of space
	// is required if the encoded value was a number,
	// so that the reader knows there aren't more
	// digits coming.
	e.WriteByte('\n')

	if _, err = enc.w.Write(e.Bytes()); err != nil {
		enc.err = err
	}
	putEncodeState(e)
	return err
}
