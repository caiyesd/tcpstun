package tcpstun

import (
	"fmt"
	"log"
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
		// if !strings.Contains(err.Error(), "use of closed network connection") {
		// 	log.Println("failed to read client address", err)
		// }
		return nil, err
	}
	end := time.Now().Add(10 * time.Second)
	for time.Now().Before(end) {
		conn, err := reuseDial(l.network, l.localAddr, clientAddr)
		if err != nil {
			// log.Println("failed to accept client", clientAddr, "retrying")
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
	return &listener{network, localAddr, stunConn}, nil
}
