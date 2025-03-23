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
			s.handleConnection(conn)
			s.removeConn <- conn
		case <-s.quit:
			return
		}
	}
}

// Handles an individual client connection
func (s *server) handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		buf := make([]byte, 16)
		_, err := reader.Read(buf)
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
			}
			// Handle other read errors
			if err != io.EOF {
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
	s := newServer(numWorkers)
	s.start(*port)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	<-signalCh
	s.stop()
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
		addConn:           make(chan net.Conn, maxConn*2),
		removeConn:        make(chan net.Conn, maxConn*2),
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
	s.wg.Add(1)
	go func() {
		s.wg.Done()
		for {
			select {
			case <-s.quit:
				for conn, _ := range s.activeConnections {
					conn.Close()
				}
				fmt.Println("All active connections are closed.")
				return
			case conn := <-s.addConn:
				s.activeConnections[conn] = true
			case conn := <-s.removeConn:
				conn.Close()
				delete(s.activeConnections, conn)
			}
		}
	}()
}
