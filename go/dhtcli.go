package main

import (
	"crypto/rand"
	"fmt"
	"log"
)

func main() {
	var nodeId = make([]byte, 20)
	_, err := rand.Read(nodeId)
	if err != nil {
		log.Fatal("Could not generate random node ID", err)
	}

	fmt.Println(nodeId)
}
