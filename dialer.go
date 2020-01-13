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
		log.Println("name", remoteName, "not found")
		return nil, fmt.Errorf("name %s not found", remoteName)
	}

	return connect(network, localAddr, targetAddr)
}
