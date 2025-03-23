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

// Worker function to process requests
func (s *server) startWorker() {
	defer s.wg.Done()
	for conn := range s.jobs {
		s.addConn <- conn
		s.handleConnection(conn)
		s.removeConn <- conn
	}
}

// Handles an individual client connection
func (s *server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		buf := make([]byte, 16)
		_, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[read error from %s]: %v\n", conn.RemoteAddr(), err)
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

		newValue := s.counter.Add(number)
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
	fmt.Println("Server stopped after processing total requests:", s.counter.Load())
}

type server struct {
	maxConn    int
	quit       chan struct{}
	addConn    chan net.Conn
	removeConn chan net.Conn
	jobs       chan net.Conn
	listener   net.Listener
	wg         sync.WaitGroup
	closeOnce  sync.Once
	counter    atomic.Uint64
}

func newServer(maxConn int) *server {
	return &server{
		maxConn:    maxConn,
		quit:       make(chan struct{}),
		addConn:    make(chan net.Conn, maxConn*2),
		removeConn: make(chan net.Conn, maxConn*2),
		jobs:       make(chan net.Conn),
		wg:         sync.WaitGroup{},
	}
}

func (s *server) start(port string) {
	if s.listener != nil {
		panic("server already started")
	}
	s.trackActiveConnections()
	listener, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	s.listener = listener

	for i := 0; i < s.maxConn; i++ {
		s.wg.Add(1)
		go s.startWorker()
	}

	fmt.Printf("Server Listening on port %s with %d workers\n", port, s.maxConn)

	s.wg.Add(1)
	go s.accept()
}

func (s *server) accept() {
	defer s.wg.Done()
	for {
		conn, er := s.listener.Accept()
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

func (s *server) trackActiveConnections() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		activeConnections := make(map[net.Conn]bool)
		for {
			select {
			case <-s.quit:
				for conn, _ := range activeConnections {
					conn.Close()
					delete(activeConnections, conn)
				}
				fmt.Println("All active connections are closed.")
				return
			case conn := <-s.addConn:
				activeConnections[conn] = true
			case conn := <-s.removeConn:
				delete(activeConnections, conn)
			}
		}
	}()
}
