package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
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
	var ownId = nodeId{}
	_, err := rand.Read(ownId[:])
	if err != nil {
		log.Fatalf("error while generating node id: %s", err)
	}

	listenOn, err := net.ResolveUDPAddr("udp", "127.0.0.1:6880")
	if err != nil {
		panic(err)
	}

	var myNodeInfo = nodeInfo{
		nodeId:  ownId,
		address: *listenOn,
	}
	var client = newDhtClient(myNodeInfo, newRoutingTable(ENTRIES, myNodeInfo))
	newKrpcRuntime(listenOn).start(client)

	for {
		select {
		default:
			fmt.Scanln()
		}
	}
}
