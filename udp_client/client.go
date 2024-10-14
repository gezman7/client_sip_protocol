package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	//// Define command-line flags for IP and port
	//ip := flag.String("ip", "127.0.0.1", "Server IP address")
	//port := flag.String("port", "8060", "Server port")
	//help := flag.Bool("help", false, "Display help")
	//
	//// Parse command-line flags
	//flag.Parse()
	//
	//// Display help if requested
	//if *help {
	//	fmt.Println("Usage:")
	//	fmt.Println("  client -ip <server_ip> -port <server_port>")
	//	fmt.Println("Options:")
	//	fmt.Println("  -ip      Server IP address (default: 127.0.0.1)")
	//	fmt.Println("  -port    Server port (default: 8060)")
	//	fmt.Println("  -help    Display this help message")
	//	return
	//}
	//
	//// Validate IP and port
	//if *ip == "" || *port == "" {
	//	fmt.Println("Error: IP and port must both be specified.")
	//	flag.Usage()
	//	os.Exit(1)
	//}

	serverAddr := listenForInvite()
	log.Printf("Received Invite packet from server: %s\n", serverAddr)

	// Connect to the server
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect to server at %s: %v\n", serverAddr, err)
	}
	defer conn.Close()

	log.Printf("Connected to server at %s\n", serverAddr)

	// Register the client with the server
	err = sendPacket(conn, "Register")
	if err != nil {
		log.Fatalf("Failed to send Register packet: %v\n", err)
	}

	log.Printf("Sent Register packet to server: %v\n", conn.RemoteAddr())

	// Continuously listen for incoming OptionRequest packets and respond with ACK
	buffer := make([]byte, 1024)
	for {
		// Set a read timeout in case we want to stop the client
		conn.SetReadDeadline(time.Now().Add(2 * time.Minute))

		// Receive the incoming packet from the server
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from server: %v\n", err)
			break
		}

		// Check if the packet is an "OptionRequest"
		packet := string(buffer[:n])
		if packet == "OptionRequest" {
			fmt.Println("Received OptionRequest from server")

			// Send an ACK back to the server
			err = sendPacket(conn, "ACK")
			if err != nil {
				log.Printf("Failed to send ACK: %v\n", err)
			} else {
				fmt.Println("Sent ACK to server")
			}
		}
	}
}

func listenForInvite() string {
	addr := net.UDPAddr{
		Port: 8060,
		IP:   net.ParseIP("0.0.0.0"),
	}

	log.Printf("Listening for Invite packet on port %d\n", addr.Port)
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to set up server: %v\n", err)
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, remoteAddr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		log.Fatalf("Failed to read from UDP connection: %v\n", err)
	}

	log.Printf("Received Invite packet from server: %s\n", remoteAddr.String())
	packet := string(buffer[:n])
	if packet == "Invite" {
		return remoteAddr.String()
	}

	log.Fatalf("Failed to receive Invite packet: %v\n", err)
	return ""
}

func sendPacket(conn net.Conn, message string) error {
	_, err := conn.Write([]byte(message))
	return err
}
