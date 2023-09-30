package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/atotto/clipboard"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const maxRetries = 100
const retryDelay = 2 * time.Second

func getClipboardHash() (string, error) {
	content, err := clipboard.ReadAll()
	if err != nil {
		return "", err
	}
	hasher := md5.New()
	hasher.Write([]byte(content))
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func sendClipboardData(address string) error {
	content, err := clipboard.ReadAll()
	if err != nil {
		return err
	}

	var conn net.Conn
	for i := 0; i < maxRetries; i++ {
		conn, err = net.Dial("tcp", address)
		if err == nil || err != nil && i == maxRetries-1 {
			break
		}
		time.Sleep(retryDelay)
	}

	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(content))
	return err
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, conn)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		return
	}

	if err := clipboard.WriteAll(buf.String()); err != nil {
		fmt.Println("Error writing to clipboard:", err)
	}
}

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("%s allows you to sync the clipboard between two computers\n", os.Args[0])
		fmt.Printf("Usage: %s <remote IP> <remote port> <local port>\n", os.Args[0])
		return
	}

	remoteAddress := os.Args[1] + ":" + os.Args[2]
	localPort := os.Args[3]

	go func() {
		listener, err := net.Listen("tcp", ":"+localPort)
		if err != nil {
			fmt.Println("Error starting server:", err)
			return
		}
		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connection:", err)
				continue
			}
			go handleConnection(conn)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		fmt.Println("Received OS shutdown signal. Exiting...")
		// Cleanup logic if necessary
		os.Exit(0)
	}()

	var lastHash string
	for {
		currentHash, err := getClipboardHash()
		if err != nil {
			fmt.Println("Error getting clipboard hash:", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if currentHash != lastHash {
			if err := sendClipboardData(remoteAddress); err != nil {
				fmt.Println("Failed to send clipboard data after retries:", err)
			}
			lastHash = currentHash
		}

		time.Sleep(1 * time.Second)
	}
}

