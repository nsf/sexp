package sexp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Convinience shortcut function for loading and unmarshaling an S-exp file
// into an arbitrary type.
func Load(filename string, out interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	ast, err := Parse(f, "", -1, nil)
	if err != nil {
		return err
	}
	return ast.Unmarshal(out)
}

// Parse an S-exp stream from a given io.Reader.
//
// Filename is used for informative purposes only. Length must reflect the length
// of a stream or -1 if unknown. Source context is optional as well, it will be
// used to encode source location information. If no source context was provided,
// the one will be created in-place, meaning you will not be able to decode
// source locations. Returns the root node of an AST tree if there were no errors.
func Parse(r io.Reader, filename string, length int, ctx *SourceContext) (*Node, error) {
	if ctx == nil {
		ctx = &SourceContext{}
	}
	f := ctx.AddFile(filename, length)
	return ParseFile(r, f)
}

// Same as Parse, except it takes source file created by SourceContext.AddFile
// directly. It's here for advanced purposes such as parallel parsing. You can
// add multiple files to the source context at once and then parse these files
// in parallel. However, keep in mind that all the lengths of the streams
// must be known ahead of time.
func ParseFile(r io.Reader, f *SourceFile) (*Node, error) {
	var p parser

	// avoid unnecessary double buffering
	if br, ok := r.(*bufio.Reader); ok {
		p.r = br
	} else {
		p.r = bufio.NewReader(r)
	}

	p.f = f
	p.line = 1
	p.last_seq = seq{offset: -1}
	p.expect_eof = true
	return p.parse()
}

var seq_delims = map[rune]rune{
	'(': ')',
	'`': '`',
	'"': '"',
}

func is_hex(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}

