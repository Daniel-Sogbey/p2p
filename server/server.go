package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/schollz/progressbar/v3"
)

const ChunkSize = 4096 //4KB

func main() {
	listener, err := net.Listen("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}

	defer listener.Close()

	fmt.Println("server is listening on port 9000...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleClient(conn)
	}

}

func handleClient(conn net.Conn) {
	defer conn.Close()
	fmt.Println("New client connected:", conn.RemoteAddr())

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from client:", err)
		return
	}

	requestedFiles := strings.Split(string(buffer[:n]), ",")

	for _, fileName := range requestedFiles {
		fileName = strings.TrimSpace(fileName)
		sendFile(conn, fileName)
	}

}

func sendFile(conn net.Conn, fileName string) {
	fmt.Println("Client request file:", fileName)

	fileInfo, err := os.Stat(fileName)
	if err != nil {
		fmt.Println("Error getting file info:", err)
		return
	}

	fileSize := fileInfo.Size()
	fmt.Println("Sending file size:", fileSize)

	sizeBuffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBuffer, uint64(fileSize))
	conn.Write(sizeBuffer)

	fileNameBuffer := make([]byte, 256)
	copy(fileNameBuffer, []byte(fileName))
	conn.Write(fileNameBuffer)

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		conn.Write([]byte("ERROR: File not found"))
		return
	}

	defer file.Close()

	chuck := make([]byte, ChunkSize)
	totalSent := 0

	bar := progressbar.DefaultBytes(fileSize, "Uploading "+fileName)

	for {
		if totalSent >= int(fileSize) {
			fmt.Println("\nFile fully sent. Closing connection.")
			return
		}
		requestBuffer := make([]byte, 4)
		_, err := conn.Read(requestBuffer)
		if err != nil {
			fmt.Println("Error reading client request:", err)
			return
		}

		bytesRead, err := file.Read(chuck)
		if err != nil {
			break
		}

		_, err = conn.Write(chuck[:bytesRead])
		if err != nil {
			fmt.Println("Error writing buffer to connection:", err)
			return
		}
		totalSent += bytesRead
		bar.Add(bytesRead)
	}

}
