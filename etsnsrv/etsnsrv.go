//Create an ETSN server that sends its connections to other procs over unix domain sockets.
//
//etsnsrv(1) creates an ETSN server. The protocols the server supports are the names of any
//unix domain sockets in the directory specified by -dir (default /tmp/etsn), hereafter $DIR.
//When a request comes in asking for protocol X, etsnsrv(1) opens the file $DIR/X and sends
//the file descriptor for the TCP connection to the socket X (the in band data will always
//be the null byte).
//
//$DIR should be dedicated to etsnsrv(1). Client implementations are free to remove files
//that have the name of the service they're trying to register.
//
//etsnsrv(1) does not daemonize; use a process supervisor.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/JImmyFrasche/etsn"
	"github.com/goerlang/fd"
)

var (
	Where = flag.String("dir", "/tmp/etsn/", "directory to look for protocol handler socket files")
	Mode  = flag.String("tcp", "tcp", "One of: tcp, tcp4, tcp6 (4 and 6 refers to IPv{4,6})")
	Addr  = flag.String("listen", ":5908", "Address to listen on")
)

//Try to UnixConn-ify file proto in Where.
func getConn(proto string) (*net.UnixConn, error) {
	file, err := os.Open(filepath.Join(*Where, proto))
	if err != nil {
		return nil, err
	}
	conn, err := net.FileConn(file)
	if err != nil {
		return nil, err
	}
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return nil, fmt.Errorf("etsnsrv: %s is not a unix domain socket", proto)
	}
	return uc, nil
}

func snd(what *net.TCPConn, on *net.UnixConn) error {
	f, err := what.File()
	if err != nil {
		return err
	}
	defer f.Close()
	return fd.Put(on, f)
}

func main() {
	flag.Parse()

	//This proves nothing, but might elicit a few well times forehead slaps
	di, err := os.Stat(*Where)
	if err != nil {
		log.Fatalln(err)
	}
	if !di.IsDir() {
		log.Fatalln(*Where, "is not a directory.")
	}

	s := etsn.New(func(e error) {
		if e != nil {
			log.Println(e)
		}
	})

	s.ProtocolMissing(func(proto string, conn *net.TCPConn) error {
		defer conn.Close()
		ps, err := getConn(proto)
		if err != nil {
			return err
		}
		return snd(conn, ps)
	})

	err = s.Listen(*Mode, *Addr)
	if err != nil {
		log.Fatalln(err)
	}
}
