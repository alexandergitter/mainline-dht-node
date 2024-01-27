package main

import "testing"

func TestNodeIdBitSet(t *testing.T) {
	var id = hexStringToNodeId("9000000080000000000000000000000000000001")
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

func TestLongestCommonPrefixLength(t *testing.T) {
	var a = hexStringToNodeId("0000000000000000000000000000000000000000")
	var b = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var c = hexStringToNodeId("00ffffffffffffffffffffffffffffffffffffff")
	var d = hexStringToNodeId("002fffffffffffffffffffffffffffffffffffff")
	var e = hexStringToNodeId("007fffffffffffffffffffffffffffffffffffff")

	if commonPrefixLength(a, a) != 160 {
		t.Error("Expected 160")
	}
	if commonPrefixLength(a, b) != 0 {
		t.Error("Expected 0")
	}
	if commonPrefixLength(a, c) != 8 {
		t.Error("Expected 8")
	}
	if commonPrefixLength(a, d) != 10 {
		t.Error("Expected 10")
	}
	if commonPrefixLength(a, e) != 9 {
		t.Error("Expected 9")
	}
}
