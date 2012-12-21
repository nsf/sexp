package sexp

import (
	"strings"
	"testing"
	"reflect"
	"regexp"
	"errors"
)

func must_contain(t *testing.T, err, what string) {
	re := regexp.MustCompile(what)
	if !re.MatchString(err) {
		t.Errorf(`expected: "%s", got: "%s"`, what, err)
	} else {
		t.Logf(`ok: %s`, err)
	}
}

func error_must_contain(t *testing.T, err error, what string) {
	if err == nil {
		t.Errorf("non-nil error expected")
		return
	}
	must_contain(t, err.Error(), what)
}

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

// always fails
type neversmiley string
func (s *neversmiley) UnmarshalSexp(n *Node) error {
	return NewUnmarshalError(n, reflect.TypeOf(s), ":-( Y U NO HAPPY?")
}
type neversmiley2 string
func (s *neversmiley2) UnmarshalSexp(n *Node) error {
	return errors.New("inevitable failure")
}

func TestUnmarshal(t *testing.T) {
	var a [3]int8
	test_unmarshal(t, "5 10 -15", &a)
	t.Logf("%d %d %d", a[0], a[1], a[2])

	var b [3]uint16
	test_unmarshal(t, "1024 750 300", &b)
	t.Logf("%d %d %d", b[0], b[1], b[2])

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

func test_unmarshal_error(t *testing.T, source, what string, args ...interface{}) {
	ast, err := Parse(strings.NewReader(source), "", -1, nil)
	if err != nil {
		t.Error(err)
	}
	err = ast.Unmarshal(args...)
	error_must_contain(t, err, what)
}

func TestUnmarshalErrors(t *testing.T) {
	var (
		a [3]uint
		b neversmiley
		c [1]neversmiley2
	)


	expect_panic(func() {
		test_unmarshal_error(t, "1 2 3", "", a)
	}, func(v interface{}) {
		if s, ok := v.(string); ok {
			must_contain(t, s, "Node.Unmarshal expects a non-nil pointer")
		} else {
			t.Errorf("unexpected panic: %s", v)
		}
	})
	test_unmarshal_error(t, "123", "Y U NO HAPPY", &b)
	test_unmarshal_error(t, "123", "inevitable failure", &c)
}
