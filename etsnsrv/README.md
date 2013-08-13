```

etsnsrv(1)                       User Commands                      etsnsrv(1)



NAME
       etsnsrv  -  Create  an  ETSN server that sends its connections to other
       procs over unix domain sockets.

SYNOPSIS
       etsnsrv [-dir Where] [-tcp Mode] [-listen Addr]

DESCRIPTION
       etsnsrv(1) creates an ETSN server.  The protocols the  server  supports
       are  the names of any unix domain sockets in the directory specified by
       -dir (default /tmp/etsn), hereafter $DIR.  When a request comes in ask‐
       ing for protocol X, etsnsrv(1) opens the file $DIR/X and sends the file
       descriptor for the TCP connection to the socket X  (the  in  band  data
       will always be the null byte).

       $DIR  should  be  dedicated  to etsnsrv(1).  Client implementations are
       free to remove files that have the name of the service  they're  trying
       to register.

       etsnsrv(1) does not daemonize; use a process supervisor.

OPTIONS
       -dir Where = /tmp/etsn/
              directory to look for protocol handler socket files

       -tcp Mode = tcp
              One of: tcp, tcp4, tcp6 (4 and 6 refers to IPv{4,6})

       -listen Addr = :5908
              Address to listen on



version 2013-08-13                2013-08-13                        etsnsrv(1)

```

