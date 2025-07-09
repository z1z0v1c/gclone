package server

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/z1z0v1c/gclone/pkg/http"
)

type Server struct {
	port    uint16
	wwwRoot string
}

func NewServer(port uint16, wwwRoot string) *Server {
	return &Server{
		port:    port,
		wwwRoot: wwwRoot,
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(int(s.port)))
	if err != nil {
		return fmt.Errorf("failed to start server on port %d: %v", s.port, err)
	}

	fmt.Printf("[INFO] Listening on port: %d\n", s.port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("failed to establish the connection: %v", err)
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	req, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Printf("[ERROR] Failed to read request: %v\n", err)

		s.sendErrorResponse(conn, "400 Bad Request")
		return
	}

	parts := strings.Split(req, " ")
	if len(parts) < 3 {
		fmt.Printf("[ERROR] Incomplete request: %v\n", err)

		s.sendErrorResponse(conn, "400 Bad Request")
		return
	}

	method, path, httpVersion := parts[0], parts[1], strings.TrimSpace(parts[2])
	fmt.Printf("[INFO] Request: %s %s %s\n", method, path, httpVersion)

	// Only support GET requests for now
	if method != http.MethodGet {
		fmt.Printf("[ERROR] Request method not allowed: %s\n", method)

		s.sendErrorResponse(conn, "405 Method Not Allowed")
		return
	}

	if path == "/" {
		path = "/index.html"
	}

	wwwRoot, err := filepath.Abs(s.wwwRoot)
	if err != nil {
		fmt.Printf("[ERROR] Invalid www root: %v\n", err)

		s.sendErrorResponse(conn, "400 Bad Request")
		return
	}

	path = filepath.Join(wwwRoot, path)

	// Prevent directory traversal
	path, err = filepath.Abs(path)
	if err != nil || !strings.HasPrefix(path, wwwRoot) {
		fmt.Printf("[ERROR] Forbidden path: %s %v", path, err)

		s.sendErrorResponse(conn, "403 Forbidden")
		return
	}

	fmt.Printf("Path: %s\n", path)

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("[ERROR] File not found: %s %v", path, err)

		s.sendErrorResponse(conn, "404 Not Found")
		return
	}

	s.sendSuccessResponse(conn, data)
}

func (s *Server) sendSuccessResponse(conn net.Conn, data []byte) {
	resp := fmt.Sprintf("HTTP/1.1 200 OK\r\n\r\n%s\r\n", data)
	conn.Write([]byte(resp))
}

func (s *Server) sendErrorResponse(conn net.Conn, status string) {
	resp := fmt.Sprintf("HTTP/1.1 %s\r\n\r\n%s\r\n", status, status)
	conn.Write([]byte(resp))
}
