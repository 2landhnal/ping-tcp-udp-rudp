package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	kcp "github.com/xtaci/kcp-go/v5"
	// "time"
)

type RUDPPlayer struct {
	Conn *kcp.UDPSession
	ID   string
}

var (
	players  = make(map[string]*RUDPPlayer)
	rudpLock sync.Mutex
	// deltaTime = 50 * time.Millisecond
)

func handleConnection(conn *kcp.UDPSession) {
	defer func() {
		log.Printf("[conn %s] closing", conn.RemoteAddr())
		conn.Close()
		rudpLock.Lock()
		delete(players, conn.RemoteAddr().String())
		rudpLock.Unlock()
	}()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("[conn %s] connected", remoteAddr)

	// Player
	player := &RUDPPlayer{
		Conn: conn,
		ID:   remoteAddr,
	}
	rudpLock.Lock()
	players[remoteAddr] = player
	rudpLock.Unlock()

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
			writeMessage(data, player)
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
// 		for _, p := range players {
// 			data, _ := json.Marshal(p)
// 			data = append(data, '\n')
// 			writeMessage(data, p)
// 		}
// 	}
// }

func writeMessage(data []byte, p *RUDPPlayer) {
	if _, err := p.Conn.Write(data); err != nil {
		log.Printf("[broadcast] write to %s error: %v", p.ID, err)
	} else {
		log.Printf("[broadcast] write to %s", p.ID)
	}
}

func runRudp() {
	port := 9002
	listener, err := kcp.ListenWithOptions("0.0.0.0:"+fmt.Sprint(port), nil, 10, 3)
	if err != nil {
		log.Fatalf("Listen 0.0.0.0:%d failed: %v", port, err)
	}
	log.Printf("RUDP Server listening on 0.0.0.0:%d", port)

	// go broadcast()

	for {
		// Accept() block until a new connection is established
		conn, err := listener.AcceptKCP()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		// nodelay, interval, resend, nc
		// nodelay: 0 (disable), 1 (enable)
		// interval: internal update timer interval in ms (e.g. 10ms)
		// resend: 0 (normal), 1 (fast resend, ít retransmit delay hơn)
		// nc: 0 (normal congestion control), 1 (disable cc, max send rate)
		conn.SetNoDelay(1, 10, 2, 1)
		go handleConnection(conn)
	}
}
