// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package edn

import (
	"bytes"
	. "gopkg.in/check.v1"
	str "strings"
	"io/ioutil"
	"testing"
)

type StreamTests struct{}

func init() { Suite(&StreamTests{}) }

// Test values for the stream test.
// One of each EDN kind.
var streamTest = []interface{}{
	0.1,
	"hello",
	42,
	nil,
	Set{}.Add(1, 2.0, 3.14),
	true,
	false,
	[]interface{}{"a", "b", "c"},
	map[string]interface{}{"K": "Kelvin", "ß": "long s"},
	3.14, // another value to make sure something can follow map
}

var streamEncoded = `0.1
"hello"
42
nil
#{1 2 3.14}
true
false
["a" "b" "c"]
{"K" "Kelvin", "ß" "long s"}
3.14
`

func (*StreamTests) TestEncoder(c *C) {
	for i := 1; i <= len(streamTest); i++ {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)
		for j, v := range streamTest[0:i] {
			if err := enc.Encode(v); err != nil {
				c.Fatalf("encode #%d: %v", j, err)
			}
		}
		written := str.TrimRight(buf.String(), "\n")
		c.Check(str.Split(written, "\n"), DeepEquals, str.Split(streamEncoded, "\n")[0:i])
	}
}

func BenchmarkEncoderEncode(b *testing.B) {
	b.ReportAllocs()
	type T struct {
		X, Y string
	}
	v := Set{}.Add("foo", "bar")
	for i := 0; i < b.N; i++ {
		if err := NewEncoder(ioutil.Discard).Encode(&v); err != nil {
			b.Fatal(err)
		}
	}
}
