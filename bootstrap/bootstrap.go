package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
)

var peersRegistry = make(map[string]bool)
var fileRegistry = make(map[string][]string)
var mu = sync.Mutex{}

func main() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Println("Error starting bootstrap node:", err)
		return
	}

	defer listener.Close()
	fmt.Println("Bootstrap node listening on port 9000...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handlePeerConnection(conn)
	}
}

func handlePeerConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	message, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading message:", err)
		return
	}

	message = strings.TrimSpace(message)
	parts := strings.Split(message, " ")

	if len(parts) < 2 {
		fmt.Println("Enter a value message")
		return
	}

	command := parts[0]

	switch command {
	case "REGISTER":
		peerAddr := parts[1]
		mu.Lock()
		peersRegistry[peerAddr] = true
		mu.Unlock()

		fmt.Println("New peer connected:", peerAddr)

		stringFiles := strings.Join(parts[2:], "")
		files := strings.Split(stringFiles, ",")
		fmt.Println("files:", files)

		for _, file := range files {
			mu.Lock()
			fileRegistry[file] = append(fileRegistry[file], peerAddr)
			mu.Unlock()
		}

		_, err := conn.Write([]byte("REGISTERED"))
		if err != nil {
			fmt.Println("Error writing response to connection:", err)
			return
		}

	case "FIND":
		fileName := parts[1]
		peers, exists := fileRegistry[fileName]

		if !exists || len(peers) == 0 {
			conn.Write([]byte("NOT FOUND\n"))
		} else {
			conn.Write([]byte(strings.Join(peers, ",") + "\n"))
		}
	default:
		fmt.Println("Unknown command")

	}

}
