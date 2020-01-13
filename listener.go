package tcpstun

import (
	"io"
	"log"
	"net"
	"strings"

	reuse "github.com/libp2p/go-reuseport"
)

type listener struct {
	network  string
	listener net.Listener
	stunConn net.Conn
}

func (l *listener) Accept() (net.Conn, error) {
	return l.listener.Accept()
	// clientAddr, err := readStr(l.stunConn)
	// if err != nil {
	// 	if io.EOF == err {
	// 		log.Println("failed to read client address current name has already been used")
	// 	} else {
	// 		log.Println("failed to read client address", err)
	// 	}
	// 	return nil, err
	// }

	// conn, err := reuseDial(l.network, l.stunConn.LocalAddr().String(), clientAddr)
	// if err == nil {
	// 	log.Println("amazing! connected in first dial")
	// 	return conn, nil
	// }

	// end := time.Now().Add(10 * time.Second)
	// for time.Now().Before(end) {
	// 	conn, err := reuseDial(l.network, l.localAddr, clientAddr)
	// 	if err != nil {
	// 		// log.Println("failed to accept client", clientAddr, "retrying")
	// 		continue
	// 	}
	// 	return conn, nil
	// }
	// return nil, fmt.Errorf("timeout")
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

	go func() {
		for {
			clientAddr, err := readStr(stunConn)
			if err != nil {
				if io.EOF == err {
					log.Println("failed to read client address current name has already been used")
				} else if strings.Contains(err.Error(), "use of closed network connection") {
					log.Println("listener is closed, stop listening")
				} else {
					log.Println("failed to read client address", err)
				}
				stunConn.Close()
				l.Close()
				return
			}

			_, err = reuseDial(network, stunConn.LocalAddr().String(), clientAddr)
			if err == nil {
				log.Println("amazing! connected in first dial")
			}
		}
	}()
	return &listener{network, l, stunConn}, nil
}
