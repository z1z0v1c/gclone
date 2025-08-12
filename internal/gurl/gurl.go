package gurl

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
)

type Gurl struct {
	protocol string
	host     string
	port     string
	path     string
}

func NewGurl(urls string) *Gurl {
	url, err := url.Parse(urls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid url: %v.\n", err)
		os.Exit(1)
	}

	protocol := url.Scheme
	if protocol != "http" {
		fmt.Fprintf(os.Stderr, "Invalid protocol. Only http is supported.\n")
		os.Exit(1)
	}

	host := url.Hostname()
	port := url.Port()
	if port == "" {
		port = "80"
	}

	path := url.Path
	if path == "" {
		path = "/"
	}

	return &Gurl{
		protocol: protocol,
		host: host,
		port: port,
		path: path,
	}
}

func (g *Gurl) Start() {
	addr := net.JoinHostPort(g.host, g.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := fmt.Sprintf("GET %s HTTP/1.1\r\n", g.path)
	req += fmt.Sprintf("Host: %s\r\n", g.host)
	req += "Accept: */*\r\n"
	req += "Connection: close\r\n"
	req += "\r\n"

	_, err = conn.Write([]byte(req))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(conn)
	inBody := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if inBody {
			fmt.Print(line)
		}

		if line == "\r\n" {
			inBody = true
		}
	}
}
