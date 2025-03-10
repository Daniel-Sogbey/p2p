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
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}

	defer conn.Close()

	fmt.Println("Connected to server!")

	var input string
	fmt.Scanln(&input)

	fileNames := strings.Split(input, ",")

	_, err = conn.Write([]byte(input))
	if err != nil {
		fmt.Println("Error sending file request:", err)
		return
	}

	for _, fileName := range fileNames {
		receiveFile(conn, fileName)
	}
	fmt.Println("All files received successfully!")
}

func receiveFile(conn net.Conn, fileName string) {
	sizeBuffer := make([]byte, 8)
	_, err := conn.Read(sizeBuffer)
	if err != nil {
		fmt.Println("Error reading file size:", err)
		return
	}

	fileSize := int64(binary.LittleEndian.Uint64(sizeBuffer))
	fmt.Println("Receiving file of size:", fileSize, "bytes")

	nameBuffer := make([]byte, 256)
	_, err = conn.Read(nameBuffer)
	if err != nil {
		fmt.Println("Error reading file name for ", fileName, ":", err)
		return
	}

	receivedFileName := strings.Trim(string(nameBuffer), "\x00")

	newFile, err := os.Create("received_" + receivedFileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}

	defer newFile.Close()

	var receivedBytes int64 = 0
	receivedBuffer := make([]byte, ChunkSize)
	bar := progressbar.DefaultBytes(fileSize, "Downloading "+fileName)

	for receivedBytes < fileSize {

		conn.Write([]byte("NEXT"))

		remainingBytes := fileSize - receivedBytes

		if remainingBytes < ChunkSize {
			receivedBuffer = make([]byte, remainingBytes)
		}

		bytesRead, err := conn.Read(receivedBuffer)
		if err != nil {
			fmt.Println("Error receiving file:", err)
			return
		}

		_, err = newFile.Write(receivedBuffer[:bytesRead])
		if err != nil {
			fmt.Println("Error writing to new file:", err)
			return
		}
		receivedBytes += int64(bytesRead)
		bar.Add(bytesRead)
	}

	fmt.Println("\nDownload complete" + receivedFileName)
}
