package main

import (
	"testing"
)

func TestBucketAddEntry(t *testing.T) {
	var bucket = newBucket(2)

	var nodeId1, _ = hexStringToNodeId("9000000800900000080000000000000000000001")
	var nodeId2, _ = hexStringToNodeId("9000000800900000080000000000000000000002")
	var nodeId3, _ = hexStringToNodeId("9000000800900000080000000000000000000003")

	bucket, success := bucket.addEntry(nodeInfo{nodeId: nodeId1})
	if !success {
		t.Error("Expected addEntry to return true")
	}

	bucket, success = bucket.addEntry(nodeInfo{nodeId: nodeId2})
	if !success {
		t.Error("Expected addEntry to return true")
	}

	bucket, success = bucket.addEntry(nodeInfo{nodeId: nodeId3})
	if success {
		t.Error("Expected addEntry to return false")
	}
	if !bucket.containsNodeId(nodeId1) {
		t.Error("Expected bucket to contain nodeId1")
	}
	if !bucket.containsNodeId(nodeId2) {
		t.Error("Expected bucket to contain nodeId2")
	}
	if bucket.containsNodeId(nodeId3) {
		t.Error("Expected bucket to not contain nodeId3")
	}
}

func TestBucketGetEntryByIdOrReturnAll(t *testing.T) {
	var bucket = newBucket(8)

	var nodeId1, _ = hexStringToNodeId("9000000800900000080000000000000000000001")
	var nodeId2, _ = hexStringToNodeId("9000000800900000080000000000000000000002")
	var nodeId3, _ = hexStringToNodeId("9000000800900000080000000000000000000003")

	bucket, _ = bucket.addEntry(nodeInfo{nodeId: nodeId1})
	bucket, _ = bucket.addEntry(nodeInfo{nodeId: nodeId2})

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
	var nodeId1, _ = hexStringToNodeId("0000000000000000000000000000000000000001")
	var nodeId2, _ = hexStringToNodeId("f000000000000000000000000000000000000002")
	var nodeId3, _ = hexStringToNodeId("0000000000000000000000000000000000000003")
	var nodeId4, _ = hexStringToNodeId("0000000000000000000000000000000000000004")
	var nodeId5, _ = hexStringToNodeId("f000000000000000000000000000000000000005")

	bucket, _ = bucket.addEntry(nodeInfo{nodeId: nodeId1})
	bucket, _ = bucket.addEntry(nodeInfo{nodeId: nodeId2})
	bucket, _ = bucket.addEntry(nodeInfo{nodeId: nodeId3})
	bucket, _ = bucket.addEntry(nodeInfo{nodeId: nodeId4})
	bucket, _ = bucket.addEntry(nodeInfo{nodeId: nodeId5})

	var zero, one = bucket.splitAt(0)
	if len(zero.entries) != 3 {
		t.Error("Expected zero to have 3 entries")
	}
	if len(one.entries) != 2 {
		t.Error("Expected one to have 2 entries")
	}
	if !zero.containsNodeId(nodeId1) {
		t.Error("Expected zero to contain nodeId1")
	}
	if !one.containsNodeId(nodeId2) {
		t.Error("Expected one to contain nodeId2")
	}
}
