package main

import (
	"bytes"
	"container/list"
	"log"
	"net"
)

func main() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
	log.Println("Hello Server!")

	clientList := list.New()
	in := make(chan string)
	go IOHandler(in, clientList)

	service := ":9988"
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	if err != nil {
		log.Println("Error: Could not resolve address")
	} else {
		netListen, err := net.Listen(tcpAddr.Network(), tcpAddr.String())
		if err != nil {
			log.Println(err)
		} else {
			defer netListen.Close()

			for {
				log.Println("Waiting for clients")
				connection, err := netListen.Accept()
				if err != nil {
					log.Println("Client error:", err)
				} else {
					go ClientHandler(connection, in, clientList)
				}
			}
		}
	}
}

func IOHandler(Incoming <-chan string, clientList *list.List) {
	for {
		log.Println("IOHandler: Waiting for input")
		input := <-Incoming
		log.Println("IOHandler: Handling", input)
		for e := clientList.Front(); e != nil; e = e.Next() {
			client := e.Value.(Client)
			client.Incoming <- input
		}
	}
}

func ClientReader(client *Client) {

	for {
		buffer := make([]byte, 2048)
		bytesRead, flag := client.Read(buffer)
		if !flag {
			break
		}

		if bytes.Equal(buffer, []byte("/quit")) {
			client.Close()
			break
		}
		log.Println("ClientReader received", client.Name, ">", string(buffer[0:bytesRead]))
		send := client.Name + "> " + string(buffer[0:bytesRead])
		client.Outgoing <- send
	}

	client.Outgoing <- client.Name + " has left chat"
	log.Println("ClientReader stopped for", client.Name)
}

func ClientSender(client *Client) {
	for {
		select {
		case buffer := <-client.Incoming:
			log.Println("ClientSender sendig", string(buffer), "to", client.Name)
			count := 0
			for i := 0; i < len(buffer); i++ {
				if buffer[i] == 0x00 {
					break
				}
				count++
			}
			log.Println("Send size: ", count)
			client.Conn.Write([]byte(buffer)[0:count])
		case <-client.Quit:
			log.Println("Client", client.Name, "quitting")
			client.Conn.Close()
			break
		}
	}
}

func ClientHandler(conn net.Conn, ch chan string, clientList *list.List) {
	buffer := make([]byte, 1024)
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		log.Println("Client connection error:", err)
	}

	name := string(buffer[0:bytesRead])
	newClient := &Client{name, make(chan string), ch, conn, make(chan bool), clientList}

	go ClientReader(newClient)
	go ClientSender(newClient)

	clientList.PushBack(*newClient)
	ch <- string(name + " has joined the chat")
}

type Client struct {
	Name       string
	Incoming   chan string
	Outgoing   chan string
	Conn       net.Conn
	Quit       chan bool
	ClientList *list.List
}

func (c *Client) Read(buffer []byte) (int, bool) {
	bytesRead, err := c.Conn.Read(buffer)
	if err != nil {
		c.Close()
		log.Println(err)
		return 0, false
	}
	log.Println("Read", bytesRead, "bytes")
	return bytesRead, true
}

func (c *Client) Close() {
	c.Quit <- true
	c.Conn.Close()
	c.RemoveMe()
}

func (c *Client) Equal(other *Client) bool {
	if bytes.Equal([]byte(c.Name), []byte(other.Name)) {
		if c.Conn == other.Conn {
			return true
		}
	}
	return false
}

func (c *Client) RemoveMe() {
	for entry := c.ClientList.Front(); entry != nil; entry = entry.Next() {
		client := entry.Value.(Client)
		if c.Equal(&client) {
			log.Println("RemoveMe:", c.Name)
			c.ClientList.Remove(entry)
		}
	}
}
