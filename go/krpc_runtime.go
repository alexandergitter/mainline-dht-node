package main

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type krpcRuntime struct {
	pendingRequests     map[string]chan<- krpcMessage
	pendingRequestsLock sync.Mutex
	addr                *net.UDPAddr
	conn                *net.UDPConn
}

func newKrpcRuntime(listenOn *net.UDPAddr) *krpcRuntime {
	return &krpcRuntime{
		pendingRequests:     make(map[string]chan<- krpcMessage),
		pendingRequestsLock: sync.Mutex{},
		addr:                listenOn,
	}
}

func (k *krpcRuntime) enqueuePendingRequest(id string) <-chan krpcMessage {
	k.pendingRequestsLock.Lock()
	defer k.pendingRequestsLock.Unlock()

	var ch = make(chan krpcMessage, 1)
	k.pendingRequests[id] = ch
	return ch
}

func (k *krpcRuntime) cancelPendingRequest(id string) {
	k.pendingRequestsLock.Lock()
	defer k.pendingRequestsLock.Unlock()

	delete(k.pendingRequests, id)
}

func (k *krpcRuntime) dequeuePendingRequest(id string) (chan<- krpcMessage, bool) {
	k.pendingRequestsLock.Lock()
	defer k.pendingRequestsLock.Unlock()

	ch, ok := k.pendingRequests[id]
	delete(k.pendingRequests, id)
	return ch, ok
}

func (k *krpcRuntime) doRequest(dest *net.UDPAddr, msg krpcQuery) (krpcMessage, error) {
	var responseChannel = k.enqueuePendingRequest(msg.transactionId)
	_, err := k.conn.WriteToUDP([]byte(msg.encode()), dest)
	if err != nil {
		return nil, err
	}

	select {
	case msg := <-responseChannel:
		return msg, nil
	case <-time.After(time.Second * 5):
		return nil, errors.New("timeout")
	}
}

func (k *krpcRuntime) receiveMessages(handler dhtClient) {
	buffer := make([]byte, 65535)

	for {
		fmt.Println("Waiting for messages...")

		bytesReceived, srcAddr, err := k.conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Println("Received", bytesReceived, "bytes", "from", srcAddr)

		// TODO: If this is invalid bencode, we should reply with an error response. For now just log and continue
		dict, err := decodeBencodeDict(string(buffer[:bytesReceived]))
		if err != nil {
			fmt.Println(err)
			continue
		}

		// TODO: If this is an invalid KRPC message, we should send an error response. For now, just log and continue
		msg, err := decodeKrpcMessage(dict)
		if err != nil {
			fmt.Println(err)
			continue
		}

		switch msg.(type) {
		case krpcResponse:
			var id = msg.(krpcResponse).transactionId
			if ch, ok := k.dequeuePendingRequest(id); ok {
				ch <- msg
			}
		case krpcError:
			var id = msg.(krpcError).transactionId
			if ch, ok := k.dequeuePendingRequest(id); ok {
				ch <- msg
			}
		case krpcQuery:
			go func() {
				var response = handler.handleQuery(msg.(krpcQuery))
				if response != nil {
					_, err := k.conn.WriteToUDP([]byte(response.encode()), srcAddr)
					if err != nil {
						fmt.Println(err)
					}
				}
			}()
		}
	}
}

func (k *krpcRuntime) start(handler dhtClient) {
	conn, err := net.ListenUDP("udp", k.addr)
	if err != nil {
		panic(err)
	}

	k.conn = conn

	go k.receiveMessages(handler)
}
