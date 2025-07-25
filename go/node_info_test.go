package main

import (
	"net"
	"testing"
)

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

func TestLongestCommonPrefixLength(t *testing.T) {
	var a, _ = hexStringToNodeId("0000000000000000000000000000000000000000")
	var b, _ = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var c, _ = hexStringToNodeId("00ffffffffffffffffffffffffffffffffffffff")
	var d, _ = hexStringToNodeId("002fffffffffffffffffffffffffffffffffffff")
	var e, _ = hexStringToNodeId("007fffffffffffffffffffffffffffffffffffff")

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

func TestCompactNodeInfo(t *testing.T) {
	var id, _ = hexStringToNodeId("000100020003000400050006000700080009000a")
	var node = nodeInfo{
		nodeId: id,
		address: net.UDPAddr{
			IP:   net.ParseIP("12.34.56.78"),
			Port: 0x9876,
		},
	}

	var compactBytes, _ = hexStringToBytes("000100020003000400050006000700080009000a0c22384e9876")
	if node.compactNodeInfo() != string(compactBytes) {
		t.Error("Expected", bytesToHexString(compactBytes), "but got", bytesToHexString([]byte(node.compactNodeInfo())))
	}

	var decoded, _ = decodeCompactNodeInfo(string(compactBytes))
	if !decoded.nodeId.isEqual(id) || decoded.address.IP.String() != "12.34.56.78" || decoded.address.Port != 0x9876 {
		t.Error("Got wrong decoded node info")
	}
}
