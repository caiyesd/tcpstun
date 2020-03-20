package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	stun "github.com/caiyesd/tcpstun"
	mux "github.com/hashicorp/yamux"
	cli "gopkg.in/urfave/cli.v1"
)

const default_stun_port = "27310"

func main() {
	log.SetFlags(log.LstdFlags /* | log.Lshortfile*/)

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "tcpstun"
	app.Version = "1.0.1"
	app.Usage = "stun | nc"
	app.Description = "A golang implementation of simple tcp stun protocol"
	app.Commands = []cli.Command{
		{
			Name:   "stun",
			Usage:  "start a stun server",
			Action: startStun,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "local-addr, l",
					Value: "0.0.0.0:" + default_stun_port,
					Usage: "local tcp stun address",
				},
			},
		}, {
			Name:   "nc",
			Usage:  "simple netcat",
			Action: startNc,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "stun-addr, s",
					Value: "127.0.0.1:" + default_stun_port,
					Usage: "local tcp stun address",
				},
				cli.BoolFlag{
					Name:  "client, c",
					Usage: "client mode",
				},
				cli.StringFlag{
					Name:  "local-addr, l",
					Value: "0.0.0.0:0",
					Usage: "local address",
				},
				cli.StringFlag{
					Name:  "name, n",
					Value: "<noname>",
					Usage: "server name",
				},
			},
		},
		{
			Name:   "pm",
			Usage:  "port mapping client",
			Action: startPm,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "stun-addr, s",
					Value: "127.0.0.1:" + default_stun_port,
					Usage: "local tcp stun address",
				},
				cli.BoolFlag{
					Name:  "client, c",
					Usage: "client mode",
				},
				cli.StringFlag{
					Name:  "local-addr, l",
					Value: "0.0.0.0:0",
					Usage: "local address",
				},
				cli.StringFlag{
					Name:  "name, n",
					Value: "<noname>",
					Usage: "server name",
				},
				cli.IntFlag{
					Name:  "port, p",
					Value: 22,
					Usage: "port",
				},
			},
		},
	}

	app.RunAndExitOnError()
}

func startStun(c *cli.Context) error {
	localAddr := c.String("local-addr")
	stun.NewStunServer(localAddr).Start()
	ch := make(chan int)
	<-ch
	return nil
}

func startNc(c *cli.Context) error {
	if c.Bool("client") {
		return startNcClient(c)
	} else {
		return startNcServer(c)
	}
}

func startNcClient(c *cli.Context) error {
	stunAddr := c.String("stun-addr")
	localAddr := c.String("local-addr")
	remoteName := c.String("name")

	conn, err := stun.Dial("tcp4", stunAddr, localAddr, remoteName)
	if err != nil {
		return err
	}
	defer conn.Close()
	handleNcClientConn(conn)
	return nil
}

func startNcServer(c *cli.Context) error {
	stunAddr := c.String("stun-addr")
	localAddr := c.String("local-addr")
	name := c.String("name")

	l, err := stun.Listen("tcp4", stunAddr, localAddr, name)
	if err != nil {
		log.Println("failed to listen at", stunAddr, localAddr, err)
		return err
	}
	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		log.Println("failed to accept at", l.Addr().String(), err)
		return err
	}
	defer conn.Close()
	handleNcServerConn(conn)
	return nil
}

func handleNcClientConn(conn net.Conn) {
	session, err := mux.Client(conn, nil)
	if err != nil {
		log.Println("failed to create mux client session", err)
		conn.Close()
		return
	}
	defer session.Close()

	w, err := session.OpenStream()
	if err != nil {
		log.Println("failed to create mux client r stream", err)
		return
	}
	r, err := session.OpenStream()
	if err != nil {
		log.Println("failed to create mux client w stream", err)
		r.Close()
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		ioCopy(os.Stdout, r)
		wg.Done()
		r.Close()
		log.Println("remote server closed")
	}()
	go func() {
		ioCopy(w, os.Stdin)
		wg.Done()
		w.Close()
	}()

	wg.Wait()
	session.Close()
}

func handleNcServerConn(conn net.Conn) {
	session, err := mux.Server(conn, nil)
	if err != nil {
		log.Println("failed to create mux server session", err)
		conn.Close()
		return
	}
	defer session.Close()

	r, err := session.Accept()
	if err != nil {
		log.Println("failed to create mux server r stream", err)
		return
	}
	w, err := session.Accept()
	if err != nil {
		log.Println("failed to create mux server w stream", err)
		r.Close()
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		ioCopy(os.Stdout, r)
		wg.Done()
		r.Close()
		log.Println("remote client closed")
	}()
	go func() {
		ioCopy(w, os.Stdin)
		wg.Done()
		w.Close()
	}()

	wg.Wait()
	session.Close()
}

func ioCopy(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := make([]byte, 1024*256)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// --------------------------------

func startPm(c *cli.Context) error {
	if c.Bool("client") {
		return startPmClient(c)
	} else {
		return startPmServer(c)
	}
}

func startPmClient(c *cli.Context) error {
	stunAddr := c.String("stun-addr")
	localAddr := c.String("local-addr")
	remoteName := c.String("name")
	port := c.Int("port")

	l, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", "127.0.0.1", port))
	if err != nil {
		return err
	}
	log.Println("listening at", fmt.Sprintf("%s:%d", "127.0.0.1", port))

	conn2, err := l.Accept()
	if err != nil {
		return err
	}

	log.Println("a client arrived")
	defer conn2.Close()
	conn, err := stun.Dial("tcp4", stunAddr, localAddr, remoteName)
	if err != nil {
		return err
	}
	log.Println("connected to remote server", conn.RemoteAddr().String())
	defer conn.Close()

	ch := make(chan int)

	go func() {
		ioCopy(conn, conn2)
		ch <- 1
	}()
	go func() {
		ioCopy(conn2, conn)
		ch <- 1
	}()

	<-ch
	return nil
}

func startPmServer(c *cli.Context) error {
	stunAddr := c.String("stun-addr")
	localAddr := c.String("local-addr")
	name := c.String("name")
	port := c.Int("port")

	l, err := stun.Listen("tcp4", stunAddr, localAddr, name)
	if err != nil {
		log.Println("failed to listen at", stunAddr, localAddr, err)
		return err
	}
	log.Println("listening at stun addr", l.Addr().String())
	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		log.Println("failed to accept at", localAddr, err)
		return err
	}
	log.Println("accepted remote client", conn.RemoteAddr().String())
	defer conn.Close()
	handlePmServerConn(conn, port)
	return nil
}

func handlePmServerConn(conn net.Conn, port int) {
	defer conn.Close()
	log.Println("connecting to", fmt.Sprintf("%s:%d", "127.0.0.1", port))
	conn2, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", "127.0.0.1", port))
	if err != nil {
		log.Println("failed to connect", fmt.Sprintf("%s:%d", "127.0.0.1", port))
		return
	}
	defer conn2.Close()
	log.Println("connected to", fmt.Sprintf("%s:%d", "127.0.0.1", port))

	ch := make(chan int)

	go func() {
		ioCopy(conn, conn2)
		ch <- 1
	}()
	go func() {
		ioCopy(conn2, conn)
		ch <- 1
	}()

	<-ch
}
