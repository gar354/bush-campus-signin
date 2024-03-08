package broadcast

import (
	"log"
	"slices"
)

type Broadcaster struct {
	channel          chan []byte
	running          bool
	clientRegister   chan chan []byte
	clientDeRegister chan chan []byte
}

func New() Broadcaster {
	return Broadcaster{
		channel: make(chan []byte),
		clientRegister: make(chan chan []byte, 4),
		clientDeRegister: make(chan chan []byte, 4),
		running: false,
	}
}

func (b *Broadcaster) Stop() {
	close(b.channel)
	b.running = false
}

func (b *Broadcaster) Send(data []byte) {
	b.channel <- data
}

func (b *Broadcaster) Register() chan []byte {
	client := make(chan []byte, 1)
	b.clientRegister <- client
	return client
}

func (b *Broadcaster) DeRegister(client chan []byte) {
	b.clientDeRegister <- client
}

func (b *Broadcaster) Serve() {
	if b.running {
		log.Println("Broadcast: Warning: broadcaster is already running")
		return
	}
	b.running = true
	var clients []chan []byte

	for {
		select {
		case client, ok := <-b.clientRegister:
			if !ok {
				log.Println("Broadcast: clientRegister channel closed")
				return
			}
			log.Println("Broadcast: broadcast client sent.")
			clients = append(clients, client)
		case client, ok := <-b.clientDeRegister:
			if !ok {
				log.Println("Broadcast: clientDeRegister channel closed")
				return
			}
			index := slices.Index(clients, client)
			clients = slices.Delete(clients, index, index+1)
			close(client)
			log.Println("Broadcast: broadcast client deregistered")

		case data, ok := <-b.channel:
			if !ok {
				log.Println("broadcast channel closed")
				return
			}
			for _, c := range clients {
				c <- data
			}
		}
	}
}

