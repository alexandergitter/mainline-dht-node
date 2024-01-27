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

func TestBucketGetEntryByIdOrAll(t *testing.T) {
	var bucket = newBucket(8)

	var nodeId1 = hexStringToNodeId("9000000800900000080000000000000000000001")
	var nodeId2 = hexStringToNodeId("9000000800900000080000000000000000000002")
	var nodeId3 = hexStringToNodeId("9000000800900000080000000000000000000003")

	bucket.addEntry(dhtNode{nodeId: nodeId1})
	bucket.addEntry(dhtNode{nodeId: nodeId2})

	var result, exactMatch = bucket.getEntryByIdOrAll(nodeId1)
	if !exactMatch || len(result) != 1 || !result[0].nodeId.isEqual(nodeId1) {
		t.Error("Expected exact match")
	}

	result, exactMatch = bucket.getEntryByIdOrAll(nodeId2)
	if !exactMatch || len(result) != 1 || !result[0].nodeId.isEqual(nodeId2) {
		t.Error("Expected exact match")
	}

	result, exactMatch = bucket.getEntryByIdOrAll(nodeId3)
	if exactMatch || len(result) != 2 {
		t.Error("Expected all entries")
	}
}
