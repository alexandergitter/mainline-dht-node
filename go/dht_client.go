package main

import "fmt"

type dhtClient struct {
	thisNodeInfo dhtNode
	routingTable routingTable
}

func (c *dhtClient) handleMessage(message krpcMessage) krpcMessage {
	fmt.Println(message)

	switch message.(type) {
	case krpcQuery:
		return krpcError{
			transactionId: message.(krpcQuery).transactionId,
			code:          KrpcErrorUnknownMethod,
			message:       "Unsupported method",
		}
	}

	return nil
}
