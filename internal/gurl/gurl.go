package gurl

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

type Gurl struct {
	protocol string
	host     string
	port     string
	path     string

	verbose bool
	method  string
	data    string
}

func NewGurl(urls string, verbose bool, method, data string) *Gurl {
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
		host:     host,
		port:     port,
		path:     path,
		verbose:  verbose,
		method:   method,
		data:     data,
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

	reqLines := []string{
		fmt.Sprintf("%s %s HTTP/1.1\r\n", g.method, g.path),
		fmt.Sprintf("Host: %s\r\n", g.host),
		"Accept: */*\r\n",
		"Connection: close\r\n",
	}

	if g.data != "" {
		reqLines = append(reqLines, fmt.Sprintf("Content-Length: %d\r\n", len(g.data)))
	}

	reqLines = append(reqLines, "\r\n")

	_, err = conn.Write([]byte(strings.Join(reqLines, "") + g.data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}

	if g.verbose {
		for _, line := range reqLines {
			fmt.Printf("> %s", line)
		}
	}

	reader := bufio.NewReader(conn)
	inBody := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if g.verbose && !inBody {
			fmt.Printf("< %s", line)
		}

		if inBody {
			fmt.Print(line)
		}

		if line == "\r\n" {
			inBody = true
		}
	}
}
