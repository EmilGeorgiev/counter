# overview
The goal of this server is to efficiently handle a high number of concurrent client connections and process 
requests that increment a shared global counter. The server must:
- be High-performance request-response loops.
- Accept unsigned integer ( verify format) and Reject malformed or tampered requests
- Accept multiple client connections concurrently.
- Process simple uint64 increment requests.
- Be optimized for I/O-bound workloads.

## Architecture Overview

#### Custom binary protocol
The custom binary protocol with fixed message size (16 bytes) is used. The message format is:
- First 8 bytes: Unsigned integer (big-endian) to increment the counter.
- Next 8 bytes: xxhash.Sum64String of the string representation of the integer.

To prevent clients from sending arbitrary 8-byte values that could be interpreted as uint64, the server:
- Require the client to send an 8-byte hash (using xxhash.Sum64String) of the string form of the number.
- Server recomputes the hash and compares.
- This ensures that the number was intentionally crafted and not random garbage.

Main Reason of the protocol is to ensure that only valid uint64 values are accepted.
Strings like "a" or "aaaaaaaa" would be incorrectly parsed as integers if using naive byte interpretation.
By enforcing a strict protocol and verifying the checksum (xxhash of the string form), the server ensures type
safety over the wire.

#### Concurrency Model
- Uses a worker pool pattern. 
- Number of workers: runtime.NumCPU() * 20 to balance concurrency and throughput.
- Each worker handles a net.Conn, reading, verifying, and processing the message.

## Why a Custom Protocol?
Using a custom TCP protocol avoids the overhead and complexity of using HTTP, gRPC or something else:
- No need for HTTP headers or JSON parsing.
- No need to define protobuf schemas.
- Lower latency, smaller payloads, faster processing.

## IO-Bound Optimization
The server is I/O bound, not CPU-bound.
Heavy concurrency using goroutines and worker channels to ensure efficient handling of blocking network reads/writes.

Uses buffered readers and atomic operations to reduce context switches and mutex contention.

## Fault Handling
Graceful shutdown via OS signal capture.
Handles EOF, read/write errors, and closes client connections cleanly.

## CLI flags
The server and client both accept CLI flags to optimize runtime behavior, such as the number of workers (--multiplier). 
This allows flexible scaling depending on the environment. When the server is running alone on a machine 
(i.e., the client is on a separate machine), the server has access to more CPU and memory resources, resulting in a 
higher number of processed requests per second. In contrast, if the server and client run on the same machine, they 
compete for CPU resources, and performance may be impacted. In this case, tuning the --multiplier flag becomes essential 
to balance server-side concurrency with system load. These flags make the system adaptable for different test setups: 
isolated server benchmarking, local client/server stress tests, or real-world deployments.

## Start the client and the server
Start the server:
```text
go run server/server.go
```
optionally you can provide flags --multiplier (Multiplier for number of workers per CPU) and --port ( specify on which port to listen)

Start the client:
```text
go run client/client.go
```
optionally you can provide flags --multiplier (Multiplier for number of workers per CPU) and --serverAddress ( the address of the server)
The client will start multiple workers that will connect to the server and send as request integer 1. 
Wait for the client to finish. Then stop the server:
```text
Control + C
```
and you will in the terminal the counter value. The number of the counter is equal to the number of requests that are processed
ny thr server because every request contains 1 as a payload

## Performance Benchmark
System Specs: 
```text
Chip: Apple M3 Pro
Cores: 12 total (6 performance cores, 6 efficiency cores)
Memory: 18 GB RAM
OS: macOS
Go version: go1.24.1 darwin/arm64    
```
Benchmark Configuration:
```text
Server and client running on the same machine
GOMAXPROCS=12 (default)
--multiplier=20 (worker threads per core for the server + 20 for the client)
Each client request sends a uint64 value and receives the updated counter
```
Result:
```text
Server stopped after processing total requests: 16,423,170    
```



