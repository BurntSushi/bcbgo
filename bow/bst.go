package bow

type bst struct {
	root     *node
	min, max *node
	size     int
}

type node struct {
	Entry
	distance    float64
	left, right *node
}

func (tree *bst) insert(entry Entry, distance float64) {
	newn := &node{entry, distance, nil, nil}
	if tree.root == nil {
		tree.root = newn
	} else {
		tree.root.insert(newn)
	}

	if tree.min == nil || newn.distance < tree.min.distance {
		tree.min = newn
	}
	if tree.max == nil || newn.distance < tree.max.distance {
		tree.max = newn
	}
	tree.size += 1
}

func (n *node) insert(newn *node) {
	if newn.distance < n.distance {
		if n.left == nil {
			n.left = newn
		} else {
			n.left.insert(newn)
		}
	} else {
		if n.right == nil {
			n.right = newn
		} else {
			n.right.insert(newn)
		}
	}
}

func (tree *bst) maxNode() *node {
	if tree.root == nil {
		return nil
	}

	var n *node
	for n = tree.root; n.right != nil; n = n.right {
	}
	return n
}

func (tree *bst) minNode() *node {
	if tree.root == nil {
		return nil
	}

	var n *node
	for n = tree.root; n.left != nil; n = n.left {
	}
	return n
}

func (tree *bst) deleteMax() {
	if tree.root == nil {
		return
	}
	if tree.root.right == nil {
		if tree.root.left != nil {
			tree.root = tree.root.left
			tree.max = tree.maxNode()
			tree.size -= 1
		}
		return
	}

	var n *node
	for n = tree.root; n.right.right != nil; n = n.right {
	}
	if n.right.left != nil {
		n.right = n.right.left
		tree.max = tree.maxNode()
	} else {
		n.right = nil
		tree.max = n
	}
	tree.size -= 1
}

func (tree *bst) deleteMin() {
	if tree.root == nil {
		return
	}
	if tree.root.left == nil {
		if tree.root.right != nil {
			tree.root = tree.root.right
			tree.min = tree.minNode()
			tree.size -= 1
		}
		return
	}

	var n *node
	for n = tree.root; n.left.left != nil; n = n.left {
	}
	if n.left.right != nil {
		n.left = n.left.right
		tree.min = tree.minNode()
	} else {
		n.left = nil
		tree.min = n
	}
	tree.size -= 1
}

func (n *node) inorder(visit func(*node)) {
	if n == nil {
		return
	}
	if n.left != nil {
		n.left.inorder(visit)
	}
	visit(n)
	if n.right != nil {
		n.right.inorder(visit)
	}
}

func (n *node) inorderReverse(visit func(*node)) {
	if n == nil {
		return
	}
	if n.right != nil {
		n.right.inorderReverse(visit)
	}
	visit(n)
	if n.left != nil {
		n.left.inorderReverse(visit)
	}
}
