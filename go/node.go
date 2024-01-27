package main

import (
	"bytes"
	"fmt"
	"math/bits"
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

func (n nodeId) String() string {
	var builder strings.Builder
	for _, b := range n {
		builder.WriteString(fmt.Sprintf("%02x", b))
	}
	return builder.String()
}

// This could benefit from some SIMD instructions
func commonPrefixLength(a, b nodeId) int {
	var result int
	for i := 0; i < 20; i++ {
		if a[i] != b[i] {
			result += bits.LeadingZeros8(a[i] ^ b[i])
			break
		}
		result += 8
	}
	return result
}

type dhtNode struct {
	nodeId  nodeId
	address net.UDPAddr
}

func (n dhtNode) String() string {
	return fmt.Sprintf("dhtNode{%s}", n.nodeId)
}

// Helpers

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
