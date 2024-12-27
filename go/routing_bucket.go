package main

import (
	"fmt"
	"strings"
)

type bucket struct {
	bucketSize int
	entries    []nodeInfo
}

func newBucket(bucketSize int) bucket {
	return bucket{
		bucketSize: bucketSize,
		entries:    make([]nodeInfo, 0, bucketSize),
	}
}

func (b bucket) addEntry(entry nodeInfo) (updated bucket, success bool) {
	if b.containsNodeId(entry.nodeId) {
		return b, true
	}

	if len(b.entries) >= b.bucketSize {
		return b, false
	}

	b.entries = append(b.entries, entry)
	return b, true
}

func (b bucket) containsNodeId(id nodeId) bool {
	for _, entry := range b.entries {
		if entry.nodeId.isEqual(id) {
			return true
		}
	}

	return false
}

func (b bucket) getEntryByIdOrReturnAll(id nodeId) (result []nodeInfo, exactMatch bool) {
	for i, entry := range b.entries {
		if entry.nodeId.isEqual(id) {
			return b.entries[i : i+1], true
		}
	}

	return b.entries, false
}

func (b bucket) splitAt(bitPosition int) (zeroBucket bucket, oneBucket bucket) {
	if bitPosition < 0 || bitPosition > 159 {
		panic("Invalid bit position")
	}

	zeroBucket = newBucket(b.bucketSize)
	oneBucket = newBucket(b.bucketSize)

	for _, entry := range b.entries {
		if entry.nodeId.isBitSet(bitPosition) {
			oneBucket, _ = oneBucket.addEntry(entry)
		} else {
			zeroBucket, _ = zeroBucket.addEntry(entry)
		}
	}

	return zeroBucket, oneBucket
}

func (b bucket) String() string {
	var builder strings.Builder

	for i := range b.bucketSize {
		if i >= len(b.entries) {
			builder.WriteString("---------------------------------------- ")
		} else {
			builder.WriteString(fmt.Sprintf("%s ", b.entries[i].nodeId))
		}
	}

	return builder.String()
}
