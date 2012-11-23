package sexp

import (
	"bufio"
	"io"
)

// The main and only AST structure. All fields are self explanatory, however
// the way they are being formed needs explanation.
//
// A list node has empty value and non-nil children pointer, which is a
// nil-terminated list of children nodes.
//
// A scalar node has nil children pointer.
//
// Take a look at this example:
//
//   ((1 2) 3 4)
//
// will yield:
//
//   Node{Children:
//     Node{Children:
//       Node{Value: "1", Next:
//       Node{Value: "2"}}, Next:
//     Node{Value: "3", Next:
//     Node{Value: "4"}}}}
type Node struct {
	Location SourceLoc
	Value    string
	Children *Node
	Next     *Node
}

// Returns true if the node is a list (has children).
func (n *Node) IsList() bool {
	return n.Children != nil
}

// Return true if the node is a scalar (has no children).
func (n *Node) IsScalar() bool {
	return n.Children == nil
}

func (n *Node) String() string {
	return n.Value
}

// Returns the number of children nodes. Has O(N) complexity.
func (n *Node) NumChildren() int {
	i := 0
	c := n.Children
	for c != nil {
		i++
		c = c.Next
	}
	return i
}

// Returns the number of sibling nodes. Has O(N) complexity.
func (n *Node) NumSiblings() int {
	i := 0
	s := n.Next
	for s != nil {
		i++
		s = s.Next
	}
	return i
}

// This error structure is Parse* functions family specific, it returns information
// about errors encountered during parsing. Location can be decoded using the
// context you passed in as an argument. If the context was nil, then the location
// is simply a byte offset from the beginning of the input stream.
type Error struct {
	Location SourceLoc
	message  string
}

// Satisfy the built-in error interface. Returns the error message (without
// source location).
func (e *Error) Error() string {
	return e.message
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
