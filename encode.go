// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package edn implements encoding and decoding of EDN values as defined in
// https://github.com/edn-format/edn
//
// The mapping between EDN and Go values is described
// in the documentation for the Marshal and Unmarshal functions.
package edn

import (
	"bytes"
	"code.google.com/p/go-uuid/uuid"
	"container/list"
	"encoding"
	"encoding/base64"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Marshal returns the EDN encoding of v.
//
// Marshal traverses the value v recursively, using the following
// type-dependent default encodings:
// (see https://github.com/edn-format/edn)
func Marshal(v interface{}) ([]byte, error) {
	e := &encodeState{}
	err := e.marshal(v)
	if err != nil {
		return nil, err
	}
	return e.Bytes(), nil
}

// An UnsupportedTypeError is returned by Marshal when attempting
// to encode an unsupported value type.
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return "edn: unsupported type: " + e.Type.String()
}

type UnsupportedValueError struct {
	Value reflect.Value
	Str   string
}

func (e *UnsupportedValueError) Error() string {
	return "edn: unsupported value: " + e.Str
}

type MarshalerError struct {
	Type reflect.Type
	Err  error
}

func (e *MarshalerError) Error() string {
	return "edn: error calling MarshalEDN for type " + e.Type.String() + ": " + e.Err.Error()
}

// An encodeState encodes EDN into a bytes.Buffer.
type encodeState struct {
	bytes.Buffer // accumulated output
	scratch      [64]byte
}

func (e *encodeState) marshal(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			if s, ok := r.(string); ok {
				panic(s)
			}
			err = r.(error)
		}
	}()
	e.reflectValue(reflect.ValueOf(v))
	return nil
}

func (e *encodeState) error(err error) {
	panic(err)
}

func (e *encodeState) reflectValue(v reflect.Value) {
	valueEncoder(v)(e, v)
}

func (e *encodeState) string(s string) (int, error) {
	len0 := e.Len()
	e.WriteString(fmt.Sprintf("%#v", s))
	return e.Len() - len0, nil
}

func (e *encodeState) stringBytes(s []byte) (int, error) {
	len0 := e.Len()
	e.WriteString(fmt.Sprintf("%#v", string(s)))
	return e.Len() - len0, nil
}

type encoderFunc func(e *encodeState, v reflect.Value)

var encoderCache struct {
	sync.RWMutex
	m map[reflect.Type]encoderFunc
}

func valueEncoder(v reflect.Value) encoderFunc {
	if !v.IsValid() {
		return invalidValueEncoder
	}
	return typeEncoder(v.Type())
}

