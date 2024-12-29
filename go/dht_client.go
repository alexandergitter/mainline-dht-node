package main

import "fmt"

type dhtClient struct {
	thisNodeInfo nodeInfo
	routingTable routingTable
}

func newDhtClient(thisNodeInfo nodeInfo, routingTable routingTable) dhtClient {
	return dhtClient{
		thisNodeInfo: thisNodeInfo,
		routingTable: routingTable,
	}
}

func (c *dhtClient) handleQuery(message krpcQuery) krpcMessage {
	fmt.Println(message)

	return krpcError{
		transactionId: message.transactionId,
		code:          KrpcErrorUnknownMethod,
		message:       "Unsupported method",
	}
}
