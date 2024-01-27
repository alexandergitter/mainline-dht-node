package main

import "fmt"

type routingTable struct {
	ownNodeId nodeId
	table     []bucket
}

func newRoutingTable(bucketSize int, ownNodeId nodeId) routingTable {
	// Technically we don't need al 160 buckets, since there are only 8 nodes with common
	// longest prefix length of 157, so that bucket will never be split.
	var initialTable = make([]bucket, 0, 160)
	initialTable = append(initialTable, newBucket(bucketSize))

	return routingTable{
		ownNodeId: ownNodeId,
		table:     initialTable,
	}
}

func (t *routingTable) addEntry(entry dhtNode) {
	var currentMaxPrefixLength = len(t.table) - 1
	var prefixLength = commonPrefixLength(t.ownNodeId, entry.nodeId)
	var bucketIndex = min(prefixLength, currentMaxPrefixLength)
	var bucket = t.table[bucketIndex]
	var updatedBucket, success = bucket.addEntry(entry)

	if success {
		t.table[bucketIndex] = updatedBucket
		return
	}

	// At this point we know that the bucket is full, so check if we can split it.
	// Option 1: New entry does not fall into the bucket our own node is currently in, so just drop it.
	if prefixLength < currentMaxPrefixLength {
		return
	}

	// Option 2:
	// New entry falls into the bucket our own node is currently in, so split the bucket, and append the
	// one with the longer common prefix to the table.
	var zeroBucket, oneBucket = bucket.splitAt(bucketIndex)
	if t.ownNodeId.isBitSet(bucketIndex) {
		// Own bit is set, so the common prefix for the zero bucket ends here, and it stays behind at bucketIndex.
		t.table[bucketIndex] = zeroBucket
		t.table = append(t.table, oneBucket)
	} else {
		t.table[bucketIndex] = oneBucket
		t.table = append(t.table, zeroBucket)
	}
	// Now that we set up the new buckets, we can try to add the entry again.
	t.addEntry(entry)
}

func printRoutingTable(table routingTable) {
	for i, bucket := range table.table {
		fmt.Printf("%3d: ", i)
		for j, entry := range bucket.entries {
			if bucket.occupied[j] {
				fmt.Printf("%s ", entry.nodeId)
			} else {
				fmt.Printf("-------- ")
			}
		}
		fmt.Println()
	}
}
