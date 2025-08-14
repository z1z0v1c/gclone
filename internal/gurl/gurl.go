package gurl

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"strings"
)

type Gurl struct {
	protocol string
	host     string
	port     string
	path     string

	// Flags
	verbose bool
	method  string
	data    string
	header  string
}

func NewGurl(urls string, verbose bool, method, data, header string) (*Gurl, error) {
	url, err := url.Parse(urls)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %v", err)
	}

	protocol := url.Scheme
	if protocol != "http" {
		return nil, fmt.Errorf("invalid protocol (only HTTP is supported)")
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
		header:   header,
	}, nil
}

func (g *Gurl) Run() error {
	addr := net.JoinHostPort(g.host, g.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to the server: %v", err)
	}
	defer conn.Close()

	reqLines := []string{
		fmt.Sprintf("%s %s HTTP/1.1\r\n", g.method, g.path),
		fmt.Sprintf("Host: %s\r\n", g.host),
		"Accept: */*\r\n",
		"Connection: close\r\n",
	}

	if g.header != "" {
		reqLines = append(reqLines, fmt.Sprintf("%s\r\n", g.header))
	}

	if g.data != "" {
		reqLines = append(reqLines, fmt.Sprintf("Content-Length: %d\r\n", len(g.data)))
	}

	reqLines = append(reqLines, "\r\n")

	_, err = conn.Write([]byte(strings.Join(reqLines, "") + g.data))
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
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

	return nil
}
