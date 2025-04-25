package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

var fileIndex = make(map[string]string)

const (
	ChuckSize     = 4096 //4KB
	bootstrapNode = "localhost:9000"
)

type PeerInfo struct {
	Address string   `json:"address"`
	Files   []string `json:"files"`
}

func scanSharedDirectory() []string {
	sharedDir := "../shared/"
	files, err := os.ReadDir(sharedDir)

	if err != nil {
		fmt.Println("Error reading shared directory:", err)
		return nil
	}

	var fileList []string
	fileIndex = make(map[string]string)

	for _, file := range files {
		if !file.IsDir() {
			fullPath := filepath.Join(sharedDir, file.Name())
			fileIndex[file.Name()] = fullPath
			fileList = append(fileList, file.Name())
		}
	}
	fmt.Println("Available files:", fileIndex)
	return fileList
}

func registerWithBootstrapNode(peerAddr string) {
	files := scanSharedDirectory()
	// peer := PeerInfo{Address: peerAddr, Files: files}

	conn, err := net.Dial("tcp", bootstrapNode)
	if err != nil {
		fmt.Println("Error connecting to bootstrap node:", err)
		return
	}
	defer conn.Close()

	message := "REGISTER " + peerAddr + " " + strings.Join(files, ",") + "\n"
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending peer info:", err)
	}

	fmt.Println("Registered with bootstrap node:", bootstrapNode)
}

func findPeersWithFile(fileName string) []string {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("Error connecting to bootstrap node:", err)
		return nil
	}
	defer conn.Close()

	message := "FIND" + fileName + "\n"
	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("ERROR sending FIND request:", err)
		return nil
	}

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("ERROR reading response:", err)
		return nil
	}

	peers := strings.TrimSpace(response)

	if peers == "NOT FOUND" {
		fmt.Println("peers", peers)
		return nil
	}

	return strings.Split(peers, ",")
}

func handleFileRequest(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	requestedFile := strings.TrimSpace(string(buffer[:n]))

	filePath, exists := fileIndex[requestedFile]
	if !exists {
		fmt.Println("Requested file not found:", requestedFile)
		conn.Write([]byte("ERROR: File not found"))
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file stats:", err)
		return
	}

	sizeBuffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBuffer, uint64(fileInfo.Size()))
	conn.Write(sizeBuffer)

	chunk := make([]byte, ChuckSize)

	for {
		bytesRead, err := file.Read(chunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error reading file:", err)
			return
		}

		_, err = conn.Write(chunk[:bytesRead])
		if err != nil {
			fmt.Println("", err)
			return
		}

	}

	fmt.Println("File sent successfully:", requestedFile)
}

func requestFile(fileName string) {

	peers := findPeersWithFile(fileName)

	if len(peers) == 0 {
		fmt.Println("File not found on any peer")
		return
	}

	peerAddr := peers[0]

	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		fmt.Println("Error connecting to peer:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(fileName))
	if err != nil {
		fmt.Println("Error sending file request:", err)
		return
	}

	newFile, err := os.Create("received_" + fileName)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer newFile.Close()

	sizeBuffer := make([]byte, 8)
	conn.Read(sizeBuffer)

	fileSize := int64(binary.LittleEndian.Uint64(sizeBuffer))

	buffer := make([]byte, 4096)
	var totalBytes int64
	bar := progressbar.DefaultBytes(fileSize, "Downloading "+fileName)

	for {
		bytesRead, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Println("Error receiving file from connection:", err)
			return
		}

		_, err = newFile.Write(buffer[:bytesRead])
		if err != nil {
			fmt.Println("Error writing to new file:", err)
			return
		}
		totalBytes += int64(bytesRead)
		bar.Add(bytesRead)
	}
}

func main() {
	peerAddr := "localhost:9001"
	registerWithBootstrapNode(peerAddr)

	listener, err := net.Listen("tcp", peerAddr)
	if err != nil {
		fmt.Println("Error starting peer server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Peer is running on", peerAddr)

	go func() {

		for {
			conn, err := listener.Accept()

			if err != nil {
				fmt.Println("Connection error:", err)
				continue
			}

			go handleFileRequest(conn)
		}
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter command (ls get <filename>, exit)")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)

		if command == "ls" {
			fmt.Println("Shared files:")

			for _, file := range fileIndex {
				fmt.Println(" -", file)
			}

		} else if strings.HasPrefix(command, "get") {
			fileName := strings.TrimPrefix(command, "get")
			requestFile(fileName)
		} else if command == "exit" {
			fmt.Println("Exiting ...")
			break
		} else {
			fmt.Println("Unknown command. Use ls, get <filename>, exit")
		}
	}

}
