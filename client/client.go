package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ByteHandle func([]byte) []byte

func connect2conn(conn1 net.Conn, conn2 net.Conn, buffSize int, timeout time.Duration,
	c12 ByteHandle, c21 ByteHandle) {
	buf := make([]byte, buffSize)
	resend := func(c1 net.Conn, c2 net.Conn, bytehandle ByteHandle) {
		for {
			if timeout != 0 {
				c1.SetReadDeadline(time.Now().Add(timeout * time.Second))
			}
			len, err := reSendTo(c1, c2, buf, bytehandle)
			if err != nil && err != io.EOF {
				fmt.Printf("%s send %s error [%s]\n", c1.RemoteAddr(), c2.RemoteAddr(), err)
				break
			}
			if len != 0 {
				fmt.Printf("%s send %s len [%d]\n", c1.RemoteAddr(), c2.RemoteAddr(), len)
			}
		}
		c1.Close()
		c2.Close()
	}
	c1 := make(chan int)
	c2 := make(chan int)
	go func() {
		resend(conn1, conn2, c12)
		c1 <- 1
	}()
	go func() {
		resend(conn2, conn1, c21)
		c2 <- 1
	}()
	<-c1
	<-c2
}

func reSendTo(conn_orgin net.Conn, conn_aim net.Conn, buf []byte, bytehandle ByteHandle) (n int, err error) {
	len, err := conn_orgin.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if len != 0 {
		if bytehandle != nil {
			len, err = conn_aim.Write(bytehandle(buf[:len]))
		} else {
			len, err = conn_aim.Write(buf[:len])
		}
		if err != nil && err != io.EOF {
			return 0, err
		}
	}
	return len, nil
}

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
	server_addr := "0.0.0.0:8091"
	target_addr := "0.0.0.0:8081"
	for {
		conn, err := net.Dial("tcp", server_addr)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("server connection is build \n")
		defer conn.Close()
		conn_target, err := net.Dial("tcp", target_addr)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("target connection is build \n")
		defer conn_target.Close()

		httpHandle := func(buf []byte) []byte {
			//newstr := string(buf)
			//fmt.Printf("\n-----------------------\n%s\n---------end--------\n", newstr)
			//return []byte(newstr)
			return buf
		}
		httpReturnHandle := func(buf []byte) []byte {
			//newstr := string(buf)
			//fmt.Printf("\n-----------------------\n%s\n---------end--------\n", newstr)
			return buf
		}
		connect2conn(conn, conn_target, 8192, 0, httpHandle, httpReturnHandle)
	}
}
