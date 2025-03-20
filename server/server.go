package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	Port = ":8080"
	//WorkerPool = 10 // Number of worker goroutines
)

var counter int64

// Worker function to process requests
func worker(jobs <-chan net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	for conn := range jobs {
		handleConnection(conn)
	}
}

// Handles an individual client connection
func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewScanner(conn)

	for reader.Scan() {
		data := reader.Text()
		_, err := strconv.Atoi(data) // Validate input
		if err != nil {
			fmt.Fprintf(conn, "Invalid number\n")
			continue
		}

		// Atomically increment the counter
		newValue := atomic.AddInt64(&counter, 1)

		// Send the updated counter value back to the client
		fmt.Fprintf(conn, "%d\n", newValue)
	}

	if err := reader.Err(); err != nil {
		fmt.Println("Connection error:", err)
	}
}

func main() {
	listener, err := net.Listen("tcp", Port)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Server is running on port: ", Port)

	// Worker pool
	jobs := make(chan net.Conn, 100)
	var wg sync.WaitGroup

	numWorkers := runtime.NumCPU() * 40
	fmt.Println("Starting with ", numWorkers, "workers")
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(jobs, &wg)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Connection error:", err)
				continue
			}

			// Send the connection to the worker pool
			jobs <- conn
		}
	}()

	<-signalCh
	fmt.Println("\nShutting down server...")
	close(jobs) // Close jobs when server shuts down
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
// 16 435 263
// 16 377 775

// Server 300 workers and client with 300 workers
// 16480410
// 16418252

// Server 480 workers and client with 480 workers
// 16320309
// 16388962
