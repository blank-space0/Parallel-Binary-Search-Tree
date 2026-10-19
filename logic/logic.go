package logic

import "errors"

type Node struct {
	left  *Node
	right *Node
	val   int
}

func BuildTree(nums []int) (*Node, error) {
	if len(nums) == 0 {
		err := errors.New("recieved an empty array when constructing BST")
		return nil, err
	}
	root := &Node{nil, nil, nums[0]}
	// skip the root
	for i := 1; i < len(nums); i++ {
		insert(nums[i], root)
	}

	return root, nil
}

func insert(num int, node *Node) *Node {
	if node == nil {
		curr := &Node{nil, nil, num}
		return curr
	}

	if num < node.val {
		node.left = insert(num, node.left)
	} else if num > node.val {
		node.right = insert(num, node.right)
	}

	return node
}

func traverse(node *Node, hash *int) {
	if node == nil {
		return
	}

	traverse(node.left, hash)
	// logic here
	curr := node.val + 2
	*hash = (*hash*curr + curr) % 1000

	traverse(node.right, hash)
}

func HashCompute(root *Node) int {
	hash := 1
	traverse(root, &hash)
	return hash
}

func compare(one *Node, two *Node) bool {
	if one == nil && two == nil {
		return true
	}

	if one == nil || two == nil || one.val != two.val {
		return false
	}

	return compare(one.left, two.left) && compare(one.right, two.right)
}
