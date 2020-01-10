package main

import (
	"io"
	"log"
	"net"
	"os"

	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	log.SetFlags(log.LstdFlags /* | log.Lshortfile*/)

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "tcpstun"
	app.Version = "1.0.0"
	app.Usage = "stun | nc"
	app.Description = "A golang implementation of simple tcpstun protocol"
	app.Commands = []cli.Command{
		{
			Name:   "stun",
			Usage:  "start a stun server",
			Action: startStun,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "local-addr, l",
					Value: "0.0.0.0:27310",
					Usage: "local tcp stun address",
				},
			},
		}, {
			Name:   "client",
			Usage:  "start a client",
			Action: startClient,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "stun-addr, s",
					Value: "0.0.0.0:27310",
					Usage: "local tcp stun address",
				},
				cli.StringFlag{
					Name:  "local-addr, l",
					Value: "0.0.0.0:27311",
					Usage: "local address",
				},
				cli.StringFlag{
					Name:  "remote-name, r",
					Value: "noname",
					Usage: "remote server name",
				},
			},
		}, {
			Name:   "server",
			Usage:  "start a server",
			Action: startServer,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "stun-addr, s",
					Value: "0.0.0.0:27310",
					Usage: "local tcp stun address",
				},
				cli.StringFlag{
					Name:  "local-addr, l",
					Value: "0.0.0.0:27312",
					Usage: "local address",
				},
				cli.StringFlag{
					Name:  "name, n",
					Value: "noname",
					Usage: "server name",
				},
			},
		},
	}

	app.RunAndExitOnError()
}

func startStun(c *cli.Context) error {
	localAddr := c.String("local-addr")
	return Stun(localAddr)
}

func startClient(c *cli.Context) error {
	stunAddr := c.String("stun-addr")
	localAddr := c.String("local-addr")
	remoteName := c.String("remote-name")

	conn, err := Dial("tcp4", stunAddr, localAddr, remoteName)
	if err != nil {
		return err
	}
	defer conn.Close()
	handleConn(conn)
	return nil
}

func startServer(c *cli.Context) error {
	stunAddr := c.String("stun-addr")
	localAddr := c.String("local-addr")
	name := c.String("name")

	l, err := Listen("tcp4", stunAddr, localAddr, name)
	if err != nil {
		return err
	}
	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		return err
	}
	defer conn.Close()
	handleConn(conn)
	return nil
}

func handleConn(conn net.Conn) {
	chan_to_stdout := ioCopy(conn, os.Stdout)
	chan_to_remote := ioCopy(os.Stdin, conn)
	select {
	case <-chan_to_stdout:
		log.Println("Remote connection is closed")
	case <-chan_to_remote:
		log.Println("Local program is terminated")
	}
}

func ioCopy(src io.Reader, dst io.Writer) <-chan int {
	buf := make([]byte, 1024)
	ch := make(chan int)
	go func() {
		defer func() {
			if conn, ok := dst.(net.Conn); ok {
				conn.Close()
				log.Printf("Connection from %v is closed\n", conn.RemoteAddr())
			}
			ch <- 0 // Notify that processing is finished
		}()
		for {
			var nBytes int
			var err error
			nBytes, err = src.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Read error: %s\n", err)
				}
				break
			}
			_, err = dst.Write(buf[0:nBytes])
			if err != nil {
				log.Fatalf("Write error: %s\n", err)
			}
		}
	}()
	return ch
}
