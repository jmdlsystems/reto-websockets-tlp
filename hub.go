package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Hub mantiene el conjunto de clientes activos y difunde mensajes
type Hub struct {
	clients      map[*Client]bool
	broadcast    chan *Message
	register     chan *Client
	unregister   chan *Client
	clientsMutex sync.RWMutex
}

// NewHub crea un nuevo hub de chat
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256), // Buffer para evitar bloqueos
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
	}
}

// Run ejecuta el bucle principal del hub
func (h *Hub) Run() {
	log.Println("Hub iniciado - procesando eventos...")
	for {
		select {
		case client := <-h.register:
			// Registrar nuevo cliente
			h.registerClient(client)

		case client := <-h.unregister:
			// Desregistrar cliente
			h.unregisterClient(client)

		case message := <-h.broadcast:
			// Difundir mensaje a todos los clientes
			h.broadcastMessage(message)
		}
	}
}

// registerClient registra un nuevo cliente
func (h *Hub) registerClient(client *Client) {
	h.clientsMutex.Lock()
	for c := range h.clients {
		if c.username == client.username {
			h.clientsMutex.Unlock()
			// Enviar mensaje de error y cerrar la conexión
			go func() {
				errMsg := NewSystemMessage("Este nombre de usuario ya está conectado. Elige otro.")
				client.send <- errMsg
				close(client.send)
			}()
			return
		}
	}
	h.clients[client] = true
	clientCount := len(h.clients)
	h.clientsMutex.Unlock()

	log.Printf("Cliente %s conectado. Total de clientes: %d", client.username, clientCount)
	// Notificar a todos los clientes que alguien se conectó
	systemMessage := NewSystemMessage(fmt.Sprintf("%s se ha conectado", client.username))
	// Envío asíncrono para evitar bloqueos
	go func() {
		select {
		case h.broadcast <- systemMessage:
		case <-time.After(time.Second):
			log.Printf("Timeout enviando mensaje de conexión para %s", client.username)
		}
	}()
}

// unregisterClient desregistra un cliente
func (h *Hub) unregisterClient(client *Client) {
	h.clientsMutex.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		//Cierre seguro del canal
		select {
		case <-client.send:
		default:
			close(client.send)
		}
		clientCount := len(h.clients)
		h.clientsMutex.Unlock()

		log.Printf("Cliente %s desconectado. Total de clientes: %d", client.username, clientCount)
		// Notificar a todos los clientes que alguien se desconectó
		systemMessage := NewSystemMessage(fmt.Sprintf("%s se ha desconectado", client.username))
		//Envío asíncrono
		go func() {
			select {
			case h.broadcast <- systemMessage:
			case <-time.After(time.Second):
				log.Printf("Timeout enviando mensaje de desconexión para %s", client.username)
			}
		}()
	} else {
		h.clientsMutex.Unlock()
	}
}

// broadcastMessage difunde un mensaje a todos los clientes conectados
func (h *Hub) broadcastMessage(message *Message) {
	h.clientsMutex.RLock()
	// Crear una copia de los clientes para evitar problemas de concurrencia
	copiaClientes := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		copiaClientes = append(copiaClientes, client)
	}

	h.clientsMutex.RUnlock()

	log.Printf("Difundiendo mensaje a %d clientes: [%s] %s",
		len(copiaClientes), message.Username, message.MessageContent)

	// Lista de clientes que fallan para desconectar después
	var clientesFallados []*Client

	// Enviar el mensaje a todos los clientes
	for _, client := range copiaClientes {
		select {
		case client.send <- message:
			log.Printf("Mensaje enviado a %s", client.username)
		case <-time.After(100 * time.Millisecond):
			// CORREGIDO: Timeout en lugar de default inmediato
			log.Printf("Timeout enviando mensaje a %s, marcando para desconexión", client.username)
			clientesFallados = append(clientesFallados, client)
		}
	}

	//  Desconectar clientes que fallaron de forma asíncrona
	if len(clientesFallados) > 0 {
		go func() {
			for _, client := range clientesFallados {
				select {
				case h.unregister <- client:
				case <-time.After(time.Second):
					log.Printf("Timeout desregistrando cliente %s", client.username)
				}
			}
		}()
	}
}

// GetClientCount retorna el número de clientes conectados
func (h *Hub) GetClientCount() int {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()
	return len(h.clients)
}

// Metodo para obtener lista de usuarios conectados
func (h *Hub) GetConnectedClients() []string {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()
	usernames := make([]string, 0, len(h.clients))
	for client := range h.clients {
		usernames = append(usernames, client.username)
	}

	return usernames
}
