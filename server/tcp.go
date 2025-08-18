package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	// "time"
)

type TCPPlayer struct {
	Conn net.Conn
	ID   string
}

var (
	tcpPlayers = make(map[string]*TCPPlayer)
	tcpLock    sync.Mutex
	// deltaTime = 50 * time.Millisecond
)

func handleTCPConnection(conn net.Conn) {
	defer func() {
		log.Printf("[conn %s] closing", conn.RemoteAddr())
		conn.Close()
		tcpLock.Lock()
		delete(tcpPlayers, conn.RemoteAddr().String())
		tcpLock.Unlock()
	}()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("[conn %s] connected", remoteAddr)

	// Player
	player := &TCPPlayer{
		Conn: conn,
		ID:   remoteAddr,
	}
	tcpLock.Lock()
	tcpPlayers[remoteAddr] = player
	tcpLock.Unlock()

	// Scan message
	scanner := bufio.NewScanner(conn)
	// tăng buffer nếu cần (mặc định ~64KB)
	buf := make([]byte, 0, 1024*64)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var msg Ping
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Printf("[conn %s] json error: %v, line=%q", remoteAddr, err, string(line))
			continue
		}

		log.Println("[conn", remoteAddr, "] received message:", msg)
		if msg.Type == "ping" {
			// Echo back the ping message for latency test
			response := Ping{
				Type: "pong",
				Id:   msg.Id,
			}
			data, _ := json.Marshal(response)
			data = append(data, '\n')
			writeTCPMessage(data, player)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[conn %s] read error: %v", remoteAddr, err)
	}
}

// func broadcast() {
// 	t := time.NewTicker(deltaTime)
// 	defer t.Stop()
// 	for range t.C {
// 		for _, p := range tcpPlayers {
// 			data, _ := json.Marshal(p)
// 			data = append(data, '\n')
// 			writeTCPMessage(data, p)
// 		}
// 	}
// }

func writeTCPMessage(data []byte, p *TCPPlayer) {
	if _, err := p.Conn.Write(data); err != nil {
		log.Printf("[broadcast] write to %s error: %v", p.ID, err)
	} else {
		log.Printf("[broadcast] write to %s", p.ID)
	}
}

func runTcp() {
	port := 9000
	listener, err := net.Listen("tcp4", "0.0.0.0:"+fmt.Sprint(port))
	if err != nil {
		log.Fatalf("Listen 0.0.0.0:%s failed: %v", port, err)
	}
	log.Printf("TCP Server listening on 0.0.0.0:%d", port)

	// go broadcast()

	for {
		// Accept() block until a new connection is established
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handleTCPConnection(conn)
	}
}
