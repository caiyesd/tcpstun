package main

import (
	"io"
	"log"
	"net"
	"os"
	"sync"

	stun "github.com/caiyesd/tcpstun"
	mux "github.com/hashicorp/yamux"
	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	log.SetFlags(log.LstdFlags /* | log.Lshortfile*/)

	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "tcpstun"
	app.Version = "1.0.0"
	app.Usage = "stun | server | client"
	app.Description = "A golang implementation of simple tcp stun protocol"
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
			Name:   "server",
			Usage:  "start a netcat server",
			Action: startServer,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "stun-addr, s",
					Value: "127.0.0.1:27310",
					Usage: "local tcp stun address",
				},
				cli.StringFlag{
					Name:  "local-addr, l",
					Value: "0.0.0.0:27311",
					Usage: "local address",
				},
				cli.StringFlag{
					Name:  "name, n",
					Value: "noname",
					Usage: "server name",
				},
			},
		}, {
			Name:   "client",
			Usage:  "start a netcat client",
			Action: startClient,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "stun-addr, s",
					Value: "127.0.0.1:27310",
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
	stun.NewStunServer(localAddr).Start()
	ch := make(chan int)
	<-ch
	return nil
}

func startClient(c *cli.Context) error {
	stunAddr := c.String("stun-addr")
	localAddr := c.String("local-addr")
	remoteName := c.String("name")

	conn, err := stun.Dial("tcp4", stunAddr, localAddr, remoteName)
	if err != nil {
		return err
	}
	log.Println("connected to remote server", conn.RemoteAddr().String())
	defer conn.Close()
	handleClientConn(conn)
	return nil
}

func startServer(c *cli.Context) error {
	stunAddr := c.String("stun-addr")
	localAddr := c.String("local-addr")
	name := c.String("name")

	l, err := stun.Listen("tcp4", stunAddr, localAddr, name)
	if err != nil {
		log.Println("failed to listen at", stunAddr, localAddr, err)
		return err
	}
	log.Println("listening at", localAddr)
	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		log.Println("failed to accept at", localAddr, err)
		return err
	}
	log.Println("accepted remote client", conn.RemoteAddr().String())
	defer conn.Close()
	handleServerConn(conn)
	return nil
}

func handleClientConn(conn net.Conn) {
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

func handleServerConn(conn net.Conn) {
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
