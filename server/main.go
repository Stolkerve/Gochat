package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"

	goaway "github.com/TwiN/go-away"
)

type MsgType uint

const (
	NewConnection MsgType = iota
	Ban
	CloseConnection
	NewMsg
)

type Msg struct {
	Conn net.Conn
	Text string
	Type MsgType
}

type Server struct {
	Clients       map[string]net.Conn
	BannedClients map[string]string
	InMgs         chan Msg
}

type Client struct {
	Conn        net.Conn
	OutMgs      chan Msg
	profanities uint
	msgRate     uint
	lastMsgDate time.Time
}

func handleServer(server Server) {
	for {
		msg := <-server.InMgs
		switch msg.Type {
		case NewConnection:
			addr := msg.Conn.RemoteAddr().String()
			if v, ok := server.BannedClients[addr]; ok {
				log.Printf("Denay conecction of %s for %s", addr, v)
				break
			}

			newCli := Client{
				Conn:   msg.Conn,
				OutMgs: server.InMgs,
			}
			server.Clients[addr] = msg.Conn
			log.Printf("%s is connected", addr)

			go handleClient(newCli)
		case NewMsg:
			{
				for addr, c := range server.Clients {
					if addr != msg.Conn.RemoteAddr().String() {
						if _, err := c.Write([]byte(msg.Text + "\n")); err != nil {
							server.InMgs <- Msg{
								Conn: c,
								Type: CloseConnection,
							}
						}
					}
				}
			}
		case CloseConnection:
			{
				closeConnection(msg.Conn, msg.Conn.RemoteAddr().String(), &server)
			}
		case Ban:
			{
				addr := msg.Conn.RemoteAddr().String()
				server.BannedClients[addr] = msg.Text
				closeConnection(msg.Conn, addr, &server)
				log.Println(server.BannedClients)
			}
		}
	}
}

func closeConnection(conn net.Conn, addr string, server *Server) {
	conn.Close()
	log.Printf("%s is disconnected", addr)
	delete(server.Clients, addr)
}

func handleClient(cli Client) {
	reader := bufio.NewReader(cli.Conn)

	for {
		line, _, err := reader.ReadLine()
		fmt.Println(line)
		if err != nil {
			cli.OutMgs <- Msg{
				Conn: cli.Conn,
				Type: CloseConnection,
			}
			return
		}

		deltaTime := cli.lastMsgDate.Sub(time.Now()).Abs()

		if deltaTime < time.Millisecond*time.Duration(500) {
			cli.msgRate += 1
		} else {
			cli.msgRate = 0
		}

		if cli.msgRate > 50 {
			cli.OutMgs <- Msg{
				Conn: cli.Conn,
				Type: Ban,
				Text: "Span",
			}
			return
		}

		cli.lastMsgDate = time.Now()

		lineStr := string(line)
		if goaway.IsProfane(lineStr) {

			if deltaTime < time.Second*time.Duration(1) {
				cli.profanities += 1
			} else {
				cli.profanities = 0
			}
			if cli.profanities > 50 {
				cli.OutMgs <- Msg{
					Conn: cli.Conn,
					Type: Ban,
					Text: "Profanity",
				}
				return
			}
			lineStr = goaway.Censor(lineStr)
		}

		cli.OutMgs <- Msg{
			Conn: cli.Conn,
			Type: NewMsg,
			Text: lineStr,
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatalln(err.Error())
		log.Fatalln("No se pudo enlazar con el puerto 8080")
	}

	server := Server{
		Clients:       make(map[string]net.Conn),
		InMgs:         make(chan Msg),
		BannedClients: make(map[string]string),
	}

	go handleServer(server)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("No se pudo aceptar una coneccion")
		}

		server.InMgs <- Msg{
			Conn: conn,
			Type: NewConnection,
		}
	}
}
