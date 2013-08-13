//etsnunix implements an interface similar to etsn, but for listening
//for connections on a unix socket provided by the etsnsrv command.
package etsnunix

import (
	"errors"
	"net"
	"os"
	"path/filepath"

	"github.com/JImmyFrasche/etsn"
	"github.com/goerlang/fd"
)

//Server encapsulates the state of an etsnsrv listener.
type Server struct {
	dir string
	log func(error)
}

//New creates a new server that advertises its protocols at dir.
//
//logger is called whenever there's an error establishing a connection
//within Listen. Note that the error may be nil.
//If logger is nil, a no op logger is used.
func New(dir string, logger func(error)) *Server {
	if logger == nil {
		logger = func(error) {}
	}
	return &Server{
		dir: dir,
		log: logger,
	}
}

//Listen advertises a single protocol, proto, in the directory the server
//was created with. It will invoke handler in a new goroutine every time
//a fd representing a tcp socket is sent down the unix domain socket
//created at dir/proto by an instance of etnsrv on dir.
//
//Warning: if there is an existing file named dir/proto it will be deleted.
//
//It is safe to call multiple times on same server, with different proto.
func (s *Server) Listen(proto string, handler func(*net.TCPConn) error) error {
	if len(proto) > 255 {
		return etsn.ErrProtocolIdentifierTooLong
	}
	nm := filepath.Join(s.dir, proto)
	if err := os.Remove(nm); err != nil {
		return err
	}
	c, err := net.Dial("unix", nm)
	if err != nil {
		return err
	}
	conn := c.(*net.UnixConn)
	for {
		fs, err := fd.Get(conn, 1, nil)
		if err != nil {
			//BUG(jmf) There are surely many an error that should lead to
			//us breaking out of the listen loop. For example if another
			//process deletes our socket
			s.log(err)
			continue
		}
		if len(fs) != 1 {
			s.log(errors.New("Did not receive exactly one fd"))
			continue
		}
		f := fs[0]
		ic, err := net.FileConn(f)
		if err != nil {
			f.Close()
			s.log(err)
			continue
		}
		s.log(f.Close())
		tcp, ok := ic.(*net.TCPConn)
		if !ok {
			s.log(errors.New("Received invalid socket type"))
			ic.Close()
			continue
		}
		go func() {
			s.log(handler(tcp))
		}()
	}
}
