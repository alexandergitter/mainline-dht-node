package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
)

const ENTRIES = 8

func getMyIp() (net.IP, error) {
	var res, err = http.Get("https://api.ipify.org")
	if err != nil {
		return nil, err
	}

	ipStr, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var result = net.ParseIP(string(ipStr))
	return result, nil
}

func main() {
	var ownId = hexStringToNodeId("0000000000000000000000000000000000000000")

	//var ownId = make([]byte, 20)
	//_, err := rand.Read(ownId)
	//if err != nil {
	//	log.Fatal("Could not generate random node ID", err)
	//}

	var node1 = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var node2 = hexStringToNodeId("4000000000000000000000000000000000000001")
	var node3 = hexStringToNodeId("fffffffffffffffffffffffffffffffffffffffe")
	var node4 = hexStringToNodeId("7000000000000000000000000000000000000000")
	var node5 = hexStringToNodeId("4102030405060708090a0b0c0d0e0f1011121319")
	var node6 = hexStringToNodeId("3000000000000011111111111111111111111111")

	var table = newRoutingTable(2, dhtNode{nodeId: ownId})
	table.addEntry(dhtNode{nodeId: node1})
	table.addEntry(dhtNode{nodeId: node2})
	table.addEntry(dhtNode{nodeId: node3})
	table.addEntry(dhtNode{nodeId: node4})
	table.addEntry(dhtNode{nodeId: node5})
	table.addEntry(dhtNode{nodeId: node6})
	printRoutingTable(table)

	var nodeId1 = hexStringToNodeId("ffffffffffffffffffffffffffffffffffffffff")
	var nodeId2 = hexStringToNodeId("0fffffffffffffffffffffffffffffffffffffff")
	var nodeId3 = hexStringToNodeId("00ffffffffffffffffffffffffffffffffffffff")
	var nodeId4 = hexStringToNodeId("000fffffffffffffffffffffffffffffffffffff")

	table = newRoutingTable(2, dhtNode{nodeId: ownId})
	table.addEntry(dhtNode{nodeId: nodeId1})
	table.addEntry(dhtNode{nodeId: nodeId2})
	table.addEntry(dhtNode{nodeId: nodeId3})
	table.addEntry(dhtNode{nodeId: nodeId4})
	printRoutingTable(table)

	v, _ := decodeBencode("d1:yli324ee1:ad2:id20:abcdefghij0123456789e1:q4:ping1:t2:aae")
	fmt.Println(bencodeValueToString(v))
}
