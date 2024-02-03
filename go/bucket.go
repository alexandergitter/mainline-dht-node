package main

import (
	"fmt"
	"strings"
)

type bucket struct {
	bucketSize int
	occupied   []bool
	entries    []dhtNode
}

func newBucket(bucketSize int) bucket {
	return bucket{
		bucketSize: bucketSize,
		occupied:   make([]bool, bucketSize),
		entries:    make([]dhtNode, bucketSize),
	}
}

func (b bucket) addEntry(entry dhtNode) (updated bucket, success bool) {
	if b.containsNodeId(entry.nodeId) {
		return b, true
	}

	for i, occupied := range b.occupied {
		if !occupied {
			b.occupied[i] = true
			b.entries[i] = entry
			return b, true
		}
	}

	return b, false
}

func (b bucket) containsNodeId(id nodeId) bool {
	for i, occupied := range b.occupied {
		if occupied && b.entries[i].nodeId.isEqual(id) {
			return true
		}
	}

	return false
}

func (b bucket) getEntryByIdOrReturnAll(id nodeId) (result []dhtNode, exactMatch bool) {
	result = make([]dhtNode, 0, b.bucketSize)

	for i, occupied := range b.occupied {
		if occupied {
			result = append(result, b.entries[i])

			if b.entries[i].nodeId.isEqual(id) {
				return result[len(result)-1:], true
			}
		}
	}

	return result, false
}

func (b bucket) splitAt(bitPosition int) (zeroBucket bucket, oneBucket bucket) {
	if bitPosition < 0 || bitPosition > 159 {
		panic("Invalid bit position")
	}

	zeroBucket = newBucket(b.bucketSize)
	oneBucket = newBucket(b.bucketSize)

	for i, occupied := range b.occupied {
		if !occupied {
			continue
		}

		if b.entries[i].nodeId.isBitSet(bitPosition) {
			oneBucket.occupied[i] = true
			oneBucket.entries[i] = b.entries[i]
		} else {
			zeroBucket.occupied[i] = true
			zeroBucket.entries[i] = b.entries[i]
		}
	}

	return zeroBucket, oneBucket
}

func (b bucket) String() string {
	var builder strings.Builder
	for i, occupied := range b.occupied {
		if occupied {
			builder.WriteString(fmt.Sprintf("%s ", b.entries[i].nodeId))
		} else {
			builder.WriteString("---------------------------------------- ")
		}
	}
	return builder.String()
}
