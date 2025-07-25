package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
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

	listenOn, err := net.ResolveUDPAddr("udp", os.Args[1])
	if err != nil {
		panic(err)
	}

	var myNodeInfo = nodeInfo{
		nodeId:  ownId,
		address: *listenOn,
	}
	var client = startDhtClient(myNodeInfo, newRoutingTable(ENTRIES, myNodeInfo), listenOn)

	var input string
	for {
		fmt.Scanln(&input)
		switch input {
		case "quit":
			fmt.Println("Exiting...")
			return
		default:
			var addr, _ = net.ResolveUDPAddr("udp", input)
			var dest = nodeInfo{
				nodeId:  ownId,
				address: *addr,
			}
			var res, _ = client.ping(dest)
			fmt.Println(res)
		}
	}
}
