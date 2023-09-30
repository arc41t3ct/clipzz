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

	conn, err := net.Dial("tcp", address)
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
		fmt.Println("Usage: program <remote IP> <remote port> <local port>")
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
		fmt.Println("Received OS shutdown signal. Exiting gracefully...")
		// Cleanup logic if necessary
		fmt.Println("Graceful shutdown complete.")
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
				fmt.Println("Error sending clipboard data:", err)
			}
			lastHash = currentHash
		}

		time.Sleep(1 * time.Second)
	}
}

