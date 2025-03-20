package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"runtime"
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

	//writer := bufio.NewWriter(conn)
	//reader := bufio.NewScanner(conn)

	<-startCh
	for {
		select {
		case <-stopCh: // Stop when time is up
			fmt.Printf("Client %d stopping...\n", id)
			return
		default:
			// Send "1" to the server
			number := uint64(1)
			buf := make([]byte, 8)
			binary.BigEndian.PutUint64(buf, number)

			_, err = conn.Write(buf)
			if err != nil {
				fmt.Printf("Client %d: Write error: %v\n", id, err)
				return
			}

			//_, err := writer.WriteString("1\n")
			//if err != nil {
			//	fmt.Printf("Client %d: Write error: %v\n", id, err)
			//	return
			//}
			//writer.Flush()

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
