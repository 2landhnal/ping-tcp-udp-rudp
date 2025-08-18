package main

import (
	"encoding/json"
	"log"
	"net"
)

type UDPPlayer struct {
	Conn net.Conn
	ID   string
}

func handleUDPConnection(conn *net.UDPConn) {
	defer func() {
		log.Printf("[conn %s] closing", conn.RemoteAddr())
		conn.Close()
	}()
	buf := make([]byte, 4096)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("read error: %v", err)
			continue
		}

		var msg Ping
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			log.Printf("[conn %s] json error: %v, line=%q", clientAddr, err, string(buf[:n]))
			continue
		}

		log.Printf("[conn %s] received message: %+v", clientAddr, msg)

		if msg.Type == "ping" {
			// Echo back the ping message for latency test
			response := Ping{
				Type: "pong",
				Id:   msg.Id,
			}
			data, _ := json.Marshal(response)
			_, err := conn.WriteToUDP(data, clientAddr)
			if err != nil {
				log.Printf("[send %s] write error: %v", clientAddr, err)
			} else {
				log.Printf("[send %s] pong %s", clientAddr, msg.Id)
			}
		}
	}
}

func runUdp() {
	port := 9001

	addr := net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: port,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Listen UDP 0.0.0.0:%d failed: %v", port, err)
	}
	defer conn.Close()

	log.Printf("UDP server listening on 0.0.0.0:%d", port)
	handleUDPConnection(conn)
}
