// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package edn

import (
	"reflect"
)

type Set map[interface{}]bool

var setType = reflect.TypeOf(Set(nil))

func (set Set) Add(keys ...interface{}) Set {
	for _, k := range keys {
		set[k] = true
	}
	return set
}

func (set Set) Discard(k interface{}) {
	delete(set, k)
}

func (set Set) Has(k interface{}) (ret bool) {
	_, ret = set[k]
	return
}

type Keyword string

var keywordType = reflect.TypeOf(Keyword(""))

func K(k string) Keyword {
	return Keyword(k)
}

type Symbol string

var symbolType = reflect.TypeOf(Symbol(""))

func S(s string) Symbol {
	return Symbol(s)
}

// KMap is useful for generating EDN maps with Keywords as keys.
// For example: Marshal(KMap{"foo": 45, "bar": 3.14}) => {:foo 45, :bar 3.14}
type KMap map[string]interface{}

var keywordMapType = reflect.TypeOf(KMap{})