func is_space(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func is_delimiter(r rune) bool {
	return is_space(r) || r == ')' || r == ';' || r == 0
}

type seq struct {
	offset int
	rune   rune
}

type delim_state struct {
	last_seq   seq
	expect_eof bool
}

type parser struct {
	r      *bufio.Reader
	f      *SourceFile
	buf    bytes.Buffer
	line   int
	offset int
	cur    rune
	curlen int
	delim_state
}

func (p *parser) advance_delim_state() delim_state {
	s := p.delim_state
	p.last_seq = seq{p.offset, p.cur}
	p.expect_eof = false
	return s
}

func (p *parser) restore_delim_state(s delim_state) {
	p.delim_state = s
}

func (p *parser) error(loc SourceLoc, format string, args ...interface{}) {
	panic(&Error{
		Location: loc,
		message:  fmt.Sprintf(format, args...),
	})
}

func (p *parser) next() {
	p.offset += p.curlen
	r, s, err := p.r.ReadRune()
	if err != nil {
		if err == io.EOF {
			if p.expect_eof {
				p.cur = 0
				p.curlen = 0
				return
			}
			p.error(p.f.Encode(p.last_seq.offset),
				"missing matching sequence delimiter '%c'",
				seq_delims[p.last_seq.rune])
		}
		p.error(p.f.Encode(p.offset),
			"unexpected read error: %s", err)
	}

	p.cur = r
	p.curlen = s
	if r == '\n' {
		p.f.AddLine(p.offset + p.curlen)
	}
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

func (p *parser) skip_comment() {
	for {
		// there was an EOF, return
		if p.cur == 0 {
			return
		}

		// read until '\n'
		if p.cur != '\n' {
			p.next()
		} else {
			// skip '\n' and return
			p.next()
			return
		}
	}
	panic("unreachable")
}

func (p *parser) parse_node() *Node {
again:
	// the convention is that this function is called on a non-space `p.cur`
	switch p.cur {
	case ')':
		return nil
	case '(':
		return p.parse_list()
	case '"':
		return p.parse_string()
	case '`':
		return p.parse_raw_string()
	case ';':
		// skip comment
		p.skip_comment()
		p.skip_spaces()
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
	save := p.advance_delim_state()

	head := &Node{Location: loc}
	p.next() // skip opening '('

	var lastchild *Node
	for {
		p.skip_spaces()
		if p.cur == ')' {
			// skip enclosing ')', but it could be EOF also
			p.restore_delim_state(save)
			p.next()
			return head
		}

		node := p.parse_node()
		if node == nil {
			continue
		}
		if head.Children == nil {
			head.Children = node
		} else {
			lastchild.Next = node
		}
		lastchild = node
	}
	panic("unreachable")
}

func (p *parser) parse_esc_seq() {
	loc := p.f.Encode(p.offset)

	p.next() // skip '\\'
	switch p.cur {
	case 'a':
		p.next()
		p.buf.WriteByte('\a')
	case 'b':
		p.next()
		p.buf.WriteByte('\b')
	case 'f':
		p.next()
		p.buf.WriteByte('\f')
	case 'n':
		p.next()
		p.buf.WriteByte('\n')
	case 'r':
		p.next()
		p.buf.WriteByte('\r')
	case 't':
		p.next()
		p.buf.WriteByte('\t')
	case 'v':
		p.next()
		p.buf.WriteByte('\v')
	case '\\':
		p.next()
		p.buf.WriteByte('\\')
	case '"':
		p.next()
		p.buf.WriteByte('"')
	default:
		switch p.cur {
		case 'x':
			p.next() // skip 'x'
			p.parse_hex_rune(2)
		case 'u':
			p.next() // skip 'u'
			p.parse_hex_rune(4)
		case 'U':
			p.next() // skip 'U'
			p.parse_hex_rune(8)
		default:
			p.error(loc, `unrecognized escape sequence within '"' string`)
		}
	}
}

func (p *parser) parse_hex_rune(n int) {
	if n > 8 {
		panic("hex rune is too large")
	}

	var hex [8]byte
	p.next_hex(hex[:n])
	r, err := strconv.ParseUint(string(hex[:n]), 16, n*4) // 4 bits per hex digit
	panic_if_error(err)
	if n == 2 {
		p.buf.WriteByte(byte(r))
	} else {
		p.buf.WriteRune(rune(r))
	}
}

func (p *parser) next_hex(s []byte) {
	for i, n := 0, len(s); i < n; i++ {
		if !is_hex(p.cur) {
			loc := p.f.Encode(p.offset)
			p.error(loc, `'%c' is not a hex digit`, p.cur)
		}
		s[i] = byte(p.cur)
		p.next()
	}
}

func (p *parser) parse_string() *Node {
	loc := p.f.Encode(p.offset)
	save := p.advance_delim_state()

	p.next() // skip opening '"'
	for {
		switch p.cur {
		case '\n':
			p.error(loc, `newline is not allowed within '"' strings`)
		case '\\':
			p.parse_esc_seq()
		case '"':
			node := &Node{
				Location: loc,
				Value:    p.buf.String(),
			}
			p.buf.Reset()

			// consume enclosing '"', could be EOF
			p.restore_delim_state(save)
			p.next()
			return node
		default:
			p.buf.WriteRune(p.cur)
			p.next()
		}
	}
	panic("unreachable")
}

func (p *parser) parse_raw_string() *Node {
	loc := p.f.Encode(p.offset)
	save := p.advance_delim_state()

	p.next() // skip opening '`'
	for {
		if p.cur == '`' {
			node := &Node{
				Location: loc,
				Value:    p.buf.String(),
			}
			p.buf.Reset()
			// consume enclosing '`', could be EOF
			p.restore_delim_state(save)
			p.next()
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
				Value:    p.buf.String(),
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
		p.skip_spaces()
		node := p.parse_node()
		if node == nil {
			p.error(p.f.Encode(p.offset),
				"unexpected ')' at the top level")
		}
		if root.Children == nil {
			root.Children = node
		} else {
			lastchild.Next = node
		}
		lastchild = node
	}
	panic("unreachable")
}
