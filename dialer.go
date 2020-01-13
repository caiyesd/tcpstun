package tcpstun

import (
	"fmt"
	"log"
	"math/rand"
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
		Timeout:   time.Second * 1,
	}
	return d.Dial(network, raddr)
}

func Dial(network, stunAddr, localAddr, remoteName string) (net.Conn, error) {
	addr, err := reuse.ResolveAddr(network, localAddr)
	if err != nil {
		return nil, err
	}
	localAddr = addr.String()
	log.Println("using local address", localAddr)

	stunConn, err := reuseDial(network, localAddr, stunAddr)
	if err != nil {
		log.Println("failed to dail stun server", err)
		return nil, err
	}
	defer stunConn.Close()

	writeByte(stunConn, 0) // I'm client
	writeStr(stunConn, remoteName)
	targetAddr, err := readStr(stunConn)
	if err != nil {
		log.Println("failed to read stun server", stunAddr, err)
		stunConn.Close()
		return nil, err
	}
	if targetAddr == "" {
		log.Println("remote", remoteName, "not found")
		stunConn.Close()
		return nil, fmt.Errorf("remote %s not found", remoteName)

	}
	stunConn.Close()

	end := time.Now().Add(60 * time.Second)
	for time.Now().Before(end) {
		conn, err := reuseDial(network, localAddr, targetAddr)
		if err != nil {
			log.Println("failed to dial target server", targetAddr, "retrying")
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			continue
		}
		return conn, nil
	}
	return nil, fmt.Errorf("timeout to dial")
}
