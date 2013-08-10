A Go implementation of the ETSN protocol:  https://raw.github.com/250bpm/nanomsg/master/rfc/etsn-01.txt

Install:
```
go get github.com/JImmyFrasche/etsn
```

Documentation: http://godoc.org/github.com/JImmyFrasche/etsn

Example server:
```
package main

import (
	"github.com/JImmyFrasche/etsn"
	"io"
	"log"
	"net"
)

func main() {
	s := etsn.New(func(e error) {
		log.Println(e)
	})
	s.Register("echo", func(c *net.TCPConn) {
		log.Println(io.Copy(c, c))
		log.Println(c.Close())
	})
	log.Fatalln(s.Listen("tcp", "127.0.0.1:"))
}
```

Example Client:
```
package main

import (
	"bufio"
	"github.com/JImmyFrasche/etsn"
	"io"
	"log"
	"os"
)

func main() {
	c, err := etsn.Dial("tcp", "127.0.0.1:", "echo")
	if err != nil {
		log.Fatalln(err)
	}

	w := bufio.NewWriter(c)
	w.WriteString("Hello\n")
	w.Flush()
	c.CloseWrite()
	io.Copy(os.Stdout, c)
}
```
