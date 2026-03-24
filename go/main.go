package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

const ENTRIES = 8

func getMyIp() (net.IP, error) {
	var res, err = http.Get("https://api.ipify.org")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	ipStr, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	// TODO: check and handle for unsuccessful HTTP status codes
	var result = net.ParseIP(string(ipStr))
	return result, nil
}

func printUsage() {
	fmt.Println("available commands:")
	fmt.Println("  ping <ip:port>")
	fmt.Println("  quit")
}

func main() {
	var ownId nodeId

	if len(os.Args) < 3 {
		_, err := rand.Read(ownId[:])
		if err != nil {
			log.Fatalf("error while generating node id: %s", err)
		}
	} else {
		var err error
		ownId, err = hexStringToNodeId(os.Args[2])
		if err != nil {
			log.Fatalf("error parsing node id: %s", err)
		}
	}

	listenOn, err := net.ResolveUDPAddr("udp", os.Args[1])
	if err != nil {
		panic(err)
	}

	fmt.Println("Listening on", listenOn, "with node id", ownId)

	var myNodeInfo = nodeInfo{
		nodeId:  ownId,
		address: *listenOn,
	}
	var client = startDhtClient(myNodeInfo, newRoutingTable(ENTRIES, myNodeInfo), listenOn)

	reader := bufio.NewReader(os.Stdin)

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			fmt.Println("read error:", err)
			continue
		}

		parts := strings.Fields(input)
		if len(parts) == 0 {
			printUsage()
			continue
		}

		command := parts[0]
		args := parts[1:]

		switch command {
		case "quit":
			fmt.Println("Exiting...")
			return

		case "ping":
			if len(args) != 1 {
				printUsage()
				continue
			}

			addr, err := net.ResolveUDPAddr("udp", args[0])
			if err != nil {
				fmt.Println("invalid address:", err)
				continue
			}

			client.ping(*addr)

		default:
			printUsage()
		}
	}
}
