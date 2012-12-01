package sexp

import (
	"strings"
	"testing"
	"reflect"
)

func test_unmarshal_generic(t *testing.T, data string, f func(*Node, ...interface{}) error, v ...interface{}) {
	root, err := Parse(strings.NewReader(data), "", -1, nil)
	if err != nil {
		t.Error(err)
		return
	}

	err = f(root, v...)
	if err != nil {
		t.Error(err)
	}
}

func test_unmarshal(t *testing.T, data string, v ...interface{}) {
	test_unmarshal_generic(t, data, (*Node).Unmarshal, v...)
}

func test_unmarshal_children(t *testing.T, data string, v ...interface{}) {
	test_unmarshal_generic(t, data, (*Node).UnmarshalChildren, v...)
}


const countries = `
;; a list of arbitrary countries
(countries (
	Spain
	Russia ; I live here :-D
	Japan
	China
	England
	Germany
	France
	Sweden
	Iraq
	Iran
	Indonesia
	India
	USA
	Canada
	Brazil
))
`

// just to test Unmarshaler interface
type smiley string
func (s *smiley) UnmarshalSexp(n *Node) error {
	if !n.IsScalar() {
		return NewUnmarshalError(n, reflect.TypeOf(s),
			"scalar value required")
	}
	*s = smiley(n.Value + " :-D")
	return nil
}

func TestUnmarshal(t *testing.T) {
	var a [3]int8
	test_unmarshal(t, "5 10 -15", &a)
	t.Logf("%d %d %d", a[0], a[1], a[2])

	var m map[string][]string
	test_unmarshal(t, countries, &m)
	for _, country := range m["countries"] {
		t.Logf("%q", country)
	}

	var s []smiley
	test_unmarshal(t, `what if we try`, &s)
	for _, s := range s {
		t.Logf("%q", s)
	}
}
