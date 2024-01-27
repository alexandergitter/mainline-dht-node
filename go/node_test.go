package main

import "testing"

func TestNodeIdBitSet(t *testing.T) {
	var id, _ = hexStringToNodeId("9000000080000000000000000000000000000001")
	if id.isBitSet(0) != true {
		t.Error("Expected bitSet(0) to return true")
	}
	if id.isBitSet(1) != false {
		t.Error("Expected bitSet(1) to return false")
	}
	if id.isBitSet(3) != true {
		t.Error("Expected bitSet(3) to return true")
	}
	if id.isBitSet(32) != true {
		t.Error("Expected bitSet(32) to return true")
	}
	if id.isBitSet(33) != false {
		t.Error("Expected bitSet(33) to return false")
	}
	if id.isBitSet(159) != true {
		t.Error("Expected bitSet(159) to return true")
	}
}
