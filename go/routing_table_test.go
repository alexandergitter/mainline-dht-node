package main

import "testing"

func TestRoutingTableAddEntry(t *testing.T) {
	var ownId, _ = hexStringToNodeId("0000000000000000000000000000000000000000")
	var distantId1, _ = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var distantId2, _ = hexStringToNodeId("8000000000000000000000000000000000000000")
	var distantId3, _ = hexStringToNodeId("ffffffffffffff00000000000000000000000000")
	var nearId1, _ = hexStringToNodeId("7000000000000000000000000000000000000000")
	var nearId2, _ = hexStringToNodeId("0000000000ffffffffffffffffffffffffffffff")

	var table = newRoutingTable(2, ownId)
	table.addEntry(dhtNode{nodeId: distantId1})
	table.addEntry(dhtNode{nodeId: distantId1})

	if table.table[0].occupied[0] == true && table.table[0].occupied[1] == true {
		t.Error("Buckets must not contain duplicate entries")
	}

	table.addEntry(dhtNode{nodeId: distantId2})
	table.addEntry(dhtNode{nodeId: distantId3})

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

	table.addEntry(dhtNode{nodeId: nearId1})
	table.addEntry(dhtNode{nodeId: nearId2})

	if !table.table[1].containsNodeId(nearId1) {
		t.Error("Expected bucket with longer prefix to contain nearId1")
	}
	if !table.table[1].containsNodeId(nearId2) {
		t.Error("Expected bucket with longer prefix to contain nearId2")
	}
}
