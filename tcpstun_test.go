package tcpstun

import (
	"log"
	"strings"
	"testing"
	"time"
)

const StunAddr = "127.0.0.1:23710"
const ClientAddr = "127.0.0.1:0"
const ServerAddr = "127.0.0.1:0"
const ServerName = "ABCD"

func TestTrojanProtocol(t *testing.T) {
	stunServer := NewStunServer(StunAddr)
	stunServer.Start()
	// defer stunServer.Stop()
	time.Sleep(time.Second * 1)

	l, err := Listen("tcp4", StunAddr, ClientAddr, ServerName)
	if err != nil {
		log.Println("failed to listen", err)
		panic(err)
	}

	ch := make(chan int)

	go func() {
		// for {
		conn, err := l.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				log.Println("failed to read client address", err)
			}
			return
		}
		buffer := make([]byte, 128)
		s, err := conn.Read(buffer)
		if err != nil {
			log.Println("failed to read @server", err)
			panic(err)
		}
		log.Println("echo", string(buffer[:s]))
		conn.Write(buffer[:s])
		conn.Close()
		// }
		ch <- 1
	}()

	conn1, err := Dial("tcp4", StunAddr, ServerAddr, ServerName)
	if err != nil {
		log.Println("failed to dail server", err)
		panic(err)
	}
	conn1.Write([]byte("hello world"))

	buffer := make([]byte, 128)
	s, err := conn1.Read(buffer)
	if err != nil {
		log.Println("failed to read @client", err)
		panic(err)
	}
	log.Println("got", string(buffer[:s]))
	conn1.Close()

	// conn2, err := Dial("tcp4", StunAddr, ServerAddr, ServerName)
	// if err != nil {
	// 	log.Println("failed to dail server", err)
	// }
	// conn2.Write([]byte("hello golang"))

	// n, err = conn2.Read(buffer)
	// if err != nil {
	// 	log.Println("failed to read @client", err)
	// 	panic(err)
	// }
	// log.Println("got", string(buffer[:n]))
	// conn2.Close()

	<-ch
	l.Close()

	stunServer.Stop()
}
