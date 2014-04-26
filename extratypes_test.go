// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package edn

import (
	. "gopkg.in/check.v1"
)

type ExtraTypesTests struct{}

func init() { Suite(&ExtraTypesTests{}) }

func (*ExtraTypesTests) TestSet(c *C) {
	set := Set{}
	c.Check(set.Has(nil), Equals, false)
	c.Check(set.Has(42), Equals, false)
	c.Check(set.Add("hello", 3.14, true, false, nil), DeepEquals, set)
	for _, k := range []interface{}{"hello", 3.14, true, false, nil} {
		c.Check(set.Has(k), Equals, true)
	}
}

func (*ExtraTypesTests) TestK(c *C) {
	c.Check(string(K("abc")), Equals, "abc")
	c.Check(string(K(":foo/abc")), Equals, ":foo/abc")
}

func (*ExtraTypesTests) TestKMap(c *C) {
	if b, err := Marshal(KMap{"foo": 123, "bar": true}); err == nil {
		c.Assert(string(b), Equals, "{:foo 123, :bar true}")
	} else {
		c.Fatal(err)
	}
}
