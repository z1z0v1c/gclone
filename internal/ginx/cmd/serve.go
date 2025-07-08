package cmd

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var Serve = &cobra.Command{
	Use:   "serve",
	Short: "Serve runs a Ginx instance",
	Run:   serve,
}

func serve(c *cobra.Command, args []string) {
	ln, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Listening on port 80")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(conn)

	}
}

func handleConnection(conn net.Conn) {
	var resp string
	defer conn.Close()

	msg, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	path := strings.Split(msg, " ")[1]

	if path == "/" {
		path = "/index.html"
	}

	wwwRoot, err := filepath.Abs("./internal/ginx/www")
	if err != nil {
		log.Fatal("Invalid www root:", err)
	}

	path = filepath.Join(wwwRoot, path)

	// Prevent directory traversal
	path, err = filepath.Abs(path)
	if err != nil || !strings.HasPrefix(path, wwwRoot) {
		resp = "HTTP/1.1 403 Forbidden\r\n"
		conn.Write([]byte(resp))
	}
	fmt.Printf("Path: %s\n", path)

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		resp = "HTTP/1.1 404 Not Found\r\n"
	} else {
		resp = fmt.Sprintf("HTTP/1.1 200 OK\r\n\r\n%s\r\n", data)
	}

	conn.Write([]byte(resp))
}
