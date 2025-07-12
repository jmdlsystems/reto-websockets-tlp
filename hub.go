package main

import (
	"fmt"
	"log"
	"sync"
)

// Hub mantiene el conjunto de clientes activos y difunde mensajes
type Hub struct {
	// Clientes registrados
	clients map[*Client]bool
	
	// Canal para mensajes que se difunden a todos los clientes
	broadcast chan *Message
	
	// Canal para registrar nuevos clientes
	register chan *Client
	
	// Canal para desregistrar clientes
	unregister chan *Client
	
	// Mutex para proteger el acceso concurrente a los clientes
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
	h.clients[client] = true
	clientCount := len(h.clients)
	h.clientsMutex.Unlock()
	
	log.Printf("Cliente %s conectado. Total de clientes: %d", client.username, clientCount)
	
	// Notificar a todos los clientes que alguien se conectó
	systemMessage := NewSystemMessage(fmt.Sprintf("%s se ha conectado", client.username))
	
	// Enviar la notificación de forma asíncrona para evitar bloqueos
	go func() {
		select {
		case h.broadcast <- systemMessage:
		default:
			log.Printf("Canal de broadcast lleno, no se pudo enviar mensaje de conexión")
		}
	}()
}

// unregisterClient desregistra un cliente
func (h *Hub) unregisterClient(client *Client) {
	h.clientsMutex.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		
		// Cerrar el canal de envío del cliente de forma segura
		select {
		case <-client.send:
			// Canal ya cerrado
		default:
			close(client.send)
		}
		
		clientCount := len(h.clients)
		h.clientsMutex.Unlock()
		
		log.Printf("Cliente %s desconectado. Total de clientes: %d", client.username, clientCount)
		
		// Notificar a todos los clientes que alguien se desconectó
		systemMessage := NewSystemMessage(fmt.Sprintf("%s se ha desconectado", client.username))
		
		// Enviar la notificación de forma asíncrona
		go func() {
			select {
			case h.broadcast <- systemMessage:
			default:
				log.Printf("Canal de broadcast lleno, no se pudo enviar mensaje de desconexión")
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
	clientsCopy := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clientsCopy = append(clientsCopy, client)
	}
	
	h.clientsMutex.RUnlock()
	
	log.Printf("[broadcastMessage] Difundiendo mensaje a %d clientes: %+v", len(clientsCopy), message)
	
	// Lista de clientes que fallan para desconectar después
	var failedClients []*Client
	
	// Enviar el mensaje a todos los clientes
	for _, client := range clientsCopy {
		log.Printf("[broadcastMessage] Intentando enviar a %s", client.username)
		
		select {
		case client.send <- message:
			log.Printf("[broadcastMessage] Mensaje enviado a %s", client.username)
		default:
			log.Printf("[broadcastMessage] Canal lleno o cerrado para %s, marcando para desconexión", client.username)
			failedClients = append(failedClients, client)
		}
	}
	
	// Desconectar clientes que fallaron de forma asíncrona para evitar deadlocks
	if len(failedClients) > 0 {
		go func() {
			for _, client := range failedClients {
				select {
				case h.unregister <- client:
				default:
					log.Printf("No se pudo desregistrar cliente %s", client.username)
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

// Método auxiliar para cerrar un cliente de forma segura
func (h *Hub) SafeCloseClient(client *Client) {
	select {
	case h.unregister <- client:
	default:
		log.Printf("No se pudo desregistrar cliente %s a través del canal", client.username)
	}
}