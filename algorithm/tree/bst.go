package tree

import "fmt"

type BST struct {
	root *Node
}

func NewBST() *BST {
	return &BST{}
}

func (b *BST) Insert(value int) {
	b.root = _insert(b.root, value)
}

func _insert(node *Node, value int) *Node {
	if node == nil {
		return &Node{Value: value}
	}

	if value < node.Value {
		node.Left = _insert(node.Left, value)
	} else {
		node.Right = _insert(node.Right, value)
	}

	return node
}

func (b *BST) PreTraversal() {
	_preTraversal(b.root)
}

func _preTraversal(node *Node) {
	if node == nil {
		return
	}
	fmt.Println(node.Value)
	_preTraversal(node.Left)
	_preTraversal(node.Right)
}

func (b *BST) InTraversal() {
	_inTraversal(b.root)
}

func _inTraversal(node *Node) {
	if node == nil {
		return
	}
	_inTraversal(node.Left)
	fmt.Println(node.Value)
	_inTraversal(node.Right)
}

func (b *BST) PostTraversal() {
	_postTraversal(b.root)
}

func _postTraversal(node *Node) {
	if node == nil {
		return
	}
	_postTraversal(node.Left)
	_postTraversal(node.Right)
	fmt.Println(node.Value)
}
