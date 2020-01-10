package main

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
		Timeout:   time.Second * 10,
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

	WriteByte(stunConn, 0) // I'm client
	WriteStr(stunConn, remoteName)
	targetAddr, err := ReadStr(stunConn)
	if err != nil {
		log.Println("failed to dial stun server", stunAddr, err)
		return nil, err
	}
	if targetAddr == "" {
		log.Println("remote", remoteName, "not found")
		return nil, fmt.Errorf("remote %s not found", remoteName)
	}
	end := time.Now().Add(10 * time.Second)
	for time.Now().Before(end) {
		conn, err := reuseDial(network, localAddr, targetAddr)
		if err != nil {
			log.Println("failed to dial target server", targetAddr, "retrying")
			continue
		}
		return conn, nil
	}
	return nil, fmt.Errorf("timeout")
}
