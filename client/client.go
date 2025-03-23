package main

import (
	"encoding/binary"
	"fmt"
	xxhash "github.com/cespare/xxhash/v2"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	ServerAddress = "127.0.0.1:8080" // Adjust if needed
)

func clientWorker(id int, wg *sync.WaitGroup, startCh, stopCh <-chan struct{}) {
	defer wg.Done()

	// Open a persistent TCP connection
	conn, err := net.Dial("tcp", ServerAddress)
	if err != nil {
		fmt.Printf("Client %d: Connection failed: %v\n", id, err)
		return
	}
	defer conn.Close()

	number := uint64(1)
	payloadBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(payloadBytes, number)

	str := strconv.FormatUint(number, 10)
	sum := xxhash.Sum64String(str)
	hashBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(hashBytes, sum)
	<-startCh
	for {
		select {
		case <-stopCh: // Stop when time is up
			fmt.Printf("Client %d stopping...\n", id)
			return
		default:
			_, err = conn.Write(append(payloadBytes, hashBytes...))
			if err != nil {
				fmt.Printf("Client %d: Write error: %v\n", id, err)
				return
			}

			resp := make([]byte, 8)
			_, err = conn.Read(resp)
			if err != nil {
				fmt.Println("Read error:", err)
				return
			}
		}
	}
}

func main() {
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU() * 50 // Optimize for I/O-bound workload

	fmt.Println("Starting", numWorkers, "client workers...")

	startCh := make(chan struct{}) // Channel to signal stopping
	stopCh := make(chan struct{})  // Channel to signal stopping
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go clientWorker(i, &wg, startCh, stopCh)
	}

	time.Sleep(1 * time.Second) // wait and give time to all the workers to start
	close(startCh)
	time.Sleep(1 * time.Minute) // Stop after 1 minute
	close(stopCh)

	wg.Wait() // Wait for all workers to finish
	fmt.Println("All clients finished after 1 minute.")
}
