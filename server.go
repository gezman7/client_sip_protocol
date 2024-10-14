package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Connection struct to manage live connection data
type Connection struct {
	ClientIP         string
	NumOfOptionsSent int
	LastAckRcv       time.Time
	IsClosed         bool
	timer            *time.Timer
	Created          time.Time
}

// Global variables
var (
	connections = make(map[string]*Connection)
	connMutex   = &sync.Mutex{}
)

func main() {

	ip := flag.String("ip", "10.0.1.27", "Server IP address")

	addr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"), // Your local IP
		Port: 8060,                   // Desired local port
	}

	// Start listening for incoming connections
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to set up listening server: %v\n", err)
	}
	defer conn.Close()

	log.Println("UDP Server started on port 8060")
	go inviteClient(conn, *ip)

	for {
		// Buffer to read incoming packets
		buffer := make([]byte, 1024)
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Failed to read from UDP connection: %v\n", err)
			continue
		}

		// Process the incoming packet in a separate goroutine
		go handlePacket(conn, remoteAddr, buffer[:n])
	}
}

func inviteClient(conn *net.UDPConn, ip string) {
	time.Sleep(2 * time.Second)

	clientAddr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: 5060,
	}

	log.Printf("Inviting client at %+v\n with port 5060", clientAddr)

	_, err := conn.WriteToUDP([]byte("Invite"), clientAddr)
	if err != nil {
		log.Printf("Failed to send Invite packet: %v\n", err)
	}
}

func handlePacket(conn *net.UDPConn, addr *net.UDPAddr, packet []byte) {
	clientIP := addr.String()

	connMutex.Lock()
	defer connMutex.Unlock()

	if _, exists := connections[clientIP]; !exists {
		// New connection
		log.Printf("Registering new client: %s\n", clientIP)
		connections[clientIP] = &Connection{
			ClientIP:         clientIP,
			NumOfOptionsSent: 0,
			LastAckRcv:       time.Now(),
			IsClosed:         false,
			Created:          time.Now(),
		}
		// Start sending option requests every minute
		startOptionRequestTimer(conn, addr)
		AckRegister(conn, addr)
	} else {
		// Known connection
		connection := connections[clientIP]
		if connection.IsClosed {
			log.Printf("Connection to %s has been closed and reviced new connection\n", clientIP)
		}

		// Reset option count and last sent time on receiving ACK
		connection.IsClosed = false
		connection.NumOfOptionsSent = 0
		connection.LastAckRcv = time.Now()
		connection.Created = time.Now()
		fmt.Printf("ACK received from %s\n", clientIP)

		// Restart the option request timer
		if connection.timer != nil {
			connection.timer.Stop()
		}
		startOptionRequestTimer(conn, addr)

	}
}

func AckRegister(conn *net.UDPConn, addr *net.UDPAddr) {
	clientIP := addr.String()

	log.Printf("Sending AckRegister to %s\n", clientIP)

	_, err := conn.WriteToUDP([]byte("AckRegister"), addr)
	if err != nil {
		log.Printf("Failed to send option request to %s: %v\n", clientIP, err)
		return
	}
}

func startOptionRequestTimer(conn *net.UDPConn, addr *net.UDPAddr) {
	clientIP := addr.String()
	connection := connections[clientIP]

	// Send an option request after 1 minute
	connection.timer = time.AfterFunc(1*time.Minute, func() {
		sendOptionRequest(conn, addr)
	})
}

func sendOptionRequest(conn *net.UDPConn, addr *net.UDPAddr) {
	clientIP := addr.String()
	connMutex.Lock()
	connection, exists := connections[clientIP]
	connMutex.Unlock()

	if !exists || connection.IsClosed {
		return
	}

	// Send the option request
	log.Printf("Sending option request to %s\n", clientIP)
	_, err := conn.WriteToUDP([]byte("OptionRequest"), addr)
	if err != nil {
		log.Printf("Failed to send option request to %s: %v\n", clientIP, err)
		return
	}

	// Update connection state
	connection.NumOfOptionsSent++

	// Retry after 4 seconds if not ACKed
	time.AfterFunc(4*time.Second, func() {
		connMutex.Lock()
		if connection.NumOfOptionsSent > 6 {
			connection.IsClosed = true
			log.Printf("************Closing connection to %s*************\n", clientIP)
			log.Printf("No ack received for 7 retries - last ack got on %s\n", connection.LastAckRcv)
			log.Printf("Connection created at %s\n", connection.Created)
			log.Printf("Connection was running for %s\n", time.Since(connection.Created))
			connection.timer.Stop()
			connMutex.Unlock()
			return
		}
		if connection.NumOfOptionsSent > 0 && !connection.IsClosed {
			log.Printf("Resending option request to %s\n", clientIP)
			connMutex.Unlock()
			sendOptionRequest(conn, addr)
			return
		} else {
			connMutex.Unlock()
			return
		}
	})
}
