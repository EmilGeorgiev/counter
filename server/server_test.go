package main

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/cespare/xxhash/v2"
)

func TestServer_ClientSendUint64(t *testing.T) {
	go runTCPServer(1, Port)
	time.Sleep(1 * time.Second) // wait the server to start
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Connection failed: %v\n", err)
	}
	defer conn.Close()

	number := uint64(10)
	p := createPayload(number)
	resp := write(t, conn, p)
	actual := binary.BigEndian.Uint64(resp)

	equal(t, number, actual)
	equal(t, number, counter)

	// send second number:
	number2 := uint64(20)
	p2 := createPayload(number2)
	resp = write(t, conn, p2)
	actual = binary.BigEndian.Uint64(resp)

	equal(t, number+number2, actual)
	equal(t, number+number2, counter)
}

func TestServer_ClientSendString(t *testing.T) {
	go runTCPServer(1, Port)
	time.Sleep(1 * time.Second) // wait the server to start
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Connection failed: %v\n", err)
	}
	defer conn.Close()

	str := "aaaaaaa"
	sum := xxhash.Sum64String(str)
	checkSum := make([]byte, 8)
	binary.BigEndian.PutUint64(checkSum, sum)

	reqPayload := append([]byte(str), checkSum...)
	connShouldBeClosedAfterSendTheRequest(t, conn, reqPayload)

	equal(t, 0, counter)
}

func TestServer_ClientSendWrongCheckSumForUint64(t *testing.T) {
	go runTCPServer(1, Port)
	time.Sleep(1 * time.Second) // wait the server to start
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Connection failed: %v\n", err)
	}
	defer conn.Close()

	number := uint64(55)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, number)

	wrongCheckSum := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	reqPayload := append(buf, wrongCheckSum...)
	connShouldBeClosedAfterSendTheRequest(t, conn, reqPayload)

	equal(t, 0, counter)
}

func TestServer_ClientSendIntegerString(t *testing.T) {
	go runTCPServer(1, Port)
	time.Sleep(1 * time.Second) // wait the server to start
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Connection failed: %v\n", err)
	}
	defer conn.Close()

	sum := xxhash.Sum64String("123")
	checkSum := make([]byte, 8)
	binary.BigEndian.PutUint64(checkSum, sum)

	reqPayload := append([]byte("123"), checkSum...)
	connShouldBeClosedAfterSendTheRequest(t, conn, reqPayload)

	equal(t, 0, counter)
}

func TestServer_ClientSendStringButValidCheckSumForItsIntegerRepresentation(t *testing.T) {
	go runTCPServer(1, Port)
	time.Sleep(1 * time.Second) // wait the server to start
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Connection failed: %v\n", err)
	}
	defer conn.Close()

	number := uint64(7016996765293437281)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, number)

	sum := xxhash.Sum64String(strconv.FormatUint(number, 10))
	checkSum := make([]byte, 8)
	binary.BigEndian.PutUint64(checkSum, sum)

	resp := write(t, conn, append([]byte("aaaaaaaa"), checkSum...))
	actual := binary.BigEndian.Uint64(resp)

	equal(t, number, actual)
	equal(t, number, counter)
}

func equal(t *testing.T, expected uint64, actual uint64) {
	if expected != actual {
		t.Errorf("Expected: %d", expected)
		t.Errorf("Actual  : %d", actual)
	}
}

func createPayload(number uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, number)

	sum := xxhash.Sum64String(strconv.FormatUint(number, 10))
	checkSum := make([]byte, 8)
	binary.BigEndian.PutUint64(checkSum, sum)

	return append(buf, checkSum...)
}

func write(t *testing.T, conn net.Conn, p []byte) []byte {
	if _, err := conn.Write(p); err != nil {
		t.Fatalf("Write error: %v\n", err)
	}

	resp := make([]byte, 8)
	if _, err := conn.Read(resp); err != nil {
		t.Fatalf("Read error: %v\n", err)
	}
	return resp
}

func connShouldBeClosedAfterSendTheRequest(t *testing.T, conn net.Conn, p []byte) {
	if _, err := conn.Write(p); err != nil {
		t.Fatalf("Write error: %v\n", err)
		return
	}

	resp := make([]byte, 8)
	_, err := conn.Read(resp)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Expected error io.EOF but got: %v\n", err)
	}
}
