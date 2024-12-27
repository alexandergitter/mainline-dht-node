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

type envelope struct {
	address *net.UDPAddr
	message krpcMessage
}

func receiver(conn *net.UDPConn, data chan envelope) {
	buffer := make([]byte, 65535)

	for {
		fmt.Println("Waiting for messages...")

		n, udp, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("Received", n, "bytes", "from", udp)

		// TODO: If this is invalid bencode, we should reply with an error response. For now just log and continue
		dict, err := decodeBencodeDict(string(buffer[:n]))
		if err != nil {
			fmt.Println(err)
			continue
		}

		// TODO: If this is an invalid KRPC message, we should send an error response. For now, just log and continue
		krpc, err := decodeKrpcMessage(dict)
		if err != nil {
			fmt.Println(err)
			continue
		}

		data <- envelope{
			address: udp,
			message: krpc,
		}
	}
}

func sender(conn *net.UDPConn, data chan envelope) {
	for {
		select {
		case env := <-data:
			_, err := conn.WriteToUDP([]byte(env.message.encode()), env.address)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func main() {
	var ownId = nodeId{}
	_, err := rand.Read(ownId[:])
	if err != nil {
		log.Fatalf("error while generating node id: %s", err)
	}

	listenOn, err := net.ResolveUDPAddr("udp", "127.0.0.1:6880")
	conn, err := net.ListenUDP("udp", listenOn)
	if err != nil {
		panic(err)
	}

	var myNodeInfo = dhtNode{
		nodeId:  ownId,
		address: *listenOn,
	}
	var client = dhtClient{
		thisNodeInfo: myNodeInfo,
		routingTable: newRoutingTable(ENTRIES, myNodeInfo),
	}

	senderChannel := make(chan envelope)
	receiverChannel := make(chan envelope)

	go receiver(conn, receiverChannel)
	go sender(conn, senderChannel)

	for {
		select {
		case env := <-receiverChannel:
			fmt.Println(env)
			var response = client.handleMessage(env.message)
			if response != nil {
				senderChannel <- envelope{
					address: listenOn,
					message: response,
				}
			}
		default:
			fmt.Scanln()
		}
	}
}
