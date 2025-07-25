package main

import (
	"fmt"
	"net"
)

type dhtClient struct {
	thisNodeInfo nodeInfo
	routingTable *routingTable
	krpcRuntime  *krpcRuntime
}

func startDhtClient(thisNodeInfo nodeInfo, routingTable *routingTable, listenOn *net.UDPAddr) *dhtClient {
	var krpcRuntime = newKrpcRuntime(listenOn)
	var dhtClient = &dhtClient{
		thisNodeInfo: thisNodeInfo,
		routingTable: routingTable,
		krpcRuntime:  krpcRuntime,
	}

	krpcRuntime.start(dhtClient)

	return dhtClient
}

func (c *dhtClient) handlePing(args bencodeDict) krpcMessage {
	id, ok := args["id"]
	if !ok {
		return &krpcError{
			code:    KrpcErrorProtocol,
			message: "Missing 'id' argument",
		}
	}

	nodeId, ok := id.(bencodeString)
	if !ok || len(nodeId) != 20 {
		return &krpcError{
			code:    KrpcErrorProtocol,
			message: "Invalid 'id' argument",
		}
	}

	return &krpcResponse{
		returnValues: bencodeDict{
			"id": bencodeString(c.thisNodeInfo.nodeId[:]),
		},
	}
}

var handlerFunctions = map[string]func(*dhtClient, bencodeDict) krpcMessage{
	"ping": (*dhtClient).handlePing,
}

func (c *dhtClient) handleQuery(message *krpcQuery) krpcMessage {
	fmt.Println(message)

	handler, ok := handlerFunctions[message.methodName]
	if !ok {
		return &krpcError{
			code:    KrpcErrorUnknownMethod,
			message: fmt.Sprintf("Unknown method '%s'", message.methodName),
		}
	}

	return handler(c, message.arguments)
}

func (c *dhtClient) ping(dest nodeInfo) (krpcMessage, error) {
	var msg = krpcQuery{
		methodName: "ping",
		arguments:  bencodeDict{"id": bencodeString(c.thisNodeInfo.nodeId[:])},
	}

	return c.krpcRuntime.doRequest(&dest.address, msg)
}
