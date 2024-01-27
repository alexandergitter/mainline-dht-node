package main

import "testing"

func TestTraversalContextNext(t *testing.T) {
	var tc = traversalContext{
		ownNodeId:   nodeId{},
		depth:       5,
		isOwnBucket: true,
	}

	// Bit set (1), does not match own node ID
	var next = tc.next(true)
	if next.ownNodeId != tc.ownNodeId || next.depth != 6 || next.isOwnBucket == true {
		t.Error("Expected next to return a new traversal context with depth 6 and isOwnBucket false")
	}

	// Bit not set (0), does match own node ID
	next = tc.next(false)
	if next.ownNodeId != tc.ownNodeId || next.depth != 6 || next.isOwnBucket == false {
		t.Error("Expected next to return a new traversal context with depth 6 and isOwnBucket true")
	}
}

func TestRoutingTreeAddEntry(t *testing.T) {
	var ownId, _ = hexStringToNodeId("0000000000000000000000000000000000000000")
	var distantId1, _ = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var distantId2, _ = hexStringToNodeId("8000000000000000000000000000000000000000")
	var distantId3, _ = hexStringToNodeId("ffffffffffffff00000000000000000000000000")
	var nearId1, _ = hexStringToNodeId("7000000000000000000000000000000000000000")
	var nearId2, _ = hexStringToNodeId("0000000000ffffffffffffffffffffffffffffff")

	var table = newRoutingTree(2, ownId)
	table.addEntry(dhtNode{nodeId: distantId1})
	table.addEntry(dhtNode{nodeId: distantId1})

	if table.root.(leafNode).bucket.occupied[0] == true && table.root.(leafNode).bucket.occupied[1] == true {
		t.Error("Buckets must not contain duplicate entries")
	}

	table.addEntry(dhtNode{nodeId: distantId2})
	table.addEntry(dhtNode{nodeId: distantId3})

	// Tree should be split at this point, with the latest distant node discarded
	if table.root.isLeaf() {
		t.Error("Expected root to be an inner node")
	}

	if table.root.(innerNode).left.(leafNode).bucket.containsNodeId(distantId1) {
		t.Error("Expected left bucket to not contain distantId1")
	}
	if table.root.(innerNode).left.(leafNode).bucket.containsNodeId(distantId2) {
		t.Error("Expected left bucket to not contain distantId2")
	}
	if table.root.(innerNode).left.(leafNode).bucket.containsNodeId(distantId3) {
		t.Error("Expected left bucket to not contain distantId3")
	}
	if !table.root.(innerNode).right.(leafNode).bucket.containsNodeId(distantId1) {
		t.Error("Expected right bucket to contain distantId1")
	}
	if !table.root.(innerNode).right.(leafNode).bucket.containsNodeId(distantId2) {
		t.Error("Expected right bucket to contain distantId2")
	}
	if table.root.(innerNode).right.(leafNode).bucket.containsNodeId(distantId3) {
		t.Error("Expected right bucket to not contain distantId3")
	}

	table.addEntry(dhtNode{nodeId: nearId1})
	table.addEntry(dhtNode{nodeId: nearId2})

	if !table.root.(innerNode).left.(leafNode).bucket.containsNodeId(nearId1) {
		t.Error("Expected left bucket to contain nearId1")
	}
	if !table.root.(innerNode).left.(leafNode).bucket.containsNodeId(nearId2) {
		t.Error("Expected left bucket to contain nearId2")
	}
}
