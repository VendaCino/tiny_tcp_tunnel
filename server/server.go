package main

import (
	"container/list"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var httpcontent = `HTTP/1.1 200 OK
Date: Sat, 29 Jul 2017 06:18:23 GMT
Content-Type: text/html
Connection: Keep-Alive
Server: BWS/1.1
X-UA-Compatible: IE=Edge,chrome=1
BDPAGETYPE: 3
Set-Cookie: BDSVRTM=0; path=/

<html>
<head > 
    <meta charset="UTF-8"/>  
    <title>tcp_tunnel</title> 
</head>

<body>
<h1 style="color:green">proxy client is not connected</h1>

</body>
</html>
`

func setExitChan(exitProgram chan int) {
	sigs := []os.Signal{syscall.SIGINT, syscall.SIGQUIT}
	sigRecv := make(chan os.Signal, 1)
	signal.Notify(sigRecv, sigs...)
	go func() {
		sig := <-sigRecv
		fmt.Printf("Received signal : %s \n", sig)
		exitProgram <- 0
	}()
}

func main() {
	exitProgram := make(chan int, 0)
	//setExitChan(exitProgram)// reset ctrl+c
	addr := "0.0.0.0:8090"
	addr_proxy := "0.0.0.0:8091"
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("err in listen [%s]", err)
	}
	defer listen.Close()
	listen_proxy, err := net.Listen("tcp", addr_proxy)
	if err != nil {
		log.Fatalf("err in listen [%s]", err)
	}
	defer listen_proxy.Close()

	fmt.Printf("start \n")
	var conn_proxy net.Conn = nil
	var goConnProxyListen func()
	var goConnNormalListen func()

	goConnProxyListen = func() {
		fmt.Printf("proxy port waiting for connection... \n")
		conn_proxy, err = listen_proxy.Accept()
		if err != nil {
			log.Fatal(err)
			return
		}
		fmt.Printf("proxy connection is build to %s \n", conn_proxy.RemoteAddr())
		go goConnProxyListen()
	}

	conn_pool := list.New()

	buf := make([]byte, 8192)
	goConnNormalListen = func() {
		fmt.Printf("normal port waiting for connection... \n")
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("normal connection is build to %s ", conn.RemoteAddr())
		conn_pool.PushBack(conn)
		fmt.Printf("conn_pool len %d\n", conn_pool.Len())
		go goConnNormalListen()
		go func() {
			for {
				if conn_proxy == nil {
					len, err := conn.Read(buf)
					if err == nil && len != 0 {
						conn.Write([]byte(httpcontent))
					}
					continue
				}
				len, err := io.Copy(conn_proxy, conn)
				if err != nil {
					fmt.Printf("[error!] c2s %s send %s error [%s]\n", conn.RemoteAddr(), conn_proxy.RemoteAddr(), err)
					break
				}
				if len != 0 {
					fmt.Printf("< c2s %s send %s len [%d]\n", conn.RemoteAddr(), conn_proxy.RemoteAddr(), len)
				}
			}
		}()
	}

	fmt.Printf("start go\n")
	go goConnProxyListen()
	go goConnNormalListen()

	go func() {
		fmt.Printf("start gogo\n")
		buf := make([]byte, 8192)
		for {
			if conn_proxy == nil {
				time.Sleep(time.Millisecond * 10)
				continue
			}
			len, err := conn_proxy.Read(buf)
			if err != nil && err != io.EOF {
				fmt.Printf("[error!] conn_proxy read error [%s]\n", err)
				conn_proxy = nil
				continue
			}
			if len != 0 {
				for e := conn_pool.Front(); e != nil; {
					conn := e.Value.(net.Conn)
					len, err := conn.Write(buf[:len])
					if err != nil && err != io.EOF {
						fmt.Printf("[error!] s2c %s send %s error [%s]\n", conn_proxy.RemoteAddr(), conn.RemoteAddr(), err)
						fmt.Printf("close connection : [%s]\n", conn.RemoteAddr())
						go conn.Close()
						next := e.Next()
						conn_pool.Remove(e)
						e = next
						continue
					}
					if len != 0 {
						fmt.Printf("> s2c %s send %s len [%d]\n", conn_proxy.RemoteAddr(), conn.RemoteAddr(), len)
					}
					e = e.Next()
				}
			}

		}

	}()

	<-exitProgram
}
