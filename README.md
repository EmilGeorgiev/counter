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


