package sexp

import (
	"io"
	"fmt"
	"bufio"
	"bytes"
)

func new_error(location SourceLoc, format string, args ...interface{}) *Error {
	return &Error{
		Location: location,
		message: fmt.Sprintf(format, args...),
	}
}

func is_space(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func is_delimiter(r rune) bool {
	return is_space(r) || r == ')'
}

type parser struct {
	r *bufio.Reader
	f *SourceFile
	buf bytes.Buffer
	line int
	offset int
	cur rune
}

func (p *parser) next() {
	r, s, err := p.r.ReadRune()
	if err != nil {
		if err == io.EOF {
			panic(new_error(p.f.Encode(p.offset),
				"unexpected EOF"))
		}
		panic(err)
	}

	p.offset += s
	if r == '\n' {
		p.f.AddLine(p.offset)
	}
	p.cur = r
}

func (p *parser) next_expect_eof() {
	r, s, err := p.r.ReadRune()
	if err != nil {
		if err != io.EOF {
			panic(err)
		}
		r, s = 0, 0
	}

	p.offset += s
	if r == '\n' {
		p.f.AddLine(p.offset)
	}
	p.cur = r
}

func (p *parser) skip_spaces() {
	for {
		if is_space(p.cur) {
			p.next()
		} else {
			return
		}
	}
	panic("unreachable")
}

func (p *parser) skip_spaces_expect_eof() {
	for {
		if is_space(p.cur) {
			p.next_expect_eof()
		} else {
			return
		}
	}
	panic("unreachable")
}

func (p *parser) skip_comment() {
	for {
		// read until '\n'
		if p.cur != '\n' {
			p.next_expect_eof()
		} else {
			p.next_expect_eof()
			return
		}
	}
	panic("unreachable")
}

func (p *parser) parse_node() *Node {
again:
	// the convention is that this function is called on a non-space `p.cur`
	switch p.cur {
	case '(':
		return p.parse_list()
	case '"':
		return p.parse_string()
	case '`':
		return p.parse_raw_string()
	case ';':
		// skip comment
		p.skip_comment()
		p.skip_spaces_expect_eof()
		goto again
	case 0:
		// delayed expected EOF
		panic(io.EOF)
	default:
		return p.parse_ident()
	}
	panic("unreachable")
}

func (p *parser) parse_list() *Node {
	loc := p.f.Encode(p.offset)

	head := &Node{Location: loc}
	p.next() // skip opening '('

	var lastchild *Node
	for {
		p.skip_spaces()
		if p.cur == ')' {
			// skip enclosing ')', but it could be EOF also
			p.next_expect_eof()
			return head
		}

		node := p.parse_node()
		if head.Children == nil {
			head.Children = node
		} else {
			lastchild.Next = node
		}
		lastchild = node
	}
	panic("unreachable")
}

func (p *parser) parse_string() *Node {
	loc := p.f.Encode(p.offset)
	prev := p.cur

	p.next() // skip opening '"'
	p.buf.WriteByte('"')
	for {
		switch p.cur {
		case '\n':
			panic(new_error(loc,
				"newline is not allowed within \"\" strings"))
		case '"':
			if prev != '\\' {
				p.buf.WriteByte('"')
				node := &Node{
					Location: loc,
					Value: p.buf.String(),
				}
				p.buf.Reset()
				// consume enclosing '"', could be EOF
				p.next_expect_eof()
				return node
			}
			fallthrough
		default:
			prev = p.cur
			p.buf.WriteRune(p.cur)
			p.next()
		}
	}
	panic("unreachable")
}

func (p *parser) parse_raw_string() *Node {
	loc := p.f.Encode(p.offset)
	p.next() // skip opening '`'
	p.buf.WriteByte('`')
	for {
		if p.cur == '`' {
			p.buf.WriteByte('`')
			node := &Node{
				Location: loc,
				Value: p.buf.String(),
			}
			p.buf.Reset()
			// consume enclosing '`', could be EOF
			p.next_expect_eof()
			return node
		} else {
			p.buf.WriteRune(p.cur)
			p.next()
		}
	}
	panic("unreachable")
}

func (p *parser) parse_ident() *Node {
	loc := p.f.Encode(p.offset)
	for {
		if is_delimiter(p.cur) {
			node := &Node{
				Location: loc,
				Value: p.buf.String(),
			}
			p.buf.Reset()
			return node
		} else {
			p.buf.WriteRune(p.cur)
			p.next()
		}
	}
	panic("unreachable")
}

func (p *parser) parse() (root *Node, err error) {
	defer func() {
		if e := recover(); e != nil {
			p.f.Finalize(p.offset)
			if e == io.EOF {
				return
			}
			if sexperr, ok := e.(*Error); ok {
				root = nil
				err = sexperr
				return
			}
			panic(e)
		}
	}()

	root = new(Node)
	p.next()

	// don't worry, will eventually panic with io.EOF :D
	var lastchild *Node
	for {
		p.skip_spaces_expect_eof()
		node := p.parse_node()
		if root.Children == nil {
			root.Children = node
		} else {
			lastchild.Next = node
		}
		lastchild = node
	}
	panic("unreachable")
}
