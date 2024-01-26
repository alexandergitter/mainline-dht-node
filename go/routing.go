package main

import (
	"fmt"
	"strconv"
	"strings"
)

type nodeId [20]byte

func (n nodeId) bitSet(index int) bool {
	return (n[index/8] & (1 << uint(7-index%8))) != 0
}

type routingEntry struct {
	nodeId nodeId
}

type routingTable struct {
	bucketSize int
	ownNodeId  nodeId
	root       routingNode
}

func newRoutingTable(bucketSize int, ownNodeId nodeId) routingTable {
	return routingTable{
		bucketSize: bucketSize,
		ownNodeId:  ownNodeId,
		root: leafNode{
			occupied: make([]bool, bucketSize),
			entries:  make([]routingEntry, bucketSize),
		},
	}
}

type traversalContext struct {
	ownNodeId   nodeId
	depth       int
	isOwnBucket bool
	bucketSize  int
}

func (tc traversalContext) next(bitSet bool) traversalContext {
	if bitSet {
		return traversalContext{
			ownNodeId:   tc.ownNodeId,
			depth:       tc.depth + 1,
			isOwnBucket: tc.isOwnBucket && tc.ownNodeId.bitSet(tc.depth),
			bucketSize:  tc.bucketSize,
		}
	} else {
		return traversalContext{
			ownNodeId:   tc.ownNodeId,
			depth:       tc.depth + 1,
			isOwnBucket: tc.isOwnBucket && !tc.ownNodeId.bitSet(tc.depth),
			bucketSize:  tc.bucketSize,
		}
	}
}

func (t *routingTable) addEntry(entry routingEntry) {
	t.root = t.root.addEntry(entry, traversalContext{
		ownNodeId:   t.ownNodeId,
		depth:       0,
		isOwnBucket: true,
		bucketSize:  t.bucketSize,
	})
}

type routingNode interface {
	isLeaf() bool
	addEntry(entry routingEntry, tc traversalContext) routingNode
}

type innerNode struct {
	left  routingNode
	right routingNode
}

type leafNode struct {
	occupied []bool
	entries  []routingEntry
}

func (n innerNode) isLeaf() bool {
	return false
}

func (n innerNode) addEntry(entry routingEntry, tc traversalContext) routingNode {
	if tc.depth < 0 || tc.depth > 159 {
		panic("Invalid routing tree depth")
	}

	if entry.nodeId.bitSet(tc.depth) {
		n.right = n.right.addEntry(entry, tc.next(true))
	} else {
		n.left = n.left.addEntry(entry, tc.next(false))
	}

	return n
}

func (n leafNode) isLeaf() bool {
	return true
}

func (n leafNode) addEntry(entry routingEntry, tc traversalContext) routingNode {
	if tc.depth < 0 || tc.depth > 159 {
		panic("Invalid routing tree depth")
	}

	for i, occupied := range n.occupied {
		if !occupied {
			n.occupied[i] = true
			n.entries[i] = entry
			return n
		}
	}

	// All entries in this leaf are occupied:
	// - if this is not our own bucket we just drop the entry
	if !tc.isOwnBucket {
		return n
	}

	// - otherwise, split first
	return n.split(tc).addEntry(entry, tc)
}

func (n leafNode) split(tc traversalContext) innerNode {
	if tc.depth < 0 || tc.depth > 159 {
		panic("Invalid routing tree depth")
	}

	var leftSide = leafNode{
		occupied: make([]bool, tc.bucketSize),
		entries:  make([]routingEntry, tc.bucketSize),
	}

	for i, entry := range n.entries {
		if !entry.nodeId.bitSet(tc.depth) {
			leftSide.occupied[i] = true
			leftSide.entries[i] = entry

			n.occupied[i] = false
			n.entries[i] = routingEntry{}
		}
	}

	return innerNode{
		left:  leftSide,
		right: n,
	}
}

// Debugging helpers

func maxDepth(n routingNode) int {
	if n.isLeaf() {
		return 0
	} else {
		var inner = n.(innerNode)
		return 1 + max(maxDepth(inner.left), maxDepth(inner.right))
	}
}

func printRoutingTableTree(t routingTable) {
	printSubTree(t.root, "", traversalContext{
		ownNodeId:   t.ownNodeId,
		depth:       0,
		isOwnBucket: true,
		bucketSize:  t.bucketSize,
	})
}

func printSubTree(n routingNode, leading string, tc traversalContext) {
	fmt.Print(leading)

	if n.isLeaf() {
		var leaf = n.(leafNode)
		if tc.isOwnBucket {
			fmt.Print(">  ")
		} else {
			fmt.Print("|  ")
		}
		for i, entry := range leaf.entries {
			if leaf.occupied[i] {
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

func nodeIdToString(n nodeId) string {
	var result = make([]string, 20)
	for i, b := range n {
		result[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(result, "")
}

func hexStringToNodeId(s string) (nodeId, error) {
	var result nodeId

	if len(s) != 40 {
		return result, fmt.Errorf("Invalid node ID length")
	}

	for i := 0; i < 20; i++ {
		if b, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8); err != nil {
			return result, err
		} else {
			result[i] = byte(b)
		}
	}

	return result, nil
}
