package tcpstun

import (
	"io"
	"log"
	"net"
	"strings"
	"time"

	reuse "github.com/libp2p/go-reuseport"
)

type listener struct {
	network  string
	listener net.Listener
	stunConn net.Conn
}

func (l *listener) Accept() (net.Conn, error) {
	clientAddr, err := readStr(l.stunConn)
	if err != nil {
		if io.EOF == err {
			log.Println("failed to read client address current name has already been used")
		} else if strings.Contains(err.Error(), "use of closed network connection") {
			log.Println("listener is closed, stop listening")
		} else {
			log.Println("failed to read client address", err)
		}
		l.Close()
		return nil, err
	}

	log.Println("try to connect to client first, this time is expected to fail")
	conn, err := reuseDial(l.network, l.stunConn.LocalAddr().String(), clientAddr)
	if err == nil {
		log.Println("amazing! connected in first dial")
		return conn, err
	}
	log.Println("try to accept for 6 seconds")
	// if l.listener.(*net.TCPListener).SetDeadline(time.Now().Add(6 * time.Second)); err != nil {
	// 	log.Println("failed to SetDeadline on listener", err)
	// 	return nil, err
	// }
	go func() {
		time.Sleep(6 * time.Second)
		conn, _ := net.DialTimeout("tcp4", l.listener.Addr().String(), time.Second*5)
		if conn != nil {
			conn.Close()
		}
	}()
	conn, err = l.listener.Accept()
	if err != nil {
		log.Println("try to dial as client")
		conn, err = reuseDial(l.network, l.stunConn.LocalAddr().String(), clientAddr)
		if err != nil {
			log.Println("failed to dial as client, give up", err)
			return nil, err
		}
	} else if conn.RemoteAddr().String() != clientAddr {
		conn.Close()
		log.Println("try to dial as client")
		conn, err = reuseDial(l.network, l.stunConn.LocalAddr().String(), clientAddr)
		if err != nil {
			log.Println("failed to dial as client, give up", err)
			return nil, err
		}
	}
	log.Println("accepted from", conn.RemoteAddr().String())
	return conn, nil
}

func (l *listener) Close() error {
	l.listener.Close()
	return l.stunConn.Close()
}

func (l *listener) Addr() net.Addr {
	return l.listener.Addr()
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
	l, err := reuse.Listen(network, stunConn.LocalAddr().String())
	if err != nil {
		log.Println("failed to listen at", stunConn.LocalAddr().String(), err)
		return nil, err
	}
	return &listener{network, l, stunConn}, nil
}
