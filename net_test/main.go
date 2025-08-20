package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	kcp "github.com/xtaci/kcp-go/v5"
)

type Ping struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

func connectTCP(addr string) (net.Conn, error) {
	return net.Dial("tcp", addr)
}

func connectKCP(addr string) (net.Conn, error) {
	return kcp.Dial(addr)
}

func connectUDP(addr string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, udpAddr)
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run client.go <protocol> <addr:port> <numClients> <numMessages>")
		return
	}
	protocol := os.Args[1]
	addr := os.Args[2]
	numClients := atoi(os.Args[3])
	numMessages := atoi(os.Args[4])

	if protocol != "tcp" && protocol != "rudp" && protocol != "udp" {
		log.Fatal("Protocol must be either 'tcp', 'rudp' or 'udp'")
		return
	}

	var wg sync.WaitGroup
	times := sync.Map{} // uuid -> sendTime
	latencies := make([]time.Duration, 0, numClients*numMessages)
	lock := sync.Mutex{}

	start := time.Now()
	for c := 0; c < numClients; c++ {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()

			switch protocol {
			case "tcp", "rudp":
				var conn net.Conn
				var err error
				if protocol == "rudp" {
					conn, err = connectKCP(addr)
				} else {
					conn, err = connectTCP(addr)
				}
				if err != nil {
					fmt.Println("dial error:", err)
					return
				}
				defer conn.Close()

				decoder := json.NewDecoder(conn)
				for i := 0; i < numMessages; i++ {
					id := uuid.New().String()
					ping := Ping{Type: "ping", Id: id}
					data, _ := json.Marshal(ping)
					data = append(data, '\n')

					sendTime := time.Now()
					times.Store(id, sendTime)

					_, err := conn.Write(data)
					if err != nil {
						fmt.Println("write error:", err)
						return
					}

					var resp Ping
					if err := decoder.Decode(&resp); err != nil {
						fmt.Println("decode error:", err)
						return
					}

					if val, ok := times.Load(resp.Id); ok {
						sendTime := val.(time.Time)
						latency := time.Since(sendTime)
						lock.Lock()
						latencies = append(latencies, latency)
						lock.Unlock()
					}
				}

			case "udp":
				conn, err := connectUDP(addr)
				if err != nil {
					fmt.Println("dial error:", err)
					return
				}
				defer conn.Close()

				buf := make([]byte, 2048)
				for i := 0; i < numMessages; i++ {
					id := uuid.New().String()
					ping := Ping{Type: "ping", Id: id}
					data, _ := json.Marshal(ping)

					sendTime := time.Now()
					times.Store(id, sendTime)

					_, err := conn.Write(data)
					if err != nil {
						fmt.Println("write error:", err)
						return
					}

					// Set read timeout to avoid blocking forever if packet is lost
					conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
					n, _, err := conn.ReadFromUDP(buf)
					if err != nil {
						if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
							// fmt.Printf("timeout waiting for response to message %d, skipping...\n", i+1)
							continue // Skip this message and continue to next one
						}
						fmt.Println("read error:", err)
						return
					}

					var resp Ping
					if err := json.Unmarshal(buf[:n], &resp); err != nil {
						fmt.Println("unmarshal error:", err)
						continue
					}

					if val, ok := times.Load(resp.Id); ok {
						sendTime := val.(time.Time)
						latency := time.Since(sendTime)
						lock.Lock()
						latencies = append(latencies, latency)
						lock.Unlock()
					}
				}
			}
		}(c)
	}
	wg.Wait()
	totalTime := time.Since(start)

	// thống kê latency
	var sum, min, max time.Duration
	if len(latencies) > 0 {
		min, max = latencies[0], latencies[0]
	}
	for _, l := range latencies {
		sum += l
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}

	var avgLatency time.Duration
	if len(latencies) > 0 {
		avgLatency = sum / time.Duration(len(latencies))
	}

	fmt.Printf("Sent %d messages from %d clients\n", numClients*numMessages, numClients)
	fmt.Printf("Total time: %v\n", totalTime)
	fmt.Printf("Throughput: %.2f msgs/sec\n", float64(numClients*numMessages)/totalTime.Seconds())
	fmt.Printf("Successful responses: %d/%d\n", len(latencies), numClients*numMessages)
	if len(latencies) > 0 {
		fmt.Printf("Latency: avg=%v, min=%v, max=%v\n", avgLatency, min, max)
	} else {
		fmt.Printf("No successful responses received\n")
	}
}

func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
