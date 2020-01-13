package tcpstun

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	reuse "github.com/libp2p/go-reuseport"
)

func connect(network, localAddr, targetAddr string) (net.Conn, error) {
	log.Println("connecting to", targetAddr)

	ch := make(chan net.Conn)
	ch2 := make(chan struct{})
	go func() {
		defer func() {
			log.Println("tmp listener released")
		}()
		listener, err := reuse.Listen(network, localAddr)
		ch2 <- struct{}{}
		if err != nil {
			log.Println("failed to listen at", localAddr)
			ch <- nil
			return
		}
		log.Println("tmp listening at", listener.Addr().String())
		defer listener.Close()
		conn, err := listener.Accept()
		if err != nil {
			log.Println("failed to accept")
			ch <- nil
			return
		}
		ch <- conn
	}()

	<-ch2

	end := time.Now().Add(time.Second * 60)
	for time.Now().Before(end) {
		log.Println("trying...", end.Sub(time.Now()).Seconds())
		conn, err := reuseDial(network, localAddr, targetAddr)
		if err != nil {
			select {
			case conn := <-ch:
				if conn != nil {
					log.Println("accepted", targetAddr)
					return conn, nil
				} else {
					log.Println("failed to listen")
					return nil, err
				}
			case <-time.After(time.Second):
			}
		} else {
			conn2, err2 := net.Dial(network, localAddr)
			if err2 != nil {
				log.Println("you should not see me")
			} else {
				conn2.Close()
			}
			conn2 = <-ch
			if conn2 != nil {
				conn2.Close()
			}
			log.Println("connected to", targetAddr)
			return conn, nil
		}
	}
	log.Println("timeout to connect to", targetAddr, "clear local listener")
	conn2, err2 := net.Dial(network, localAddr)
	if err2 != nil {
		log.Println("you should not see me")
	} else {
		conn2.Close()
	}
	conn2 = <-ch
	if conn2 != nil {
		conn2.Close()
	}
	return nil, fmt.Errorf("dial %s timeout", targetAddr)
}

type listener struct {
	stunConn net.Conn
}

func (l *listener) Accept() (net.Conn, error) {
	network := l.stunConn.LocalAddr().Network()
	localAddr := l.stunConn.LocalAddr().String()
	clientAddr, err := readStr(l.stunConn)
	if err != nil {
		if io.EOF == err {
			log.Println("failed to read client address current name has already been used")
		} else {
			log.Println("failed to read client address", err)
		}
		l.Close()
		return nil, err
	}

	return connect(network, localAddr, clientAddr)
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
	log.Println("using local address", stunConn.LocalAddr().String())
	return &listener{stunConn}, nil
}
