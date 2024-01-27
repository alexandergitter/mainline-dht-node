package main

import "fmt"

const ENTRIES = 8

func main() {
	var ownId, _ = hexStringToNodeId("0000000000000000000000000000000000000000")
	//var ownId = make([]byte, 20)
	//_, err := rand.Read(ownId)
	//if err != nil {
	//	log.Fatal("Could not generate random node ID", err)
	//}

	var node1, _ = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var node2, _ = hexStringToNodeId("4000000000000000000000000000000000000001")
	var node3, _ = hexStringToNodeId("fffffffffffffffffffffffffffffffffffffffe")
	var node4, _ = hexStringToNodeId("7000000000000000000000000000000000000000")
	var node5, _ = hexStringToNodeId("4102030405060708090a0b0c0d0e0f1011121319")
	var node6, _ = hexStringToNodeId("3000000000000011111111111111111111111111")

	var table = newRoutingTable(2, nodeId(ownId))
	table.addEntry(dhtNode{nodeId: node1})
	table.addEntry(dhtNode{nodeId: node2})
	table.addEntry(dhtNode{nodeId: node3})
	table.addEntry(dhtNode{nodeId: node4})
	table.addEntry(dhtNode{nodeId: node5})
	table.addEntry(dhtNode{nodeId: node6})
	printRoutingTable(table)

	v, _ := decodeBencode("d1:yli324ee1:ad2:id20:abcdefghij0123456789e1:q4:ping1:t2:aae")
	fmt.Println(bencodeValueToString(v))
}
