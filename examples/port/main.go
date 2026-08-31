package main

import (
	"fmt"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		panic("unexpected listener address")
	}
	fmt.Println(addr.Port)
}