func typeEncoder(t reflect.Type) encoderFunc {
	encoderCache.RLock()
	f := encoderCache.m[t]
	encoderCache.RUnlock()
	if f != nil {
		return f
	}

	// To deal with recursive types, populate the map with an
	// indirect func before we build it. This type waits on the
	// real func (f) to be ready and then calls it.  This indirect
	// func is only used for recursive types.
	encoderCache.Lock()
	if encoderCache.m == nil {
		encoderCache.m = make(map[reflect.Type]encoderFunc)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	encoderCache.m[t] = func(e *encodeState, v reflect.Value) {
		wg.Wait()
		f(e, v)
	}
	encoderCache.Unlock()

	// Compute fields without lock.
	// Might duplicate effort but won't hold other computations back.
	f = newTypeEncoder(t, true)
	wg.Done()
	encoderCache.Lock()
	encoderCache.m[t] = f
	encoderCache.Unlock()
	return f
}

var (
	timeType          = reflect.TypeOf(time.Time{})
	textMarshalerType = reflect.TypeOf(new(encoding.TextMarshaler)).Elem()
	listType          = reflect.TypeOf(list.List{})
	uuidType          = reflect.TypeOf(uuid.UUID{})
)

// newTypeEncoder constructs an encoderFunc for a type.
// The returned encoder only checks CanAddr when allowAddr is true.
func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {
	// Special case for time.Time because it already implements
	// TextMarshaler which is not what we want as EDN.
	if t == timeType {
		return timeEncoder
	}
	if t.Kind() == reflect.Ptr && t.Elem() == timeType {
		return newPtrEncoder(t)
	}

	if t.Implements(textMarshalerType) {
		return textMarshalerEncoder
	}
	if t.Kind() != reflect.Ptr && allowAddr {
		if reflect.PtrTo(t).Implements(textMarshalerType) {
			return newCondAddrEncoder(addrTextMarshalerEncoder, newTypeEncoder(t, false))
		}
	}

	switch t.Kind() {
	case reflect.Bool:
		return boolEncoder
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intEncoder
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintEncoder
	case reflect.Float32:
		return float32Encoder
	case reflect.Float64:
		return float64Encoder
	case reflect.String:
		return stringEncoder
	case reflect.Interface:
		return interfaceEncoder
	case reflect.Map:
		return newMapEncoder(t)
	case reflect.Slice:
		return newSliceEncoder(t)
	case reflect.Array:
		return newArrayEncoder(t)
	case reflect.Ptr:
		return newPtrEncoder(t)
	default:
		if t == listType {
			return listEncoder
		}
		return unsupportedTypeEncoder
	}
}

func invalidValueEncoder(e *encodeState, v reflect.Value) {
	e.WriteString("nil")
}

func timeEncoder(e *encodeState, v reflect.Value) {
	t := v.Interface().(time.Time)
	e.WriteString("#inst ")
	if _, err := e.string(t.UTC().Format(time.RFC3339Nano)); err != nil {
		e.error(err)
	}
}

func textMarshalerEncoder(e *encodeState, v reflect.Value) {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		e.WriteString("nil")
		return
	}
	m := v.Interface().(encoding.TextMarshaler)
	b, err := m.MarshalText()
	if err == nil {
		_, err = e.stringBytes(b)
	}
	if err != nil {
		e.error(&MarshalerError{v.Type(), err})
	}
}

func addrTextMarshalerEncoder(e *encodeState, v reflect.Value) {
	va := v.Addr()
	if va.IsNil() {
		e.WriteString("nil")
		return
	}
	m := va.Interface().(encoding.TextMarshaler)
	b, err := m.MarshalText()
	if err == nil {
		_, err = e.stringBytes(b)
	}
	if err != nil {
		e.error(&MarshalerError{v.Type(), err})
	}
}

func boolEncoder(e *encodeState, v reflect.Value) {
	if v.Bool() {
		e.WriteString("true")
	} else {
		e.WriteString("false")
	}
}

func intEncoder(e *encodeState, v reflect.Value) {
	b := strconv.AppendInt(e.scratch[:0], v.Int(), 10)
	e.Write(b)
}

func uintEncoder(e *encodeState, v reflect.Value) {
	b := strconv.AppendUint(e.scratch[:0], v.Uint(), 10)
	e.Write(b)
}

type floatEncoder int // number of bits

func (bits floatEncoder) encode(e *encodeState, v reflect.Value) {
	f := v.Float()
	if math.IsInf(f, 0) || math.IsNaN(f) {
		e.error(&UnsupportedValueError{v, strconv.FormatFloat(f, 'g', -1, int(bits))})
	}
	b := strconv.AppendFloat(e.scratch[:0], f, 'g', -1, int(bits))
	e.Write(b)
}

var (
	float32Encoder = (floatEncoder(32)).encode
	float64Encoder = (floatEncoder(64)).encode
)

func stringEncoder(e *encodeState, v reflect.Value) {
	s := v.String()
	switch t := v.Type(); t {
	case symbolType, keywordType:
		if t == keywordType && !strings.HasPrefix(s, ":") {
			e.WriteByte(':')
		}
		e.WriteString(s)
		return
	default:
		e.string(s)
	}
}

func interfaceEncoder(e *encodeState, v reflect.Value) {
	if v.IsNil() {
		e.WriteString("nil")
		return
	}
	e.reflectValue(v.Elem())
}

func unsupportedTypeEncoder(e *encodeState, v reflect.Value) {
	e.error(&UnsupportedTypeError{v.Type()})
}

type mapEncoder struct {
	keyEnc  encoderFunc
	elemEnc encoderFunc
}

