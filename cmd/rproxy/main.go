package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	N := flag.Int("N", 1, "Size of connection queue")
	server := flag.String("server", ":8080", "Server side listening as 127.0.0.1:8080")
	client := flag.String("client", ":9001", "Client side listening as 127.0.0.1:9001")
	connect := flag.Bool("connect", false, "Connect at client address when connection happens at server side")
	help := flag.Bool("h", false, "Display Help.")
	flag.Parse()

	if *help {
		fmt.Println("Simple Reverse proxy.")
		fmt.Println()
		flag.PrintDefaults()
		return
	}

	serverAddr, err := net.ResolveTCPAddr("tcp", *server)
	if err != nil {
		log.Fatal(err)
	}
	clientAddr, err := net.ResolveTCPAddr("tcp", *client)
	if err != nil {
		log.Fatal(err)
	}

	ctrlc := make(chan os.Signal, 1)
	signal.Notify(ctrlc, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queue := make(chan net.Conn, *N)

	go listener(ctx, serverAddr, queue)
	if *connect {
		go clientConnect(ctx, clientAddr, queue)
	} else {
		go clientListen(ctx, clientAddr, queue)
	}

	<-ctrlc
}

func listener(ctx context.Context, addr *net.TCPAddr, queue chan net.Conn) {
	log.Println("Listening on: ", addr.String())
	listener, err := net.Listen("tcp", addr.String())
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		log.Println("New connection", conn.RemoteAddr())
		if err != nil {
			log.Println("error accepting connection", err)
			continue
		}

		select {
		case queue <- conn:
		case <-ctx.Done():
			return
		default:
			log.Println("Queue full")
			conn.Close()
		}
	}
}

func clientConnect(ctx context.Context, addr *net.TCPAddr, queue chan net.Conn) {
	log.Println("Will redirect to: ", addr.String())
	for {
		select {
		case <-ctx.Done():
			return
		case conn := <-queue:
			client, err := net.Dial("tcp", addr.String())
			if err != nil {
				log.Println(err)
			}
			go handleProxy(ctx, conn, client)
		}
	}
}

func clientListen(ctx context.Context, addr *net.TCPAddr, queue chan net.Conn) {
	clients := make(chan net.Conn)
	go listener(ctx, addr, clients)

	for {
		select {
		case <-ctx.Done():
			return
		case conn := <-queue:
			go waitForClient(ctx, conn, clients)
		}
	}
}

func waitForClient(ctx context.Context, src net.Conn, clients chan net.Conn) {
	defer src.Close()
	select {
	case <-ctx.Done():
		return
	case dst := <-clients:
		handleProxy(ctx, src, dst)
	}
}

func handleProxy(ctx context.Context, src, dst net.Conn) {
	log.Println("Relaying traffic", src.RemoteAddr(), "->", dst.RemoteAddr())
	defer log.Println("Done Relaying traffic", src.RemoteAddr(), "->", dst.RemoteAddr())

	closer := make(chan struct{}, 2)
	go copy(closer, src, dst)
	go copy(closer, dst, src)

	select {
	case <-ctx.Done():
	case <-closer:
	}

	src.Close()
	dst.Close()
}

func copy(closer chan struct{}, dst io.Writer, src io.Reader) {
	_, _ = io.Copy(dst, src)
	closer <- struct{}{} // connection is closed, send signal to stop proxy
}
