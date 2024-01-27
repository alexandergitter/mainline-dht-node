package main

import "fmt"

type routingTree struct {
	bucketSize int
	ownNodeId  nodeId
	root       treeNode
}

func newRoutingTree(bucketSize int, ownNodeId nodeId) routingTree {
	return routingTree{
		bucketSize: bucketSize,
		ownNodeId:  ownNodeId,
		root: leafNode{
			bucket: newBucket(bucketSize),
		},
	}
}

type traversalContext struct {
	ownNodeId   nodeId
	depth       int
	isOwnBucket bool
}

func (tc traversalContext) next(bitSet bool) traversalContext {
	if bitSet {
		return traversalContext{
			ownNodeId:   tc.ownNodeId,
			depth:       tc.depth + 1,
			isOwnBucket: tc.isOwnBucket && tc.ownNodeId.isBitSet(tc.depth),
		}
	} else {
		return traversalContext{
			ownNodeId:   tc.ownNodeId,
			depth:       tc.depth + 1,
			isOwnBucket: tc.isOwnBucket && !tc.ownNodeId.isBitSet(tc.depth),
		}
	}
}

func (t *routingTree) addEntry(entry dhtNode) {
	t.root = t.root.addEntry(entry, traversalContext{
		ownNodeId:   t.ownNodeId,
		depth:       0,
		isOwnBucket: true,
	})
}

type treeNode interface {
	isLeaf() bool
	addEntry(entry dhtNode, tc traversalContext) treeNode
	containsNodeId(id nodeId) bool
}

type innerNode struct {
	left  treeNode
	right treeNode
}

type leafNode struct {
	bucket bucket
}

func (n innerNode) isLeaf() bool {
	return false
}

func (n innerNode) addEntry(entry dhtNode, tc traversalContext) treeNode {
	if tc.depth < 0 || tc.depth > 159 {
		panic("Invalid routing tree depth")
	}

	if entry.nodeId.isBitSet(tc.depth) {
		n.right = n.right.addEntry(entry, tc.next(true))
	} else {
		n.left = n.left.addEntry(entry, tc.next(false))
	}

	return n
}

func (n innerNode) containsNodeId(id nodeId) bool {
	return n.left.containsNodeId(id) || n.right.containsNodeId(id)
}

func (n leafNode) isLeaf() bool {
	return true
}

func (n leafNode) addEntry(entry dhtNode, tc traversalContext) treeNode {
	if tc.depth < 0 || tc.depth > 159 {
		panic("Invalid routing tree depth")
	}

	var success bool
	n.bucket, success = n.bucket.addEntry(entry)

	if success {
		return n
	}

	// All entries in this leaf are occupied:
	// - if this is not our own bucket we just drop the entry
	if !tc.isOwnBucket {
		return n
	}

	// - otherwise, split first
	return n.split(tc).addEntry(entry, tc)
}

func (n leafNode) containsNodeId(id nodeId) bool {
	return n.bucket.containsNodeId(id)
}

func (n leafNode) split(tc traversalContext) innerNode {
	var zeroBucket, oneBucket = n.bucket.splitAt(tc.depth)
	n.bucket = oneBucket

	return innerNode{
		left:  leafNode{bucket: zeroBucket},
		right: n,
	}
}

// Helpers

func maxDepth(n treeNode) int {
	if n.isLeaf() {
		return 0
	} else {
		var inner = n.(innerNode)
		return 1 + max(maxDepth(inner.left), maxDepth(inner.right))
	}
}

func printRoutingTableTree(t routingTree) {
	printSubTree(t.root, "", traversalContext{
		ownNodeId:   t.ownNodeId,
		depth:       0,
		isOwnBucket: true,
	})
}

func printSubTree(n treeNode, leading string, tc traversalContext) {
	fmt.Print(leading)

	if n.isLeaf() {
		var leaf = n.(leafNode)
		if tc.isOwnBucket {
			fmt.Print(">  ")
		} else {
			fmt.Print("|  ")
		}
		for i, entry := range leaf.bucket.entries {
			if leaf.bucket.occupied[i] {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(nodeIdToString(entry.nodeId))
			}
		}
		fmt.Println()
	} else {
		var inner = n.(innerNode)
		fmt.Println("*")
		printSubTree(inner.left, leading+"0--", tc.next(false))
		printSubTree(inner.right, leading+"1--", tc.next(true))
	}
}
