package main

import (
	"log"
	"strings"
	"testing"
	"time"
)

func TestTrojanProtocol(t *testing.T) {
	go func() {
		Stun("127.0.0.1:9999")
	}()

	time.Sleep(time.Second * 1)

	l, err := Listen("tcp4", "127.0.0.1:9999", "127.0.0.1:8888", "ABCD")
	if err != nil {
		log.Println("failed to listen", err)
		panic(err)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					log.Println("failed to read client address", err)
				}
				break
			}
			buffer := make([]byte, 128)
			n, err := conn.Read(buffer)
			if err != nil {
				log.Println("failed to read @server", err)
				panic(err)
			}
			log.Println("echo", string(buffer[:n]))
			conn.Write(buffer[:n])
			conn.Close()
		}
	}()

	conn1, err := Dial("tcp4", "127.0.0.1:9999", "127.0.0.1:7777", "ABCD")
	if err != nil {
		log.Println("failed to dail server", err)
	}
	conn1.Write([]byte("hello world"))

	buffer := make([]byte, 128)
	n, err := conn1.Read(buffer)
	if err != nil {
		log.Println("failed to read @client", err)
		panic(err)
	}
	log.Println("got", string(buffer[:n]))
	conn1.Close()

	conn2, err := Dial("tcp4", "127.0.0.1:9999", "127.0.0.1:7777", "ABCD")
	if err != nil {
		log.Println("failed to dail server", err)
	}
	conn2.Write([]byte("hello golang"))

	n, err = conn2.Read(buffer)
	if err != nil {
		log.Println("failed to read @client", err)
		panic(err)
	}
	log.Println("got", string(buffer[:n]))
	conn2.Close()

	l.Close()

	time.Sleep(time.Second * 2)
}
