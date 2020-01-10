package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

func Stun(address string) error {
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return err
	}
	log.Println("listening at", listener.Addr().String())
	stun := new(stun)
	stun.conns = make(map[string]net.Conn)
	for {
		if conn, err := listener.Accept(); err != nil {
			log.Println("accept error:", err)
		} else {
			go stun.handle(conn)
		}
	}
}

type stun struct {
	lock  sync.Mutex
	conns map[string]net.Conn
}

func (s *stun) handle(conn net.Conn) {
	if conn == nil {
		return
	}
	defer conn.Close()

	isServer := false
	if tmp, err := ReadByte(conn); err != nil {
		log.Println("failed to read client type:", err)
		return
	} else {
		isServer = tmp != 0
	}

	name, err := ReadStr(conn)
	if err != nil {
		log.Println("failed to read name:", err)
		return
	}
	if isServer {
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
			_, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Println("server", name, conn.RemoteAddr().String(), err)
				}
				return
			}
		}
	} else {

		target := name
		s.lock.Lock()
		if conn2, ok := s.conns[target]; ok {
			log.Println("client", conn.RemoteAddr().String(), "-->", fmt.Sprintf("%s(%s)", conn2.RemoteAddr().String(), name))
			WriteStr(conn2, conn.RemoteAddr().String())
			WriteStr(conn, conn2.RemoteAddr().String())
		} else {
			log.Println("targer server", target, "not found")
			WriteStr(conn, "")
		}
		s.lock.Unlock()
	}
}
