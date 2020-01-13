package tcpstun

import (
	"fmt"
	"log"
	"net"
	"time"

	reuse "github.com/libp2p/go-reuseport"
)

func reuseDial(network, laddr, raddr string) (net.Conn, error) {
	nla, err := reuse.ResolveAddr(network, laddr)
	if err != nil {
		return nil, err
	}
	d := net.Dialer{
		Control:   reuse.Control,
		LocalAddr: nla,
		Timeout:   time.Second * 5,
	}
	return d.Dial(network, raddr)
}

func Dial(network, stunAddr, localAddr, remoteName string) (net.Conn, error) {
	stunConn, err := reuseDial(network, localAddr, stunAddr)
	if err != nil {
		log.Println("failed to dail stun server", err)
		return nil, err
	}
	defer stunConn.Close()

	localAddr = stunConn.LocalAddr().String()
	log.Println("using local address", localAddr)

	writeByte(stunConn, 0) // I'm client
	writeStr(stunConn, remoteName)
	targetAddr, err := readStr(stunConn)
	if err != nil {
		log.Println("failed to dial stun server", stunAddr, err)
		return nil, err
	}
	if targetAddr == "" {
		log.Println("remote", remoteName, "not found")
		return nil, fmt.Errorf("remote %s not found", remoteName)
	}

	log.Println("sleep 3 seconds to wait remote server to connect first")
	time.Sleep(3 * time.Second)

	log.Println("try to dail server", targetAddr)
	conn, err := reuseDial(network, localAddr, targetAddr)
	if err != nil {
		log.Println("failed to dial server", targetAddr, err)
		log.Println("try to listen as server at", localAddr)

		l, err := reuse.Listen(network, localAddr)
		if err != nil {
			log.Println("failed to listen as server at", localAddr, err)
			return nil, err
		}
		defer l.Close()

		// log.Println("try to accept for 6 seconds")
		// if l.(*net.TCPListener).SetDeadline(time.Now().Add(6 * time.Second)); err != nil {
		// 	log.Println("failed to SetDeadline on listener", err)
		// 	return nil, err
		// }
		go func() {
			time.Sleep(6 * time.Second)
			conn, _ := net.DialTimeout("tcp4", l.Addr().String(), time.Second*5)
			if conn != nil {
				conn.Close()
			}
		}()
		conn, err = l.Accept()
		if err != nil {
			log.Println("failed to accept, give up", err)
			return nil, err
		} else if conn.RemoteAddr().String() != targetAddr {
			conn.Close()
			log.Println("try to dial as client")
			conn, err = reuseDial(network, stunConn.LocalAddr().String(), targetAddr)
			if err != nil {
				log.Println("failed to dial as client, give up", err)
				return nil, err
			}
		}
	}
	return conn, nil
}
