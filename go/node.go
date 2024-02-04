package main

import (
	"bytes"
	"encoding/binary"
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
	return bytesToHexString(n[:])
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

func (n dhtNode) compactNodeInfo() string {
	var buffer = make([]byte, 0, 26)
	buffer = append(buffer, n.nodeId[:]...)
	buffer = append(buffer, n.address.IP.To4()...)
	buffer = binary.BigEndian.AppendUint16(buffer, uint16(n.address.Port))
	return string(buffer)
}

func (n dhtNode) String() string {
	return fmt.Sprintf("dhtNode{%s}", n.nodeId)
}

func decodeCompactNodeInfo(data string) dhtNode {
	if len(data) != 26 {
		panic("Invalid compact node info length")
	}

	return dhtNode{
		nodeId: nodeId([]byte(data[:20])),
		address: net.UDPAddr{
			IP:   net.IPv4(data[20], data[21], data[22], data[23]),
			Port: int(binary.BigEndian.Uint16([]byte(data[24:]))),
		},
	}
}

// Helpers

func bytesToHexString(b []byte) string {
	var builder strings.Builder
	for _, v := range b {
		builder.WriteString(fmt.Sprintf("%02x", v))
	}
	return builder.String()
}

func hexStringToBytes(s string) []byte {
	if len(s)%2 != 0 {
		panic("Invalid hex string length")
	}

	result := make([]byte, len(s)/2)
	for i := 0; i < len(s)/2; i++ {
		if b, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8); err != nil {
			panic(err)
		} else {
			result[i] = byte(b)
		}
	}
	return result
}

func hexStringToNodeId(s string) nodeId {
	if len(s) != 40 {
		panic("Invalid hex string length")
	}

	return nodeId(hexStringToBytes(s))
}
