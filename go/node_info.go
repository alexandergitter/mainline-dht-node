package main

import (
	"bytes"
	"encoding/binary"
	"errors"
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

type nodeInfo struct {
	nodeId  nodeId
	address net.UDPAddr
}

func (n nodeInfo) compactNodeInfo() string {
	var buffer = make([]byte, 0, 26)
	buffer = append(buffer, n.nodeId[:]...)
	buffer = append(buffer, n.address.IP.To4()...)
	buffer = binary.BigEndian.AppendUint16(buffer, uint16(n.address.Port))
	return string(buffer)
}

func (n nodeInfo) String() string {
	return fmt.Sprintf("nodeInfo{%s}", n.nodeId)
}

func decodeCompactNodeInfo(data string) (nodeInfo, error) {
	if len(data) != 26 {
		return nodeInfo{}, errors.New("Invalid compact node info length - expected 26 bytes")
	}

	return nodeInfo{
		nodeId: nodeId([]byte(data[:20])),
		address: net.UDPAddr{
			IP:   net.IPv4(data[20], data[21], data[22], data[23]),
			Port: int(binary.BigEndian.Uint16([]byte(data[24:]))),
		},
	}, nil
}

// Helpers

func bytesToHexString(b []byte) string {
	var builder strings.Builder
	for _, v := range b {
		builder.WriteString(fmt.Sprintf("%02x", v))
	}
	return builder.String()
}

func hexStringToBytes(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, errors.New("Hex string must have an even length")
	}

	result := make([]byte, len(s)/2)
	for i := 0; i < len(s)/2; i++ {
		if b, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8); err != nil {
			return nil, err
		} else {
			result[i] = byte(b)
		}
	}

	return result, nil
}

func hexStringToNodeId(s string) (nodeId, error) {
	if len(s) != 40 {
		return nodeId(make([]byte, 20)), errors.New("Invalid hex string length")
	}

	var idBytes, err = hexStringToBytes(s)
	if err != nil {
		return nodeId(make([]byte, 20)), err
	}

	return nodeId(idBytes), nil
}
