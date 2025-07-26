package main

import "testing"

func TestRoutingTableAddEntry(t *testing.T) {
	var ownId, _ = hexStringToNodeId("0000000000000000000000000000000000000000")
	var distantId1, _ = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var distantId2, _ = hexStringToNodeId("8000000000000000000000000000000000000000")
	var distantId3, _ = hexStringToNodeId("ffffffffffffff00000000000000000000000000")
	var nearId1, _ = hexStringToNodeId("7000000000000000000000000000000000000000")
	var nearId2, _ = hexStringToNodeId("0000000000ffffffffffffffffffffffffffffff")

	var table = newRoutingTable(2, nodeInfo{nodeId: ownId})
	table.addEntry(nodeInfo{nodeId: distantId1})
	var sizeBeforeDuplicateAdded = len(table.table[0].entries)
	table.addEntry(nodeInfo{nodeId: distantId1})

	if len(table.table[0].entries) != sizeBeforeDuplicateAdded {
		t.Error("Expected addEntry to not add duplicate entry")
	}

	table.addEntry(nodeInfo{nodeId: distantId2})
	table.addEntry(nodeInfo{nodeId: distantId3})

	// Tree should be split at this point, with the latest distant node discarded
	if len(table.table) <= 1 {
		t.Error("Expected table to have more than one bucket")
	}

	if table.table[1].containsNodeId(distantId1) {
		t.Error("Expected bucket with longer prefix to not contain distantId1")
	}
	if table.table[1].containsNodeId(distantId2) {
		t.Error("Expected bucket with longer prefix to not contain distantId2")
	}
	if table.table[1].containsNodeId(distantId3) {
		t.Error("Expected bucket with longer prefix to not contain distantId3")
	}
	if !table.table[0].containsNodeId(distantId1) {
		t.Error("Expected bucket with shorter prefix to contain distantId1")
	}
	if !table.table[0].containsNodeId(distantId2) {
		t.Error("Expected bucket with shorter prefix to contain distantId2")
	}
	if table.table[0].containsNodeId(distantId3) {
		t.Error("Expected bucket with shorter prefix to not contain distantId3")
	}

	table.addEntry(nodeInfo{nodeId: nearId1})
	table.addEntry(nodeInfo{nodeId: nearId2})

	if !table.table[1].containsNodeId(nearId1) {
		t.Error("Expected bucket with longer prefix to contain nearId1")
	}
	if !table.table[1].containsNodeId(nearId2) {
		t.Error("Expected bucket with longer prefix to contain nearId2")
	}
}

func TestRoutingTableFindNode(t *testing.T) {
	var ownId, _ = hexStringToNodeId("0000000000000000000000000000000000000000")
	var nodeId1, _ = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var nodeId2, _ = hexStringToNodeId("0fffffffffffffffffffffffffffffffffffffff")
	var nodeId3, _ = hexStringToNodeId("00ffffffffffffffffffffffffffffffffffffff")
	var nodeId4, _ = hexStringToNodeId("000fffffffffffffffffffffffffffffffffffff")

	var table = newRoutingTable(2, nodeInfo{nodeId: ownId})
	table.addEntry(nodeInfo{nodeId: nodeId1})
	table.addEntry(nodeInfo{nodeId: nodeId2})
	table.addEntry(nodeInfo{nodeId: nodeId3})
	table.addEntry(nodeInfo{nodeId: nodeId4})

	// At this point, the routing table looks something like this:
	// 0: [nodeId1]
	// 1: []
	// 2: []
	// 3: []
	// 4: [nodeId2]
	// 5: [nodeId3, nodeId4]

	var result, exactMatch = table.findNode(nodeId1)
	if !exactMatch || len(result) != 1 || !result[0].nodeId.isEqual(nodeId1) {
		t.Error("Expected exact match")
	}

	result, exactMatch = table.findNode(nodeId4)
	if !exactMatch || len(result) != 1 || !result[0].nodeId.isEqual(nodeId4) {
		t.Error("Expected exact match")
	}

	var query, _ = hexStringToNodeId("faaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	result, exactMatch = table.findNode(query) // bucket 0
	if exactMatch || len(result) != 2 || !result[0].nodeId.isEqual(nodeId1) || !result[1].nodeId.isEqual(nodeId2) {
		t.Error("Expected one entry from bucket 0 and one from bucket 4, got: ", result)
	}

	query, _ = hexStringToNodeId("3aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	result, exactMatch = table.findNode(query) // bucket 2
	if exactMatch || len(result) != 2 || !result[0].nodeId.isEqual(nodeId1) || !result[1].nodeId.isEqual(nodeId2) {
		t.Error("Expected one entry from bucket 0 and one from bucket 4, got: ", result)
	}

	query, _ = hexStringToNodeId("1faaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	result, exactMatch = table.findNode(query) // bucket 3
	if exactMatch || len(result) != 2 || !result[0].nodeId.isEqual(nodeId2) || !result[1].nodeId.isEqual(nodeId3) {
		t.Error("Expected one entry from bucket 4 and one from bucket 5, got: ", result)
	}

	query, _ = hexStringToNodeId("000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	result, exactMatch = table.findNode(query) // bucket 5
	if exactMatch || len(result) != 2 || !result[0].nodeId.isEqual(nodeId3) || !result[1].nodeId.isEqual(nodeId4) {
		t.Error("Expected all from bucket 5, got: ", result)
	}
}

func TestRoutingTableFindOwnNode(t *testing.T) {
	var ownId, _ = hexStringToNodeId("1234000000000000000000000000000000000000")
	var nodeId1, _ = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var nodeId2, _ = hexStringToNodeId("0fffffffffffffffffffffffffffffffffffffff")
	var table = newRoutingTable(2, nodeInfo{nodeId: ownId})
	table.addEntry(nodeInfo{nodeId: nodeId1})
	table.addEntry(nodeInfo{nodeId: nodeId2})

	var result, exactMatch = table.findNode(ownId)
	if !exactMatch || len(result) != 1 || !result[0].nodeId.isEqual(ownId) {
		t.Error("Expected exact match with node entry, go:", result)
	}

	result, exactMatch = table.findNodeWithoutSelf(ownId)
	if exactMatch || len(result) != 2 || result[0].nodeId.isEqual(ownId) || result[1].nodeId.isEqual(ownId) {
		t.Error("Expected findNodeWithoutSelf to not match ownId, got: ", result)
	}
}
