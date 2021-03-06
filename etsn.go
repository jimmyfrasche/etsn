//A simple implementation of ETSN: https://raw.github.com/250bpm/nanomsg/master/rfc/etsn-01.txt
package etsn

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

var (
	ErrProtocolIdentifierTooLong  = errors.New("ETSN protocol identifier exceeds 255 bytes")
	ErrUnsupportedProtocolVersion = errors.New("Unsupported ETSN protocol version")
	ErrInvalidHeader              = errors.New("Invalid ETSN header")
)

func addrfix(laddr string) string {
	if len(laddr) > 0 && laddr[len(laddr)-1] == ':' {
		laddr += "5908"
	}
	return laddr
}

//Dial connects to the specified ETSN server and requests protocol proto.
//
//nett must be one of "tcp", "tcp4", "tcp6".
//
//laddr is standard Go networking address as used in the
//net package. If the laddr string ends in ":", the default
//port, 5908, is appended.
//
//If the server server does not speak the protocol proto, an error
//will be returned; otherwise a TCP connection is returned ready to use.
func Dial(nett, laddr, proto string) (*net.TCPConn, error) {
	if len(proto) > 255 {
		return nil, ErrProtocolIdentifierTooLong
	}
	conn, err := net.Dial(nett, addrfix(laddr))
	if err != nil {
		return nil, err
	}
	n, err := conn.Write(append([]byte{1, byte(len(proto))}, proto...))
	if err != nil || n != len(proto)+2 {
		conn.Close()
		switch {
		case err != nil:
			return nil, err
		case n != len(proto)+2:
			return nil, io.ErrShortWrite
		}
	}
	return conn.(*net.TCPConn), nil
}

//Server encapsulates the state of an ETSN server.
type Server struct {
	protos  map[string]func(*net.TCPConn) error
	lock    sync.Mutex
	log     func(error)
	missing func(string, *net.TCPConn) error
}

//New returns a new Server.
//
//logger is called whenever there's an error establishing
//a connection within Listen. Note that the error may be nil.
//If logger is nil, a no op logger is used.
//The logger may be called by multiple goroutines.
//Errors returned from handlers are passed to logger.
func New(logger func(error)) *Server {
	if logger == nil {
		logger = func(error) {}
	}
	return &Server{
		protos: map[string]func(*net.TCPConn) error{},
		log:    logger,
	}
}

//ProtocolMissing is called when no protocol is found.
//The first argument is the name of the unknown protocol, otherwise
//it behaves exactly like a regular handler.
//If no ProtocolMissing handler is set, or this is called with nil,
//requests will be closed and ignored.
func (s *Server) ProtocolMissing(pm func(string, *net.TCPConn) error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.missing = pm
}

//Register registers a handler function for the protocol named proto.
//
//If there was already a protocol registered with identifier proto,
//handler will be used for any future connections. All existing
//connections of proto will remain with the previous handler until
//the connections are closed.
func (s *Server) Register(proto string, handler func(*net.TCPConn) error) error {
	if len(proto) > 255 {
		return ErrProtocolIdentifierTooLong
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.protos[proto] = handler
	return nil
}

//Unregister removes any handler associated with the identifier proto,
//if present.
//
//No existing connection will be effected.
func (s *Server) Unregister(proto string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.protos, proto)
}

//Help is a local version of the TCPMUX HELP protocol.
//It returns a list of all the protocols the server
//implements. It is not exposed by the server, but can be
//made to do so trivially, if desired: (error handling elided
//for brevity)
//	server.Register("HELP", func(c *net.TCPConn) {
//		w := bufio.NewWriter(c)
//		for _, p := range server.Help() {
//			w.WriteString(p)
//			w.WriteByte('\n')
//		}
//		w.Flush()
//		c.Close()
//	})
func (s *Server) Help() (protos []string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for p := range s.protos {
		protos = append(protos, p)
	}
	return
}

//Listen starts an ETSN server on port 5908.
//
//When connections are made they are dispatched,
//based on the client's requested protocol identifier,
//to any handler registered via Register, otherwise the
//request is dropped.
//
//If a logger was set with SetListenLogger, all errors
//during the ETSN handshake will be passed to it, there will
//be at most one error per goroutine.
//
//nett must be one of "tcp", "tcp4", "tcp6".
//
//laddr is standard Go networking address as used in the
//net package. If the laddr string ends in ":", the default
//port, 5908, is appended.
func (s *Server) Listen(nett, laddr string) error {
	Ln, err := net.Listen(nett, addrfix(laddr))
	if err != nil {
		return err
	}
	ln := Ln.(*net.TCPListener)
	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			s.log(err)
			//we assume that any error here means we don't care
			continue
		}
		go func() {
			conn.SetReadDeadline(time.Now().Add(time.Second))

			header := make([]byte, 2)
			n, err := conn.Read(header)
			if err != nil || n != 2 || header[0] != 1 {
				conn.Close()
				switch {
				case err != nil:
					s.log(err)
				case n != 2:
					s.log(ErrInvalidHeader)
				case header[0] != 1:
					s.log(ErrUnsupportedProtocolVersion)
				}
				return
			}
			length := int(header[1])
			proto := make([]byte, length)
			n, err = conn.Read(proto)
			if err != nil || n != length {
				conn.Close()
				switch {
				case err != nil:
					s.log(err)
				case n != length:
					s.log(ErrInvalidHeader)
				}
				return
			}
			conn.SetReadDeadline(time.Time{})
			sproto := string(proto)

			s.lock.Lock()
			handler, ok := s.protos[sproto]
			missing := s.missing
			s.lock.Unlock()

			if !ok {
				conn.Close()
			} else if missing != nil {
				s.log(missing(sproto, conn))
			} else {
				s.log(handler(conn))
			}
		}()
	}
}
