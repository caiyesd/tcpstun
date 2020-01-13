package tcpstun

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type stunServer struct {
	address  string
	listener net.Listener
	wg       sync.WaitGroup

	lock  sync.Mutex
	conns map[string]net.Conn
}

func NewStunServer(address string) *stunServer {
	return &stunServer{
		address: address,
	}
}

func (s *stunServer) Addr() net.Addr {
	if s.listener == nil {
		return nil
	} else {
		return s.listener.Addr()
	}
}

func (s *stunServer) Running() bool {
	return s.listener == nil
}

func (s *stunServer) Stop() error {
	if s.listener == nil {
		return nil
	}
	if err := s.listener.(*net.TCPListener).SetDeadline(time.Now()); err != nil {
		return err
	}
	listener := s.listener
	s.listener = nil
	s.wg.Wait()
	log.Println("stop listening at", listener.Addr().String())
	listener.Close()
	return nil
}

func (s *stunServer) Start() error {
	if s.listener != nil {
		return nil
	}
	listener, err := net.Listen("tcp4", s.address)
	if err != nil {
		return err
	}
	s.listener = listener

	log.Println("start listening at", listener.Addr().String())
	s.conns = make(map[string]net.Conn)
	go func() {
		s.wg.Add(1)
		defer s.wg.Done()
		for {
			if conn, err := listener.Accept(); err != nil {
				log.Println(err.Error())
				if strings.Contains(err.Error(), "i/o timeout") && s.listener == nil {
					return
				}
			} else {
				go s.handle(conn)
			}
		}
	}()
	return nil
}

func (s *stunServer) handle(conn net.Conn) {
	if conn == nil {
		return
	}
	defer conn.Close()

	isServer := false
	if tmp, err := readByte(conn); err != nil {
		log.Println("failed to read client type", err)
		return
	} else {
		isServer = tmp != 0
	}

	if isServer {
		name, err := readStr(conn)
		if err != nil {
			log.Println("failed to read name", err)
			return
		}
		s.lock.Lock()
		if _, ok := s.conns[name]; ok {
			s.lock.Unlock()
			log.Println("name", "\""+name+"\"", "has already been used")
			return
		}
		s.conns[name] = conn
		s.lock.Unlock()

		log.Println("server", fmt.Sprintf("%s(%s)", conn.RemoteAddr().String(), name), "connected")

		defer func() {
			log.Println("server", fmt.Sprintf("%s(%s)", conn.RemoteAddr().String(), name), "disconnected")
			s.lock.Lock()
			delete(s.conns, name)
			s.lock.Unlock()
		}()

		buffer := make([]byte, 1024)
		for {
			_, err = conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Println("server", name, conn.RemoteAddr().String(), err)
				}
				return
			}
		}
	} else {
		for {
			name, err := readStr(conn)
			if err != nil {
				log.Println("failed to read name", err)
				return
			}
			target := name
			s.lock.Lock()
			if conn2, ok := s.conns[target]; ok {
				log.Println("client", conn.RemoteAddr().String(), "-->", fmt.Sprintf("%s(%s)", conn2.RemoteAddr().String(), name))
				writeStr(conn2, conn.RemoteAddr().String())
				writeStr(conn, conn2.RemoteAddr().String())
			} else {
				log.Println("targer server", target, "not found")
				writeStr(conn, "")
			}
			s.lock.Unlock()
		}
	}
}
