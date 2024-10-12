package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	// Define command-line flags for IP and port
	ip := flag.String("ip", "127.0.0.1", "Server IP address")
	port := flag.String("port", "8060", "Server port")
	help := flag.Bool("help", false, "Display help")

	// Parse command-line flags
	flag.Parse()

	// Display help if requested
	if *help {
		fmt.Println("Usage:")
		fmt.Println("  client -ip <server_ip> -port <server_port>")
		fmt.Println("Options:")
		fmt.Println("  -ip      Server IP address (default: 127.0.0.1)")
		fmt.Println("  -port    Server port (default: 8060)")
		fmt.Println("  -help    Display this help message")
		return
	}

	// Validate IP and port
	if *ip == "" || *port == "" {
		fmt.Println("Error: IP and port must both be specified.")
		flag.Usage()
		os.Exit(1)
	}

	// Combine IP and port into the server address
	serverAddr := fmt.Sprintf("%s:%s", *ip, *port)

	// Connect to the server
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect to server at %s: %v\n", serverAddr, err)
	}
	defer conn.Close()

	fmt.Printf("Connected to server at %s\n", serverAddr)

	// Register the client with the server
	err = sendPacket(conn, "Register")
	if err != nil {
		log.Fatalf("Failed to send Register packet: %v\n", err)
	}
	fmt.Println("Sent Register packet to server")

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

func sendPacket(conn net.Conn, message string) error {
	_, err := conn.Write([]byte(message))
	return err
}
