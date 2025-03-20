package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	Port = ":8080"
	//WorkerPool = 10 // Number of worker goroutines
)

var counter uint64

// Worker function to process requests
func worker(connections <-chan net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	for conn := range connections {
		handleConnection(conn)
	}

}

// Handles an individual client connection
func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		buf := make([]byte, 8)
		_, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("connection is closed")
				return
			}
			fmt.Println("Read error:", err)
			return
		}

		number := binary.BigEndian.Uint64(buf)
		newValue := atomic.AddUint64(&counter, number)

		// Send the updated counter value back to the client
		response := make([]byte, 8)
		binary.BigEndian.PutUint64(response, newValue)
		_, err = conn.Write(response)
		if err != nil {
			if err == io.EOF {
				fmt.Println("connection is closed")
				return
			}
			fmt.Println("Write error:", err)
			return
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", Port)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Server is listening on port: ", Port)

	numWorkers := runtime.NumCPU() * 50
	// Worker pool
	jobs := make(chan net.Conn, numWorkers)
	var wg sync.WaitGroup

	fmt.Println("Starting with ", numWorkers, "workers")
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(jobs, &wg)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			conn, er := listener.Accept()
			if er != nil {
				fmt.Println("Connection error:", er)
				continue
			}
			// Send the connection to the worker pool
			jobs <- conn
		}
	}()

	<-signalCh
	close(jobs)
	wg.Wait()
	fmt.Println("Server stopped after processing total requests:", counter)
}

// Server 24 workers and client with 24 workers
// 1 execution: 14 210 940
// 2 execution: 14 264 364
// 3 execution: 14 072 899

// Server 24 workers and client with 12 workers
// 1 execution: 11 000 000
// 2 execution: ---
// 3 execution: ---

// Server 24 workers and client with 36 workers
// 1 execution: 14 147 658
// 2 execution: 14 662 367
// 3 execution: 14 121 892

// Server 36 workers and client with 36 workers
// 1 execution: 14 710 643
// 2 execution: 14 905 504
// 3 execution: 14803708
// 14832794
// 15444834
// 14959140
// 14971713

// Server 48 workers and client with 48 workers
// 1 execution: 15 490 077
// 2 execution: 15 550 697
// 3 execution: 15 469 991

// Server 60 workers and client with 68 workers
//15898545
// 15867473

// Server 72 workers and client with 72 workers
// 16108886
// 16065901
// 16069550

// Server 84 workers and client with 84 workers
// 16 141 911
// 16 181 873
// 16 092 664
//

// Server 96 workers and client with 96 workers
// 16088951
// 16068066
// 16166290

// Server 108 workers and client with 108 workers
// 16125822
// 16165981

// Server 240 workers and client with 240 workers
//   string      []byte
// 16 435 263   16 364 461
// 16 377 775   16 454 078

// Server 300 workers and client with 300 workers
// 16480410
// 16418252

// Server 480 workers and client with 480 workers
//  string      bytes
// 16320309   16000717
// 16388962   16347499
