package main

import "fmt"

type routingTable struct {
	thisNodeInfo nodeInfo
	bucketSize   int
	table        []bucket
}

func newRoutingTable(bucketSize int, thisNodeInfo nodeInfo) routingTable {
	// Technically we don't need all 160 buckets, since there are only 8 nodes with common
	// longest prefix length of 157, so bucket 157 will never be split.
	var initialTable = make([]bucket, 0, 160)
	initialTable = append(initialTable, newBucket(bucketSize))

	return routingTable{
		thisNodeInfo: thisNodeInfo,
		bucketSize:   bucketSize,
		table:        initialTable,
	}
}

func (t *routingTable) addEntry(entry nodeInfo) {
	var currentMaxPrefixLength = len(t.table) - 1
	var prefixLength = commonPrefixLength(t.thisNodeInfo.nodeId, entry.nodeId)
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
	// New entry falls into the bucket our own node is currently in, so split the bucket - leave the sub-bucket with the
	// shorter common prefix at this current index and append the one with the longer common prefix to the table (thereby
	// extending the prefix length the routing table covers).
	var zeroBucket, oneBucket = bucket.splitAt(bucketIndex)
	if t.thisNodeInfo.nodeId.isBitSet(bucketIndex) {
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

func (t *routingTable) findNode(targetId nodeId) (result []nodeInfo, exactMatch bool) {
	if t.thisNodeInfo.nodeId.isEqual(targetId) {
		return []nodeInfo{t.thisNodeInfo}, true
	}

	return t.findNodeWithoutSelf(targetId)
}

func (t *routingTable) findNodeWithoutSelf(targetId nodeId) (result []nodeInfo, exactMatch bool) {
	var currentMaxPrefixLength = len(t.table) - 1
	var prefixLength = commonPrefixLength(t.thisNodeInfo.nodeId, targetId)
	var startBucketIndex = min(prefixLength, currentMaxPrefixLength)
	result = make([]nodeInfo, 0, t.bucketSize)

	for offset := 0; (startBucketIndex-offset) > 0 || (startBucketIndex+offset) <= currentMaxPrefixLength; offset++ {
		var i = startBucketIndex - offset
		if i >= 0 {
			var entries, exactMatch = t.table[i].getEntryByIdOrReturnAll(targetId)

			if exactMatch {
				return entries, true
			}

			var maxSpaceLeft = t.bucketSize - len(result)
			var entriesToAppend = min(maxSpaceLeft, len(entries))
			result = append(result, entries[:entriesToAppend]...)
		}

		i = startBucketIndex + offset
		if offset > 0 && i <= currentMaxPrefixLength {
			var entries, exactMatch = t.table[i].getEntryByIdOrReturnAll(targetId)

			if exactMatch {
				return entries, true
			}

			var maxSpaceLeft = t.bucketSize - len(result)
			var entriesToAppend = min(maxSpaceLeft, len(entries))
			result = append(result, entries[:entriesToAppend]...)
		}

		if len(result) >= t.bucketSize {
			return result, false
		}
	}

	return result, false
}

func printRoutingTable(table routingTable) {
	for i, bucket := range table.table {
		fmt.Printf("%3d: %s", i, bucket)
		fmt.Println()
	}
}
