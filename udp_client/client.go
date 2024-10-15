package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type sipClientConfig struct {
	srvAddr    net.UDPAddr
	badSrvAddr *net.UDPAddr
	agentAddr  net.UDPAddr
	rTcpConn   net.TCPAddr
	lTcpConn   net.TCPAddr
}

type RequestInvite struct {
	AgentIP string
	LPort   int
}

func main() {
	client := setupClient()

	wg := sync.WaitGroup{}
	wg.Add(1)

	conn, err := net.ListenUDP("udp", &client.agentAddr)
	if err != nil {
		log.Fatalf("Failed to set up listening server: %v\n", err)
	}
	go client.listenUdp(conn, wg)

	// request invite from server with control plane tcp connection
	err = client.sendTcpReq()
	if err != nil {
		log.Fatalf("Failed to send tcp RequestInvite: %v\n", err)
	}

	if client.badSrvAddr != nil {
		go client.tryReachBadServer(conn)
	}

	wg.Wait()

}

func (client *sipClientConfig) sendTcpReq() error {
	tcpConn, err := net.DialTCP("tcp", &net.TCPAddr{
		IP:   client.agentAddr.IP,
		Port: client.rTcpConn.Port,
	}, &client.rTcpConn)
	if err != nil {
		log.Fatalf("Failed to connect to server at %s: %v\n", client.rTcpConn, err)
	}
	defer tcpConn.Close()

	reqInvite := RequestInvite{
		AgentIP: client.agentAddr.String(),
		LPort:   client.agentAddr.Port,
	}

	serializedReqInvite, err := json.Marshal(reqInvite)

	_, err = tcpConn.Write(serializedReqInvite)
	return err
}

func (client *sipClientConfig) listenUdp(conn *net.UDPConn, wg sync.WaitGroup) {
	defer wg.Done()
	for {
		buffer := make([]byte, 1024)
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

		if packet == "Invite" {
			log.Printf("Received Invite from server: %+v\n", conn.RemoteAddr())
			sendRegister(conn, conn.RemoteAddr().String())
			continue
		}

		if packet == "OptionRequest" {
			client.handleRequestOption(conn)
		}
	}
}

func sendRegister(conn *net.UDPConn, addr string) {
	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("Failed to resolve server address: %v\n", err)
	}

	_, err = conn.WriteToUDP([]byte("Register"), serverAddr)
	if err != nil {
		log.Fatalf("Failed to send Register packet: %v\n", err)
	}
}

func (client *sipClientConfig) handleRequestOption(conn *net.UDPConn) {
	log.Printf("Received OptionRequest from server: %s\n", conn.RemoteAddr().String())

	serverAddr, err := net.ResolveUDPAddr("udp", conn.RemoteAddr().String())
	if err != nil {
		log.Fatalf("Failed to resolve server address: %v\n", err)
	}

	_, err = conn.WriteToUDP([]byte("ACK"), serverAddr)
	if err != nil {
		log.Printf("Failed to send ACK: %v\n", err)
	} else {
		log.Printf("Sent ACK to server: %+v\n", serverAddr)
	}
}

func (client *sipClientConfig) tryReachBadServer(conn *net.UDPConn) {
	log.Printf("Trying to reach bad server at %s\n", client.badSrvAddr.String())
	retryCounter := 0
	for {
		sendRegister(conn, client.badSrvAddr.String())
		retryCounter++
		if retryCounter > 8 {
			log.Printf("Bad server reached maximum retries, waiting for 3 minutes before retrying\n")
			time.Sleep(3 * time.Minute)
			retryCounter = 0
		} else {
			time.Sleep(4 * time.Second)
		}

	}
}

func setupClient() *sipClientConfig {
	// Define command-line flags for IP and port
	ip := flag.String("ip", "", "Server IP address - required")
	lip := flag.String("lip", "", "Local IP address - required")
	badIp := flag.String("badip", "", "Bad Server IP address - optional")
	rPort := flag.Int("rport", 8060, "Server port")
	lPort := flag.Int("lport", 5060, "Local port")
	rTcpPort := flag.Int("rtport", 5061, "Server TCP port for control plane")
	lTcplPort := flag.Int("ltport", 5062, "local TCP port for control plane")
	help := flag.Bool("help", false, "Display help")

	// Parse command-line flags
	flag.Parse()

	// Display help if requested
	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Validate required
	if *ip == "" || *lip == "" {
		log.Fatalf("Server IP and Local IP are required\n")
	}

	log.Println("Starting UDP SIP protocol mock client...")
	fmt.Println("Flags provided:")
	flag.Visit(func(f *flag.Flag) {
		fmt.Printf("-%s: %s\n", f.Name, f.Value)
	})

	badSrvAddr := &net.UDPAddr{}
	if *badIp == "" {
		log.Println("No bad server IP provided, will not try to reach bad server")
		badSrvAddr = nil
	} else {
		badSrvAddr = &net.UDPAddr{
			IP:   net.ParseIP(*badIp),
			Port: *rPort,
		}
	}

	// Set up the client
	return &sipClientConfig{
		srvAddr: net.UDPAddr{
			IP:   net.ParseIP(*ip),
			Port: *rPort,
		},
		badSrvAddr: badSrvAddr,
		agentAddr: net.UDPAddr{
			IP:   net.ParseIP(*lip),
			Port: *lPort,
		},
		rTcpConn: net.TCPAddr{
			IP:   net.ParseIP(*ip),
			Port: *rTcpPort,
		},
		lTcpConn: net.TCPAddr{
			IP:   net.ParseIP(*lip),
			Port: *lTcplPort,
		},
	}
}
