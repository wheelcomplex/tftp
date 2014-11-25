package tftp

import (
	"net"
	"fmt"
	"io"
	"log"
	"io/ioutil"
)

/*
Server provides TFTP server functionality. It requires bind address, handlers
for read and write requests and optional logger.

	func HandleWrite(filename string, r *io.PipeReader) {
		buffer := &bytes.Buffer{}
		c, e := buffer.ReadFrom(r)
		if e != nil {
			fmt.Fprintf(os.Stderr, "Can't receive %s: %v\n", filename, e)
		} else {
			fmt.Fprintf(os.Stderr, "Received %s (%d bytes)\n", filename, c)
			...
		}
	}
	func HandleRead(filename string, w *io.PipeWriter) {
		if fileExists {
			...
			c, e := buffer.WriteTo(w)
			if e != nil {
				fmt.Fprintf(os.Stderr, "Can't send %s: %v\n", filename, e)
			} else {
				fmt.Fprintf(os.Stderr, "Sent %s (%d bytes)\n", filename, c)
			}
			w.Close()
		} else {
			w.CloseWithError(fmt.Errorf("File not exists: %s", filename))
		}
	}
	...
	addr, e := net.ResolveUDPAddr("udp", ":69")
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	log := log.New(os.Stderr, "TFTP", log.Ldate | log.Ltime)
	s := tftp.Server{addr, HandleWrite, HandleRead, log}
	e = s.Serve()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
*/
type Server struct {
	bindAddr *net.UDPAddr
	readHandler func(filename string, r *io.PipeReader)
	writeHandler func(filename string, w *io.PipeWriter)
	log *log.Logger
	retryCount int
	timeout int
}

func NewServer(bindAddr *net.UDPAddr, readHandler func(filename string, r *io.PipeReader), writeHandler func(filename string, w *io.PipeWriter)) (Server){
	log := log.New(ioutil.Discard, "", 0)
	return Server{bindAddr, readHandler, writeHandler, log, DEFAULT_RETRY_COUNT, DEFAULT_TIMEOUT}
}

func (s Server) SetLogger(logger *log.Logger) {
	s.log = logger
}

func (s Server) Log() (*log.Logger) {
	return s.log
}

func (s Server) SetRetryCount(n int) {
	s.retryCount = n
}

func (s Server) RetryCount() (n int) {
	return s.retryCount
}

func (s Server) SetTimeout(seconds int) {
	s.timeout = seconds
}

func (s Server) Timeout() (seconds int) {
	return s.timeout
}

func (s Server) Serve() (error) {
	conn, e := net.ListenUDP("udp", s.bindAddr)
	if e != nil {
		return e
	}
	for {
		e = s.processRequest(conn)
		if e != nil {
			if s.Log != nil {
				s.Log().Printf("%v\n", e);
			}
		}
	}
}

func (s Server) processRequest(conn *net.UDPConn) (error) {
	var buffer []byte
	buffer = make([]byte, MAX_DATAGRAM_SIZE)
	n, remoteAddr, e := conn.ReadFromUDP(buffer)
	if e != nil {
		return fmt.Errorf("Failed to read data from client: %v", e)
	}
	p, e := ParsePacket(buffer[:n])
	if e != nil {
		return nil
	}
	switch p := Packet(*p).(type) {
		case *WRQ:
			s.Log().Printf("got WRQ (filename=%s, mode=%s)", p.Filename, p.Mode)
			trasnmissionConn, e := s.transmissionConn()
			if e != nil {
				return fmt.Errorf("Could not start transmission: %v", e)
			}
			reader, writer := io.Pipe()
			r := &receiver{s, remoteAddr, trasnmissionConn, writer, p.Filename, p.Mode}
			go s.readHandler(p.Filename, reader)
			// Writing zero bytes to the pipe just to check for any handler errors early
			var null_buffer []byte
			null_buffer = make([]byte, 0)
			_, e = writer.Write(null_buffer)
			if e != nil {
				errorPacket := ERROR{1, e.Error()}
				trasnmissionConn.WriteToUDP(errorPacket.Pack(), remoteAddr)
				s.Log().Printf("sent ERROR (code=%d): %s", 1, e.Error())
				return e
			}
			go r.Run(true)
		case *RRQ:
			s.Log().Printf("got RRQ (filename=%s, mode=%s)", p.Filename, p.Mode)
			trasnmissionConn, e := s.transmissionConn()
			if e != nil {
				return fmt.Errorf("Could not start transmission: %v", e)
			}
			reader, writer := io.Pipe()
			r := &sender{s, remoteAddr, trasnmissionConn, reader, p.Filename, p.Mode}
			go s.writeHandler(p.Filename, writer)
			go r.Run(true)
	}
	return nil
}

func (s Server) transmissionConn() (*net.UDPConn, error) {
	addr, e := net.ResolveUDPAddr("udp", ":0")
	if e != nil {
		return nil, e
	}
	conn, e := net.ListenUDP("udp", addr)
	if e != nil {
		return nil, e
	}
	return conn, nil
}