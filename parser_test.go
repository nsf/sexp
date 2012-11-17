package sexp

import (
	"testing"
	"strings"
	"fmt"
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
)`[1:]

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

func test_file(ctx *SourceContext, name, content string, t *testing.T) {
	root, err := Parse(strings.NewReader(content), name, -1, ctx)
	if err != nil {
		t.Fatal(err)
	}

	root = root.Children
	for root != nil {
		print_ast(root, 0)
		root = root.Next
	}
}

func TestParser(t *testing.T) {
	var ctx SourceContext
	test_file(&ctx, "palindrome.scm", palindrome, t)
	test_file(&ctx, "config.sexp", config, t)
	lengths := [][2]int{
		{ctx.files[0].length, len(palindrome)},
		{ctx.files[1].length, len(config)},
	}
	for _, l := range lengths {
		if l[0] != l[1] {
			t.Errorf("lengths should match, got: %d != %d", l[0], l[1])
		}
	}
}
