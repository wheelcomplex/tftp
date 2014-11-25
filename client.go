package tftp

import (
	"net"
	"io"
	"log"
	"fmt"
	"sync"
	"io/ioutil"
)

/*
Client provides TFTP client functionality. It requires remote address and
optional logger

Uploading file to server example

	addr, e := net.ResolveUDPAddr("udp", "example.org:69")
	if e != nil {
		...
	}
	file, e := os.Open("/etc/passwd")
	if e != nil {
		...
	}
	r := bufio.NewReader(file)
	log := log.New(os.Stderr, "", log.Ldate | log.Ltime)
	c := tftp.Client{addr, log}
	c.Put(filename, mode, func(writer *io.PipeWriter) {
		n, writeError := r.WriteTo(writer)
		if writeError != nil {
			fmt.Fprintf(os.Stderr, "Can't put %s: %v\n", filename, writeError);
		} else {
			fmt.Fprintf(os.Stderr, "Put %s (%d bytes)\n", filename, n);
		}
		writer.Close()
	})
	
Downloading file from server example

	addr, e := net.ResolveUDPAddr("udp", "example.org:69")
	if e != nil {
		...
	}
	file, e := os.Create("/var/tmp/debian.img")
	if e != nil {
		...
	}
	w := bufio.NewWriter(file)
	log := log.New(os.Stderr, "", log.Ldate | log.Ltime)
	c := tftp.Client{addr, log}
	c.Get(filename, mode, func(reader *io.PipeReader) {
		n, readError := w.ReadFrom(reader)
		if readError != nil {
			fmt.Fprintf(os.Stderr, "Can't get %s: %v\n", filename, readError);
		} else {
			fmt.Fprintf(os.Stderr, "Got %s (%d bytes)\n", filename, n);
		}
		w.Flush()
		file.Close()
	})
*/
type Client struct {
	remoteAddr *net.UDPAddr
	log *log.Logger
	retryCount int
	timeout int
}

func NewClient(remoteAddr *net.UDPAddr) (Client) {
	log := log.New(ioutil.Discard, "", 0)
	return Client{remoteAddr, log, DEFAULT_RETRY_COUNT, DEFAULT_TIMEOUT}
}

func (c Client) SetLogger(logger *log.Logger) {
	c.log = logger
}

func (c Client) Log() (*log.Logger) {
	return c.log
}

func (c Client) SetRetryCount(n int) {
	c.retryCount = n
}

func (c Client) RetryCount() (n int) {
	return c.retryCount
}

func (c Client) SetTimeout(seconds int) {
	c.timeout = seconds
}

func (c Client) Timeout() (seconds int) {
	return c.timeout
}


// Method for uploading file to server
func (c Client) Put(filename string, mode string, handler func(w *io.PipeWriter)) (error) {
	addr, e := net.ResolveUDPAddr("udp", ":0")
	if e != nil {
		return e
	}
	conn, e := net.ListenUDP("udp", addr)
	if e != nil {
		return e
	}
	reader, writer := io.Pipe()
	s := &sender{c, c.remoteAddr, conn, reader, filename, mode}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		handler(writer)
		wg.Done()
	}()
	s.Run(false)
	wg.Wait()
	return nil
}

// Method for downloading file from server
func (c Client) Get(filename string, mode string, handler func(r *io.PipeReader)) (error) {
	addr, e := net.ResolveUDPAddr("udp", ":0")
	if e != nil {
		return e
	}
	conn, e := net.ListenUDP("udp", addr)
	if e != nil {
		return e
	}
	reader, writer := io.Pipe()
	r := &receiver{c, c.remoteAddr, conn, writer, filename, mode}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		handler(reader)
		wg.Done()
	}()
	r.Run(false)
	wg.Wait()
	return fmt.Errorf("Send timeout")
}
