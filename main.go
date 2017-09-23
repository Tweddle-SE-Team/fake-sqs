package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/Tweddle-SE-Team/goaws/config"
	"github.com/Tweddle-SE-Team/goaws/services/router"
)

func main() {
	var filename string
	var debug bool
	flag.StringVar(&filename, "config", "", "config file location + name")
	flag.BoolVar(&debug, "debug", false, "debug log level (default Warning)")
	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	env := "Local"
	if flag.NArg() > 0 {
		env = flag.Arg(0)
	}

	portNumbers := config.LoadYamlConfig(filename, env)

	r := router.New()

	if len(portNumbers) == 1 {
		log.Warnf("SNS and SQS listening on: 0.0.0.0:%s", portNumbers[0])
		err := http.ListenAndServe("0.0.0.0:"+portNumbers[0], r)
		log.Fatal(err)
	} else if len(portNumbers) == 2 {
		go func() {
			log.Warnf("SNS and SQS listening on: 0.0.0.0:%s", portNumbers[0])
			err := http.ListenAndServe("0.0.0.0:"+portNumbers[0], r)
			log.Fatal(err)
		}()
		log.Warnf("SNS and SQS listening on: 0.0.0.0:%s", portNumbers[1])
		err := http.ListenAndServe("0.0.0.0:"+portNumbers[1], r)
		log.Fatal(err)
	} else {
		log.Fatal("Not enough or too many ports defined to start GoAws.")
	}
}

type Server struct {
	closed   bool
	handler  http.Handler
	listener net.Listener
	mu       sync.Mutex
}

// Quit closes down the server.
func (srv *Server) Quit() error {
	srv.mu.Lock()
	srv.closed = true
	srv.mu.Unlock()

	return srv.listener.Close()
}

// URL returns a URL for the server.
func (srv *Server) URL() string {
	return "http://" + srv.listener.Addr().String()
}

// New starts a new server and returns it.
func New(addr string) (*Server, error) {
	if addr == "" {
		addr = "localhost:0"
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on localhost: %v", err)
	}

	srv := Server{listener: l, handler: router.New()}

	go http.Serve(l, &srv)

	return &srv, nil
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	srv.mu.Lock()
	closed := srv.closed
	srv.mu.Unlock()

	if closed {
		hj := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
		return
	}

	srv.handler.ServeHTTP(w, req)
}
