package sexp

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

// Returns Nth child or sibling node. If the node is a list, then zero is the
// first child, otherwise it means self.
func (n *Node) Nth(num int) (*Node, error) {
	if n.IsList() {
		i := 0
		for c := n.Children; c != nil; c = c.Next {
			if i == num {
				return c, nil
			}
			i++
		}

		num++
		return nil, new_error(n.Location,
			"cannot retrieve %d%s child node, %s",
			num, number_suffix(num),
			the_list_has_n_children(n.NumChildren()))
	} else {
		if num == 0 {
			return n, nil
		}

		i := 1
		for s := n.Next; s != nil; s = s.Next {
			if i == num {
				return s, nil
			}
			i++
		}

		return nil, new_error(n.Location,
			"cannot retrieve %d%s sibling node, "+
				"the list has %d siblings only",
			num, number_suffix(num),
			the_list_has_n_siblings(n.NumSiblings()))
	}
	panic("unreachable")
}

// Walk over children or siblings.
func (n *Node) IterNodes(f func(n *Node)) {
	var c *Node
	if n.IsList() {
		c = n.Children
	} else {
		f(n)
		c = n.Next
	}

	for ; c != nil; c = c.Next {
		f(c)
	}
}

// Walk over children nodes, assuming they are key/value pairs. It returns error
// if the iterable node is not a list or if any of its children is not a
// key/value pair.
func (n *Node) IterKeyValues(f func(k, v *Node)) error {
	if !n.IsList() {
		return new_error(n.Location,
			"node is not a list, expected list of key/value pairs")
	}

	for c := n.Children; c != nil; c = c.Next {
		if !c.IsList() {
			return new_error(c.Location,
				"node is not a list, expected key/value pair")
		}
		k, err := c.Nth(0)
		if err != nil {
			return err
		}
		v, err := c.Nth(1)
		if err != nil {
			return err
		}
		f(k, v)
	}
	return nil
}

// Unmarshals node to a list of pointers. TODO: more details here.
func (n *Node) Unmarshal(args ...interface{}) error {
	return nil
}
