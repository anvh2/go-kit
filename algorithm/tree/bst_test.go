package tree

import "testing"

func TestBST(t *testing.T) {
	bst := NewBST()
	bst.Insert(5)
	bst.Insert(3)
	bst.Insert(4)
	bst.Insert(2)

	bst.PreTraversal()
	bst.InTraversal()
	bst.PostTraversal()
}
