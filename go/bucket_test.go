package main

import "testing"

func TestBucketAddEntry(t *testing.T) {
	var bucket = newBucket(2)

	var nodeId1 = hexStringToNodeId("9000000800900000080000000000000000000001")
	var nodeId2 = hexStringToNodeId("9000000800900000080000000000000000000002")
	var nodeId3 = hexStringToNodeId("9000000800900000080000000000000000000003")

	var updated, success = bucket.addEntry(dhtNode{nodeId: nodeId1})
	if !success {
		t.Error("Expected addEntry to return true")
	}

	updated, success = bucket.addEntry(dhtNode{nodeId: nodeId2})
	if !success {
		t.Error("Expected addEntry to return true")
	}

	updated, success = bucket.addEntry(dhtNode{nodeId: nodeId3})
	if success {
		t.Error("Expected addEntry to return false")
	}
	if !updated.containsNodeId(nodeId1) {
		t.Error("Expected bucket to contain nodeId1")
	}
	if !updated.containsNodeId(nodeId2) {
		t.Error("Expected bucket to contain nodeId2")
	}
	if updated.containsNodeId(nodeId3) {
		t.Error("Expected bucket to not contain nodeId3")
	}
}

func TestBucketGetEntryByIdOrReturnAll(t *testing.T) {
	var bucket = newBucket(8)

	var nodeId1 = hexStringToNodeId("9000000800900000080000000000000000000001")
	var nodeId2 = hexStringToNodeId("9000000800900000080000000000000000000002")
	var nodeId3 = hexStringToNodeId("9000000800900000080000000000000000000003")

	bucket.addEntry(dhtNode{nodeId: nodeId1})
	bucket.addEntry(dhtNode{nodeId: nodeId2})

	var result, exactMatch = bucket.getEntryByIdOrReturnAll(nodeId1)
	if !exactMatch || len(result) != 1 || !result[0].nodeId.isEqual(nodeId1) {
		t.Error("Expected exact match")
	}

	result, exactMatch = bucket.getEntryByIdOrReturnAll(nodeId2)
	if !exactMatch || len(result) != 1 || !result[0].nodeId.isEqual(nodeId2) {
		t.Error("Expected exact match")
	}

	result, exactMatch = bucket.getEntryByIdOrReturnAll(nodeId3)
	if exactMatch || len(result) != 2 {
		t.Error("Expected all entries")
	}
}

func TestSplitAt(t *testing.T) {
	var bucket = newBucket(8)
	var nodeId1 = hexStringToNodeId("0000000000000000000000000000000000000001")
	var nodeId2 = hexStringToNodeId("f000000000000000000000000000000000000002")
	var nodeId3 = hexStringToNodeId("0000000000000000000000000000000000000003")
	var nodeId4 = hexStringToNodeId("0000000000000000000000000000000000000004")
	var nodeId5 = hexStringToNodeId("f000000000000000000000000000000000000005")

	bucket.addEntry(dhtNode{nodeId: nodeId1})
	bucket.addEntry(dhtNode{nodeId: nodeId2})
	bucket.addEntry(dhtNode{nodeId: nodeId3})
	bucket.addEntry(dhtNode{nodeId: nodeId4})
	bucket.addEntry(dhtNode{nodeId: nodeId5})

	var zero, one = bucket.splitAt(0)
	var query = hexStringToNodeId("0000000000000000000000000000000000000000")
	var result, exactMatch = zero.getEntryByIdOrReturnAll(query)
	if exactMatch || len(result) != 3 {
		t.Error("Expected zero to have 3 entries")
	}
	result, exactMatch = one.getEntryByIdOrReturnAll(query)
	if exactMatch || len(result) != 2 {
		t.Error("Expected one to have 2 entries")
	}
	if !zero.containsNodeId(nodeId1) {
		t.Error("Expected zero to contain nodeId1")
	}
	if !one.containsNodeId(nodeId2) {
		t.Error("Expected one to contain nodeId2")
	}
}
