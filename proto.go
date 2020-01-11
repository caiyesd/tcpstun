package tcpstun

import (
	"fmt"
	"net"
)

func writeByte(conn net.Conn, b byte) error {
	_, err := conn.Write([]byte{b})
	if err != nil {
		return err
	}
	return nil
}

func readByte(conn net.Conn) (byte, error) {
	buffer := make([]byte, 1)
	_, err := conn.Read(buffer)
	if err != nil {
		return 0, err
	}
	return buffer[0], nil
}

func writeStr(conn net.Conn, str string) error {
	l := byte(len([]byte(str)))
	if int(l) != len(str) || l == 0 {
		return fmt.Errorf("invalid string %s", str)
	}
	_, err := conn.Write(append([]byte{l}, []byte(str)...))
	if err != nil {
		return err
	}
	return nil
}

func readStr(conn net.Conn) (string, error) {
	buffer := make([]byte, 256)
	_, err := conn.Read(buffer[:1])
	if err != nil {
		return "", err
	}
	l := buffer[0]
	s, err := conn.Read(buffer[1 : 1+l])
	if err != nil {
		return "", err
	}
	if s != int(l) {
		return "", fmt.Errorf("invalid string bytes")
	}
	return string(buffer[1 : 1+l]), nil
}
