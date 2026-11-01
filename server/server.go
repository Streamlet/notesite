package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Streamlet/NoteIsSite/global"
)

type HttpServer interface {
	Serve() error
	Shutdown() error
}

func NewPortServer(port uint, noteRoot string, templateRoot string) (HttpServer, error) {
	var err error
	server := new(portServer)
	server.Handler, err = newRouter(noteRoot, templateRoot)
	server.port = port
	return server, err
}

func NewSockServer(sock string, noteRoot string, templateRoot string) (HttpServer, error) {
	var err error
	server := new(sockServer)
	server.Handler, err = newRouter(noteRoot, templateRoot)
	server.sock = sock
	return server, err
}

type portServer struct {
	http.Server
	port uint
}

type sockServer struct {
	http.Server
	sock string
}

func serve(s *http.Server, l net.Listener) {
	go func() {
		err := s.Serve(l)
		if err != http.ErrServerClosed {
			global.GetErrorChan() <- err
		}
	}()
}

const shutdownTimeout = 5 * time.Second

func shutdown(s *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		_ = s.Close()
		return err
	}
	return nil
}

func (s *portServer) Serve() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	serve(&s.Server, l)
	return nil
}

func (s *portServer) Shutdown() error {
	return shutdown(&s.Server)
}

func (s *sockServer) Serve() error {
	_ = os.Remove(s.sock)
	l, err := net.Listen("unix", s.sock)
	if err != nil {
		return err
	}
	serve(&s.Server, l)
	return nil
}

func (s *sockServer) Shutdown() error {
	err := shutdown(&s.Server)
	_ = os.Remove(s.sock)
	return err
}
