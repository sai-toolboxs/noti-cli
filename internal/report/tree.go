package report

import (
	"github.com/sai-toolboxs/noti-cli/internal/client"
)

// TreeNode represents a node in the block tree.
type TreeNode struct {
	Block    client.NotionBlock `json:"block"`
	Text     string             `json:"text"`
	Children []*TreeNode        `json:"children,omitempty"`
	Depth    int                `json:"depth"`
}

// BuildTree constructs a tree structure from a flat list of blocks.
// The tree preserves parent-child relationships based on heading levels.
func BuildTree(blocks []client.NotionBlock) []*TreeNode {
	if len(blocks) == 0 {
		return nil
	}

	var roots []*TreeNode
	var stack []*TreeNode

	for _, block := range blocks {
		node := &TreeNode{
			Block: block,
			Text:  extractText(&block),
		}

		if isHeading(block.Type) {
			level := headingLevel(block.Type)

			// Pop nodes from stack until we find a parent with a lower level.
			for len(stack) > 0 && stack[len(stack)-1].Depth >= level {
				stack = stack[:len(stack)-1]
			}

			node.Depth = len(stack)

			if len(stack) == 0 {
				roots = append(roots, node)
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}

			// Push this node onto the stack.
			stack = append(stack, node)
		} else {
			// Non-heading blocks are children of the current heading.
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				node.Depth = parent.Depth + 1
				parent.Children = append(parent.Children, node)
			} else {
				// No heading context, add as root.
				node.Depth = 0
				roots = append(roots, node)
			}
		}
	}

	return roots
}

// BuildTreeWithDepth constructs a tree with a maximum depth limit.
func BuildTreeWithDepth(blocks []client.NotionBlock, maxDepth int) []*TreeNode {
	roots := BuildTree(blocks)
	if maxDepth > 0 {
		roots = limitDepth(roots, maxDepth)
	}
	return roots
}

// limitDepth truncates the tree to the specified maximum depth.
func limitDepth(nodes []*TreeNode, maxDepth int) []*TreeNode {
	var result []*TreeNode
	for _, node := range nodes {
		if node.Depth >= maxDepth {
			// Don't include this node or its children.
			continue
		}
		node.Children = limitDepth(node.Children, maxDepth)
		result = append(result, node)
	}
	return result
}

// CountNodes returns the total number of nodes in the tree.
func CountNodes(nodes []*TreeNode) int {
	count := 0
	for _, node := range nodes {
		count++
		count += CountNodes(node.Children)
	}
	return count
}
