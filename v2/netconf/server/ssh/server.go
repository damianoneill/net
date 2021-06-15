package ssh

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

// Server represents a test SSH Server
type Server struct {
	listener net.Listener
	trace    *Trace
}

// Handler is the interface that is implemented to handle an SSH channel.
type Handler interface {
	// Handler is a function that handles i/o to/from an SSH channel
	Handle(ch ssh.Channel)
}

// HandlerFactory is a function that will deliver an Handler.
type HandlerFactory func(conn *ssh.ServerConn) Handler

// NewServer deflivers a new test SSH Server, with a custom channel handler.
// The server implements password authentication with the given credentials.
func NewServer(ctx context.Context, address string, port int, cfg *ssh.ServerConfig, factory HandlerFactory) (server *Server, err error) {
	server = &Server{trace: ContextSSHTrace(ctx)}

	listenAddress := fmt.Sprintf("%s:%d", address, port)
	server.listener, err = net.Listen("tcp", listenAddress)
	server.trace.Listened(address, err)
	if err != nil {
		return nil, err
	}

	go server.acceptConnections(cfg, factory)

	return server, nil
}

// Port delivers the tcp port number on which the server is listening.
func (s *Server) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

// Close closes any resources used by the server.
func (s *Server) Close() {
	_ = s.listener.Close()
}

func (s *Server) acceptConnections(config *ssh.ServerConfig, factory HandlerFactory) {
	s.trace.StartAccepting()
	for {
		nConn, err := s.listener.Accept()
		s.trace.Accepted(nConn, err)
		if err != nil {
			return
		}

		svrconn, chch, reqch, err := ssh.NewServerConn(nConn, config)
		s.trace.NewServerConn(nConn, err)
		if err != nil {
			continue
		}

		go ssh.DiscardRequests(reqch)

		// Service the incoming Channel channel.
		for newChannel := range chch {
			dataChan, requests, err := newChannel.Accept()
			s.trace.SSHChannelAccept(nConn, err)
			if err != nil {
				continue
			}

			// Handle the "subsystem" request.
			go func(in <-chan *ssh.Request) {
				for req := range in {
					err = req.Reply(req.Type == "subsystem", nil)
					s.trace.SubsystemRequestReply(err)
				}
			}(requests)

			go func() {
				defer dataChan.Close()
				factory(svrconn).Handle(dataChan)
			}()
		}
	}
}
