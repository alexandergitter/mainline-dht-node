package main

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type nodeId [20]byte

func (n nodeId) isBitSet(index int) bool {
	return (n[index/8] & (1 << uint(7-index%8))) != 0
}

func (n nodeId) isEqual(other nodeId) bool {
	return bytes.Equal(n[:], other[:])
}

type dhtNode struct {
	nodeId  nodeId
	address net.UDPAddr
}

// Helpers

func nodeIdToString(n nodeId) string {
	var result = make([]string, 20)
	for i, b := range n {
		result[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(result, "")
}

func hexStringToNodeId(s string) (nodeId, error) {
	var result nodeId

	if len(s) != 40 {
		return result, fmt.Errorf("Invalid node ID length")
	}

	for i := 0; i < 20; i++ {
		if b, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8); err != nil {
			return result, err
		} else {
			result[i] = byte(b)
		}
	}

	return result, nil
}
