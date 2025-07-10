package server

import (
	"bufio"
	"fmt"
	"io"
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

func NewServer(port uint16, wwwRoot string) (*Server, error) {
	wwwRoot, err := filepath.Abs(wwwRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid www root: %v", err)

	}

	s := &Server{
		port:    port,
		wwwRoot: wwwRoot,
	}

	return s, nil
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
		s.logAndSendErrorResponse(conn, "Failed to read request: "+err.Error(), "400 Bad Request")
		return
	}

	parts := strings.Split(req, " ")
	if len(parts) < 3 {
		s.logAndSendErrorResponse(conn, "Incomplete request", "400 Bad Request")
		return
	}

	method, path, httpVersion := parts[0], parts[1], strings.TrimSpace(parts[2])
	fmt.Printf("[INFO] Request: %s %s %s\n", method, path, httpVersion)

	// Only support GET requests for now
	if method != http.MethodGet {
		s.logAndSendErrorResponse(conn, "Request method not allowed: "+method, "405 Method Not Allowed")
		return
	}

	path, resp, err := s.getCleanAbsPath(path)
	if err != nil {
		s.logAndSendErrorResponse(conn, err.Error(), resp)
		return
	}

	data, resp, err := s.readDataFromFile(path)
	if err != nil {
		s.logAndSendErrorResponse(conn, err.Error(), resp)
		return
	}

	s.sendSuccessResponse(conn, data)
}

func (s *Server) getCleanAbsPath(path string) (string, string, error) {
	if path == "/" {
		path = "/index.html"
	}

	path = filepath.Clean(path)
	path = filepath.Join(s.wwwRoot, path)

	// Prevent directory traversal
	path, err := filepath.Abs(path)
	if err != nil {
		return "", "404 Bad Request", err
	}

	if !strings.HasPrefix(path, s.wwwRoot) {
		return "", "403 Forbidden", fmt.Errorf("forbidden path: %s %v", path, err)
	}

	fmt.Printf("[INFO] Path: %s\n", path)

	return path, "", nil
}

func (s *Server) readDataFromFile(path string) ([]byte, string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "404 Not Found", err
		} else {
			return nil, "500 Internal Server Error", err
		}
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, "500 Internal Server Error", err
	}

	// Don't serve directories
	if fileInfo.IsDir() {
		return nil, "403 Forbidden", err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "500 Internal Server Error", err
	}

	fmt.Printf("[INFO] Read %d bytes of data from the file", len(data))

	return data, "", nil
}

func (s *Server) sendSuccessResponse(conn net.Conn, data []byte) {
	resp := fmt.Sprintf("HTTP/1.1 200 OK\r\n\r\n%s\r\n", data)
	conn.Write([]byte(resp))
}

func (s *Server) logAndSendErrorResponse(conn net.Conn, logMsg, status string) {
	fmt.Printf("[ERROR] %s\n", logMsg)
	s.sendErrorResponse(conn, status)
}

func (s *Server) sendErrorResponse(conn net.Conn, status string) {
	resp := fmt.Sprintf("HTTP/1.1 %s\r\n\r\n%s\r\n", status, status)
	conn.Write([]byte(resp))
}
