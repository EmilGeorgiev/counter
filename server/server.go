package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
)

var counter uint64

// Worker function to process requests
func (s *server) startWorker() {
	defer s.wg.Done()
	for {
		select {
		case conn, ok := <-s.jobs:
			if !ok {
				return
			}
			s.addConn <- conn
			handleConnection(conn)
			s.removeConn <- conn
		case <-s.quit:
			return
		}
	}
}

// Handles an individual client connection
func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		buf := make([]byte, 16)
		_, err := reader.Read(buf)
		if err != nil {
			// Handle other read errors
			if err == io.EOF {
				fmt.Println("Connection closed by client")
			} else {
				fmt.Println("Read error:", err)
			}
			return
		}

		number := binary.BigEndian.Uint64(buf[:8])
		str := strconv.FormatUint(number, 10)
		expectedCheckSum := xxhash.Sum64String(str)
		checkSum := binary.BigEndian.Uint64(buf[8:])

		if checkSum != expectedCheckSum {
			fmt.Printf("Unexpected checkSum: %d for unasign integer: %d. Close the connection\n", checkSum, number)
			fmt.Println("Close the connection")
			return
		}

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
	port := flag.String("port", ":8080", "TCP port to listen on")
	multiplier := flag.Int("multiplier", 20, "Multiplier for number of workers per CPU")

	flag.Parse()

	numWorkers := runtime.NumCPU() * *multiplier

	fmt.Println("00000000")
	s := newServer(numWorkers)
	fmt.Println("111111")
	s.start(*port)
	fmt.Println("222222")

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	fmt.Println("33333333")
	<-signalCh
	fmt.Println("44444444")
	s.stop()
	//runTCPServer(numWorkers, *port)
	fmt.Println("Server stopped after processing total requests:", counter)
}

type server struct {
	maxConn           int
	activeConnections map[net.Conn]bool
	quit              chan struct{}
	addConn           chan net.Conn
	removeConn        chan net.Conn
	jobs              chan net.Conn
	listener          net.Listener
	wg                sync.WaitGroup
	closeOnce         sync.Once
}

func newServer(maxConn int) *server {
	return &server{
		maxConn:           maxConn,
		activeConnections: make(map[net.Conn]bool),
		quit:              make(chan struct{}),
		addConn:           make(chan net.Conn, maxConn),
		removeConn:        make(chan net.Conn, maxConn),
		jobs:              make(chan net.Conn),
		wg:                sync.WaitGroup{},
	}
}

func (s *server) start(port string) {
	s.mantainsConnPool()
	listener, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	s.listener = listener
	fmt.Println("Server is listening on port: ", port)

	fmt.Println("Starting with ", s.maxConn, "workers")
	for i := 0; i < s.maxConn; i++ {
		s.wg.Add(1)
		go s.startWorker()
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, er := listener.Accept()
			if er != nil {
				select {
				case <-s.quit:
					fmt.Println("Close the listener")
					return
				default:
					fmt.Println("Connection error:", er)
					continue
				}
			}
			// Send the connection to the worker pool
			select {
			case s.jobs <- conn:
			case <-s.quit:
				conn.Close()
				return
			}
		}
	}()
}

func (s *server) stop() {
	s.closeOnce.Do(func() {
		fmt.Println("Stopping server...")

		// Stop listener
		close(s.quit)
		s.listener.Close()

		// Close job queue and wait for workers
		close(s.jobs)
		s.wg.Wait()
		fmt.Println("Server fully stopped.")
	})
}

func (s *server) mantainsConnPool() {
	go func() {
		for {
			select {
			case <-s.quit:
				for conn, _ := range s.activeConnections {
					conn.Close()
				}
				fmt.Println("All active connections are closed.")
				return
			case conn := <-s.addConn:
				fmt.Println("add new connection to the pool")
				s.activeConnections[conn] = true
			case conn := <-s.removeConn:
				fmt.Println("remove connection from the pool")
				delete(s.activeConnections, conn)
			}
		}
	}()
}

//func runTCPServer(maxConn int, port string) {
//	listener, err := net.Listen("tcp", port)
//	if err != nil {
//		panic(err)
//	}
//	defer listener.Close()
//	fmt.Println("Server is listening on port: ", port)
//
//	quit := make(chan struct{})
//	// Worker pool
//	jobs := make(chan net.Conn, maxConn)
//	var wg sync.WaitGroup
//
//	fmt.Println("Starting with ", maxConn, "workers")
//	for i := 0; i < maxConn; i++ {
//		wg.Add(1)
//		go worker(jobs, &wg)
//	}
//
//	signalCh := make(chan os.Signal, 1)
//	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
//
//	go func() {
//		for {
//			conn, er := listener.Accept()
//			if er != nil {
//				select {
//				case <-quit:
//					fmt.Println("Close the listener")
//					return
//				default:
//					fmt.Println("Connection error:", er)
//					continue
//				}
//			}
//			// Send the connection to the worker pool
//			jobs <- conn
//		}
//	}()
//
//	<-signalCh
//	close(quit)
//	listener.Close()
//	close(jobs)
//	wg.Wait()
//}

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
