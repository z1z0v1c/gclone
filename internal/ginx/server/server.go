package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	fmt.Printf("Listening on port: %d\n", s.port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	msg, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read request: %v", err)

		s.sendErrorResponse(conn, "400 Bad Request")
		return
	}

	path := strings.Split(msg, " ")[1]
	if path == "/" {
		path = "/index.html"
	}

	wwwRoot, err := filepath.Abs(s.wwwRoot)
	if err != nil {
		fmt.Printf("Invalid www root: %v", err)

		s.sendErrorResponse(conn, "400 Bad Request")
		return
	}

	path = filepath.Join(wwwRoot, path)

	// Prevent directory traversal
	path, err = filepath.Abs(path)
	if err != nil || !strings.HasPrefix(path, wwwRoot) {
		fmt.Printf("Forbidden path: %s %v", path, err)

		s.sendErrorResponse(conn, "403 Forbidden")
		return
	}

	fmt.Printf("Path: %s\n", path)

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("File not found: %s %v", path, err)

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
