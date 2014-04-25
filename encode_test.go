// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package edn

import (
	"code.google.com/p/go-uuid/uuid"
	"container/list"
	"fmt"
	. "gopkg.in/check.v1"
	"time"
)

type EncodeTests struct{}

func init() { Suite(&EncodeTests{}) }

type vector []interface{}

type ednMap map[interface{}]interface{}

type pair struct {
	input interface{}
	want  string
}

func checkMarshal(c *C, pairs ...pair) {
	for _, p := range pairs {
		if b, err := Marshal(p.input); err == nil {
			c.Check(string(b), Equals, p.want, Commentf("%v didn't marshal to %q", p.input, p.want))
		} else {
			c.Error(err)
		}
	}
}

func (*EncodeTests) TestUnsupportedType(c *C) {
	_, err := Marshal(make(chan int))
	c.Log(err.(*UnsupportedTypeError))
}

func (*EncodeTests) TestPrimitives(c *C) {
	anInt := int(33)
	ptrToInt := &anInt
	utc, _ := time.LoadLocation("UTC")
	aTime := time.Date(2014, 3, 14, 15, 59, 59, 123456789, utc)
	aTimePtr := &aTime
	var nilTimePtr *time.Time
	aUuid := uuid.Parse("7594599c-2df6-412f-8ca0-8ef31448d923")
	aUuidPtr := &aUuid
	var nilUuidPtr *uuid.UUID
	checkMarshal(
		c,
		// literal nil
		pair{nil, "nil"},
		// reflect.Bool
		pair{false, "false"},
		pair{true, "true"},
		// reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64
		pair{int(1), "1"},
		pair{int8(127), "127"},
		pair{int16(-32768), "-32768"},
		pair{int32(2000000000), "2000000000"},
		pair{int64(-9000000000000000000), "-9000000000000000000"},
		// reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr
		pair{uint(77), "77"},
		pair{uint8(255), "255"},
		pair{uint16(65535), "65535"},
		pair{uint32(4294967295), "4294967295"},
		pair{uint64(9223372036854775807), "9223372036854775807"},
		// reflect.Float32
		pair{float32(3.14), "3.14"},
		// reflect.Float64
		pair{float64(3.14E+250), "3.14e+250"},
		// reflect.String
		pair{`Russian for hello is "привет".`, `"Russian for hello is \"привет\"."`},
		// edn.Keyword
		pair{K("foo/bar"), ":foo/bar"},
		pair{K(":wow"), ":wow"},
		// edn.Symbol
		pair{S("foo/bar"), "foo/bar"},
		pair{S("wow"), "wow"},
		// reflect.Ptr
		pair{ptrToInt, "33"},
		// time.Time
		pair{aTime, `#inst "2014-03-14T15:59:59.123456789Z"`},
		pair{aTimePtr, `#inst "2014-03-14T15:59:59.123456789Z"`},
		pair{nilTimePtr, "nil"},
		// uuid.UUID
		pair{aUuid, `#uuid "7594599c-2df6-412f-8ca0-8ef31448d923"`},
		pair{aUuidPtr, `#uuid "7594599c-2df6-412f-8ca0-8ef31448d923"`},
		pair{nilUuidPtr, "nil"},
	)
}

func (*EncodeTests) TestMaps(c *C) {
	checkMarshal(
		c,
		pair{map[int]string(nil), "{}"},
		pair{map[byte]float32{}, "{}"},
		pair{map[byte]float32{65: 3.14}, "{65 3.14}"},
		pair{
			ednMap{
				"x": -42,
				K("y"): ednMap{
					1:   S("foo"),
					nil: map[bool]float32{true: 0.1, false: 3.14},
				},
			}, `{"x" -42, :y {1 foo, nil {true 0.1, false 3.14}}}`,
		},
	)
}

func (*EncodeTests) TestSlicesAndArrays(c *C) {
	checkMarshal(
		c,
		// reflect.Slice
		pair{[]string(nil), "[]"},
		pair{[]string{}, "[]"},
		pair{[]int{-1111}, "[-1111]"},
		pair{[]vector{vector{-1, vector{0}, ednMap{K("answer"): 42}}}, "[[-1 [0] {:answer 42}]]"},
		pair{[]string{"a", "b", "ц"}, `["a" "b" "ц"]`},
		pair{[]byte("any + old & data"), `#base64 "YW55ICsgb2xkICYgZGF0YQ=="`},
		// reflect.Array
		pair{[4]interface{}{"a", vector{nil, nil}, 3.14E-33, 5}, `["a" [nil nil] 3.14e-33 5]`},
		pair{[1]int16{32767}, "[32767]"},
		pair{[3]byte{65, 66, 67}, "[65 66 67]"},
	)
}

func (*EncodeTests) TestSets(c *C) {
	checkMarshal(
		c,
		pair{Set(nil), "#{}"},
		pair{Set{}, "#{}"},
		pair{Set{}.Add(2, K(":yolo"), 2, -3.14, "yo"), `#{2 :yolo -3.14 "yo"}`},
	)
}

func (*EncodeTests) TestLists(c *C) {
	var nilPtr *list.List
	l1 := list.List{}
	l1.PushBack(1)
	l1.PushBack(K("two"))
	l2 := list.New() // this is *list.List
	l2.PushFront("a")
	l2.PushFront(Set{}.Add("b"))
	l2.PushFront(S("c"))
	l2.PushFront(vector{"d", "e"})
	l2.PushFront(K("f"))
	checkMarshal(
		c,
		pair{nilPtr, "nil"},
		pair{list.List{}, "()"},
		pair{l1, "(1 :two)"},
		pair{l2, `(:f ["d" "e"] c #{"b"} "a")`},
	)
}

type coolness struct {
	yes bool
}

func (cool coolness) MarshalText() (data []byte, err error) {
	data = []byte(fmt.Sprintf("cool=%v", cool.yes))
	return
}

func (*EncodeTests) TestCustomTextMarshal(c *C) {
	checkMarshal(c, pair{coolness{true}, `"cool=true"`})
}
