package benchmark

import (
	"encoding/binary"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"net"
	"strconv"
	"testing"
)

//func BenchmarkBytes(b *testing.B) {
//	for i := 0; i < b.N; i++ {
//		buf := make([]byte, 8)
//		binary.BigEndian.PutUint64(buf, 1)
//		_ = binary.BigEndian.Uint64(buf)
//	}
//}
//
//func BenchmarkString(b *testing.B) {
//	for i := 0; i < b.N; i++ {
//		_, _ = strconv.Atoi("1")
//	}
//}

func TestBytes(t *testing.T) {
	payloadBytes := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	hashBytes := []byte{11, 12, 13, 14, 15, 16, 17, 18}
	message := append(payloadBytes, hashBytes...)
	//message = append(message, []byte{1}...)
	fmt.Println(cap(message))

	b := message[:8]

	fmt.Println("cap(b):", cap(b))
	fmt.Println("len(b):", len(b))
	fmt.Println(b)

	b2 := message[8:]

	fmt.Println("cap(b2):", cap(b2))
	fmt.Println("len(b2):", len(b2))
	fmt.Println(b2)
}

func TestHashIntAndUint64(t *testing.T) {

	a := xxhash.Sum64String("a")
	fmt.Println(a)
	b := xxhash.Sum64String(strconv.FormatUint(uint64(97), 10))
	fmt.Println(b)
}

func TestHsh(t *testing.T) {
	str1 := strconv.FormatUint(uint64(7016996765293437281), 10)
	hash1 := xxhash.Sum64String(str1)
	fmt.Println(hash1)

	hash2 := xxhash.Sum64String("aaaaaaaa")
	fmt.Println(hash2)
}

func TestServ(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		panic(err)
		return
	}
	defer conn.Close()

	number := uint64(7016996765293437281)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, number)
	fmt.Println("number in bytes:", buf)

	sum := xxhash.Sum64String("7016996765293437281")
	checkSum := make([]byte, 8)
	binary.BigEndian.PutUint64(checkSum, sum)

	_, err = conn.Write(append(buf, checkSum...))
	if err != nil {
		fmt.Printf("Write error: %v\n", err)
		panic(err)
		return
	}

}

func TestServ2(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		panic(err)
		return
	}
	defer conn.Close()

	sum := xxhash.Sum64String("7016996765293437281")
	checkSum := make([]byte, 8)
	binary.BigEndian.PutUint64(checkSum, sum)

	fmt.Println([]byte("7016996765293437281"))
	_, err = conn.Write(append([]byte("7016996765293437281"), checkSum...))
	if err != nil {
		fmt.Printf("Write error: %v\n", err)
		panic(err)
		return
	}

}
