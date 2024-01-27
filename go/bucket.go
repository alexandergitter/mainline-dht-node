package main

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
