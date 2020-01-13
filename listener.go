package tcpstun

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

type listener struct {
	network   string
	localAddr string
	stunConn  net.Conn
}

func (l *listener) Accept() (net.Conn, error) {
	clientAddr, err := readStr(l.stunConn)
	if err != nil {
		if io.EOF == err {
			log.Println("failed to read client address current name has already been used")
		} else {
			log.Println("failed to read client address", err)
		}
		return nil, err
	}
	time.Sleep(3 * time.Second)
	end := time.Now().Add(60 * time.Second)
	for time.Now().Before(end) {
		conn, err := reuseDial(l.network, l.localAddr, clientAddr)
		if err != nil {
			log.Println("failed to accept client", clientAddr, "retrying")
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			continue
		}
		return conn, nil
	}
	return nil, fmt.Errorf("timeout")
}

func (l *listener) Close() error {
	return l.stunConn.Close()
}

func (l *listener) Addr() net.Addr {
	return l.stunConn.LocalAddr()
}

func Listen(network, stunAddr, localAddr, name string) (net.Listener, error) {
	stunConn, err := reuseDial(network, localAddr, stunAddr)
	if err != nil {
		log.Println("failed to dial stun server", stunAddr, err)
		return nil, err
	}
	err = writeByte(stunConn, 1) // I'm server
	if err != nil {
		log.Println("failed to write type to stun server", stunAddr, err)
		return nil, err
	}
	err = writeStr(stunConn, name)
	if err != nil {
		log.Println("failed to write name to stun server", stunAddr, err)
		return nil, err
	}
	return &listener{network, stunConn.LocalAddr().String(), stunConn}, nil
}
