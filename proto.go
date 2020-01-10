package main

import (
	"fmt"
	"net"
)

func WriteByte(conn net.Conn, b byte) error {
	_, err := conn.Write([]byte{b})
	if err != nil {
		return err
	}
	return nil
}

func ReadByte(conn net.Conn) (byte, error) {
	buffer := make([]byte, 1)
	_, err := conn.Read(buffer)
	if err != nil {
		return 0, err
	}
	return buffer[0], nil
}

func WriteStr(conn net.Conn, str string) error {
	l := byte(len([]byte(str)))
	if int(l) != len(str) || l == 0 {
		return fmt.Errorf("invalid string %s:", str)
	}
	_, err := conn.Write(append([]byte{l}, []byte(str)...))
	if err != nil {
		return err
	}
	return nil
}

func ReadStr(conn net.Conn) (string, error) {
	buffer := make([]byte, 256)
	_, err := conn.Read(buffer[:1])
	if err != nil {
		return "", err
	}
	l := buffer[0]
	n, err := conn.Read(buffer[1 : 1+l])
	if err != nil {
		return "", err
	}
	if n != int(l) {
		return "", fmt.Errorf("invalid string bytes")
	}
	return string(buffer[1 : 1+l]), nil
}
