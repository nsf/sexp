package sexp

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"errors"
	"bytes"
	"bufio"
)

var palindrome = `
(define (palindrome? x)
  (define (check left right)
    (if (>= left right)
        #t
        (and (char=? (string-ref x left) (string-ref x right))
             (check (add1 left) (sub1 right)))))
  (check 0 (sub1 (string-length x))))

(let ((arg (car (command-line-arguments))))
  (display
   (string-append arg
    (if (palindrome? arg)
     " is a palindrome\n"
     " isn't a palindrome\n"))))`[1:]

var config = `
(namespace Gtk)
(version 3.0)
(blacklist
  (structs (
     StockItem
  ))
  (structdefs (
     ActionEntry
     RadioActionEntry
     ToggleActionEntry
  ))
  (functions (
     accelerator_parse_with_keycode
     binding_entry_add_signal_from_string
     binding_entry_add_signall
     binding_entry_remove
     binding_entry_skip
     binding_set_find
     paper_size_get_default
     paper_size_get_paper_sizes
     rc_property_parse_border
     rc_property_parse_color
     rc_property_parse_enum
     rc_property_parse_flags
     rc_property_parse_requisition
     print_run_page_setup_dialog
     print_run_page_setup_dialog_async
     init_with_args
     stock_add        ; implemented manually and renamed to StockAddItems (name clash)
     stock_lookup     ; implemented manually
     stock_add_static ; doesn't make sense
     rc_parse_color
     rc_parse_color_full
     rc_parse_priority
     rc_parse_state
     rc_find_pixmap_in_path
     stock_set_translate_func
     tree_row_reference_deleted
     tree_row_reference_inserted
  ))
) ; testing a comment at the end of file`[1:]

var empty = `
; empty file with a comment!`[1:]

func print_ast(n *Node, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Print(" ")
	}
	if n.IsList() {
		fmt.Printf("(%s\n", n.Value)
	} else {
		fmt.Println(n.Value)
	}
	child := n.Children
	for child != nil {
		print_ast(child, indent+1)
		child = child.Next
	}
}

func test_file(t *testing.T, ctx *SourceContext, name, content string) {
	root, err := Parse(strings.NewReader(content), name, -1, ctx)
	if err != nil {
		t.Error(err)
	}

	root, err = Parse(bufio.NewReader(strings.NewReader(content)), name, -1, ctx)
	if err != nil {
		t.Error(err)
	}

	_ = root
	/*
		root = root.Children
		for root != nil {
			print_ast(root, 0)
			root = root.Next
		}
	*/
}

func format_tree(buf *bytes.Buffer, root *Node) {
	if root.IsList() {
		buf.WriteString("(")
		c := root.Children
		for {
			format_tree(buf, c)
			if c.Next == nil {
				break
			} else {
				buf.WriteString(" ")
				c = c.Next
			}
		}
		buf.WriteString(")")
		return
	}

	fmt.Fprintf(buf, "%q", root.Value)
}

func format_siblings(buf *bytes.Buffer, n *Node) {
	for n != nil {
		format_tree(buf, n)
		if n.Next != nil {
			buf.WriteString(" ")
		}
		n = n.Next
	}
}

func test_tree(t *testing.T, source, gold string) {
	root, err := Parse(strings.NewReader(source), "", -1, nil)
	if err != nil {
		t.Error(err)
		return
	}
	var buf bytes.Buffer
	format_siblings(&buf, root.Children)
	src := buf.String()
	if gold != src {
		t.Errorf("%s != %s", src, gold)
	} else {
		t.Logf("%s == %s", source, gold)
	}
}

type fail_reader int
func (fail_reader) Read(_ []byte) (int, error) {
	return 0, errors.New("fail reader always fails")
}

func TestParser(t *testing.T) {
	var ctx SourceContext
	test_file(t, &ctx, "palindrome.scm", palindrome)
	test_file(t, &ctx, "config.sexp", config)
	test_file(t, &ctx, "empty.sexp", empty)

	// string interpreter
	test_tree(t, `"\a\b\f\n\r\t\v\\\""`, `"\a\b\f\n\r\t\v\\\""`)
	test_tree(t, `"\xFF"`, `"\xff"`)
	test_tree(t, `"\u0436\r"`, `"ж\r"`)
	test_tree(t, `"\U00101234\t\t"`, `"\U00101234\t\t"`)
	test_tree(t, `"\""`, `"\""`)

	// general
	test_tree(t, "()", `""`)
	test_tree(t, "(;comment\n)", `""`)
	test_tree(t, "123 ;comment\n123", `"123" "123"`)
	test_tree(t, "(() ())", `("" "")`)
	test_tree(t, "(123 456)", `("123" "456")`)
	test_tree(t, "123 ()  456; comment", `"123" "" "456"`)
	test_tree(t, `123 ()  "456; comment"`, `"123" "" "456; comment"`)
	test_tree(t, "1 (2 (3 (4 (5))))", `"1" ("2" ("3" ("4" ("5"))))`)
	test_tree(t, "`123` `456`", `"123" "456"`)
}

func TestParserErrors(t *testing.T) {
	// fail reader
	_, err := Parse(fail_reader(0), "", -1, nil)
	if err == nil {
		t.Fatal("error expected")
	}

	var ctx SourceContext
	test := func(source string) error {
		_, err := Parse(strings.NewReader(source), "test.txt", -1, &ctx)
		return err
	}
	must_contain := func(err error, what string) {
		if err == nil {
			t.Errorf("non-nil error expected")
			return
		}
		re := regexp.MustCompile(what)
		if !re.MatchString(err.Error()) {
			t.Errorf(`expected: "%s", got: "%s"`, what, err)
		} else {
			t.Logf(`ok: %s`, err)
		}
	}
	must_contain(test(`(1 2 3`), `missing.+\)`)
	must_contain(test(`"1 2 3`), `missing.+"`)
	must_contain(test("`1 2 3"), "missing.+`")
	must_contain(test("(`1 2 3`"), `missing.+\)`)
	must_contain(test("\"1 2 3\n\""), `newline is not allowed`)
	must_contain(test(`"\z"`), `unrecognized escape sequence`)
	must_contain(test(`"\x5J"`), `is not a hex digit`)
	must_contain(test(`)`), `unexpected '\)'`)
}
