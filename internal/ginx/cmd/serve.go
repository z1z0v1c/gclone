package cmd

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

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
	defer conn.Close()

	msg, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	path := strings.Split(msg, " ")[1]

	if path == "/" {
		path = "/index.html"
	}

	path = strings.TrimPrefix(path, "/")
	fmt.Printf("Path: %s\n", path)

	var resp string
	data, err := os.ReadFile(path)
	if err != nil {
		resp = "HTTP/1.1 404 Not Found\r\n"
	} else {
		resp = fmt.Sprintf("HTTP/1.1 200 OK\r\n\r\n%s\r\n", data)
	}

	time.Sleep(5 * time.Second)

	conn.Write([]byte(resp))
}
