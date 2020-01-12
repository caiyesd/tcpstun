# tcpstun

`tcpstun` is an implementation of simple tcp stun protocol.

You can use `tcpstun` to create a peer to peer tcp connection.

## How to build

1. Install golang

2. Download `tcpstun` and dependency

```
go get -d -v github.com/caiyesd/tcpstun
go get -d -v github.com/hashicorp/yamux
go get -d -v gopkg.in/urfave/cli.v1
```

3. Build `tcpstun`

```
cd $GOPATH/src/github.com/caiyesd/tcpstun
./scripts/mkrls.sh
```

And then, you can find all binaries in `./releases/` folder.

## How to use

1. Start a tcp stun server

```
 tcpstun stun -l <0.0.0.0:port>
```

2. Push files to server

```
# server
tcpstun nc -s <stun_server:stun_port> -n <name> </dev/null | tar czvf - 

# client
tar zxvf - /path/to/dir | tcpstun nc -c -s <stun_server:stun_port> -n <name> >/dev/null
```

3. Pull files from server

```
# server
tar zxvf - /path/to/dir | tcpstun nc -s <stun_server:stun_port> -n <name> >/dev/null

# client
tcpstun nc -c -s <stun_server:stun_port> -n <name> </dev/null | tar czvf - 
```