func (me *mapEncoder) encode(e *encodeState, v reflect.Value) {
	isSet := v.Type() == setType
	sep := ", "
	if isSet {
		e.WriteByte('#')
		sep = " "
	}
	if v.IsNil() {
		e.WriteString("{}")
		return
	}
	e.WriteByte('{')
	for i, k := range v.MapKeys() {
		if i > 0 {
			e.WriteString(sep)
		}
		me.keyEnc(e, k)
		if !isSet {
			e.WriteByte(' ')
			me.elemEnc(e, v.MapIndex(k))
		}
	}
	e.WriteByte('}')
}

func newMapEncoder(t reflect.Type) encoderFunc {
	me := &mapEncoder{typeEncoder(t.Key()), typeEncoder(t.Elem())}
	return me.encode
}

func encodeByteSlice(e *encodeState, v reflect.Value) {
	if v.IsNil() {
		e.WriteString("nil")
		return
	}
	s := v.Bytes()
	e.WriteString("#base64 ")
	e.WriteByte('"')
	if len(s) < 1024 {
		// for small buffers, using Encode directly is much faster.
		dst := make([]byte, base64.StdEncoding.EncodedLen(len(s)))
		base64.StdEncoding.Encode(dst, s)
		e.Write(dst)
	} else {
		// for large buffers, avoid unnecessary extra temporary
		// buffer space.
		enc := base64.NewEncoder(base64.StdEncoding, e)
		enc.Write(s)
		enc.Close()
	}
	e.WriteByte('"')
}

func encodeUuid(e *encodeState, v reflect.Value) {
	e.WriteString("#uuid ")
	e.string(uuid.UUID(v.Bytes()).String())
}

// sliceEncoder just wraps an arrayEncoder, checking to make sure the value isn't nil.
type sliceEncoder struct {
	arrayEnc encoderFunc
}

func (se *sliceEncoder) encode(e *encodeState, v reflect.Value) {
	if v.IsNil() {
		e.WriteString("[]")
		return
	}
	se.arrayEnc(e, v)
}

func newSliceEncoder(t reflect.Type) encoderFunc {
	// Byte slices get special treatment; arrays don't.
	if t.Elem().Kind() == reflect.Uint8 {
		if t == uuidType {
			return encodeUuid
		}
		return encodeByteSlice
	}
	enc := &sliceEncoder{newArrayEncoder(t)}
	return enc.encode
}

type arrayEncoder struct {
	elemEnc encoderFunc
}

func (ae *arrayEncoder) encode(e *encodeState, v reflect.Value) {
	e.WriteByte('[')
	n := v.Len()
	for i := 0; i < n; i++ {
		if i > 0 {
			e.WriteByte(' ')
		}
		ae.elemEnc(e, v.Index(i))
	}
	e.WriteByte(']')
}

func newArrayEncoder(t reflect.Type) encoderFunc {
	enc := &arrayEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type ptrEncoder struct {
	elemEnc encoderFunc
}

func (pe *ptrEncoder) encode(e *encodeState, v reflect.Value) {
	if v.IsNil() {
		e.WriteString("nil")
		return
	}
	pe.elemEnc(e, v.Elem())
}

func newPtrEncoder(t reflect.Type) encoderFunc {
	enc := &ptrEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type condAddrEncoder struct {
	canAddrEnc, elseEnc encoderFunc
}

func (ce *condAddrEncoder) encode(e *encodeState, v reflect.Value) {
	if v.CanAddr() {
		ce.canAddrEnc(e, v)
	} else {
		ce.elseEnc(e, v)
	}
}

// newCondAddrEncoder returns an encoder that checks whether its value
// CanAddr and delegates to canAddrEnc if so, else to elseEnc.
func newCondAddrEncoder(canAddrEnc, elseEnc encoderFunc) encoderFunc {
	enc := &condAddrEncoder{canAddrEnc: canAddrEnc, elseEnc: elseEnc}
	return enc.encode
}

func listEncoder(e *encodeState, v reflect.Value) {
	l := v.Interface().(list.List)
	i := 0
	e.WriteByte('(')
	for node := l.Front(); node != nil; node = node.Next() {
		if i > 0 {
			e.WriteByte(' ')
		}
		e.reflectValue(reflect.ValueOf(node.Value))
		i++
	}
	e.WriteByte(')')
}
