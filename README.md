# sshtunnel

[![](https://godoc.org/github.com/sgreben/sshtunnel?status.svg)](http://godoc.org/github.com/sgreben/sshtunnel) [![](https://goreportcard.com/badge/github.com/sgreben/sshtunnel/goreportcard)](https://goreportcard.com/report/github.com/sgreben/sshtunnel) [![cover.run](https://cover.run/go/github.com/sgreben/sshtunnel.svg?style=flat&tag=golang-1.10)](https://cover.run/go?tag=golang-1.10&repo=github.com%2Fsgreben%2Fsshtunnel) [![Build Status](https://travis-ci.org/sgreben/sshtunnel.svg?branch=master)](https://travis-ci.org/sgreben/sshtunnel)

Go library providing a dialer for SSH-tunneled TCP and Unix domain socket connections. Please note the [**limitations**](#limitations) below.


- [Get it](#get-it)
- [Use it](#use-it)
	- [Docs](#docs)
	- [If you have an existing SSH connection...](#if-you-have-an-existing-ssh-connection)
	- [Toy example](#toy-example)
	- [Bigger example](#bigger-example)
- [Limitations](#limitations)

## Get it

```sh
go get -u "github.com/sgreben/sshtunnel"
```

## Use it

```go
import "github.com/sgreben/sshtunnel"
```

### Docs

[![](https://godoc.org/github.com/sgreben/sshtunnel?status.svg)](http://godoc.org/github.com/sgreben/sshtunnel)


### If you have an existing SSH connection...

You can directly use the [`sshtunnel.DialTCP` and `sshtunnel.DialUnix` DialFuncs](https://godoc.org/github.com/sgreben/sshtunnel#DialFunc) to obtain a `net.Conn` that goes through the tunnel.

The other functions (`Dial*`, `ReDial*`, `Listen*`) provide additional convenience features such as redialling dropped connections, or serving the tunnel locally.

### Toy example

```go
package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh"
	"github.com/sgreben/sshtunnel"
)

func main() {
	// Connect to "google.com:80" via a tunnel to "ubuntu@my-ssh-server-host:22"
	keyPath := "private-key.pem"
	authConfig := sshtunnel.ConfigAuth{
		Keys:     []sshtunnel.KeySource{{Path: &keyPath}},
	}
	sshAuthMethods, _ := authConfig.Methods()
	clientConfig := ssh.ClientConfig{
		User: "ubuntu",
		Auth: sshAuthMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	tunnelConfig := sshtunnel.Config{
		SSHAddr: "my-ssh-server-host:22",
		SSHClient: &clientConfig,
	}
	conn, _, err := sshtunnel.Dial("tcp", "google.com:80", &tunnelConfig)
	if err != nil {
		panic(err)
	}
	// Do things with conn
	fmt.Fprintln(conn, "GET /")
	io.Copy(os.Stdout, conn)
}
```

### Bigger example

See [docker-compose-hosts](https://github.com/sgreben/docker-compose-hosts).

## Limitations

- **No tests**; want some - write some.
