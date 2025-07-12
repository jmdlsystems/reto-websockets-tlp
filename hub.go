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
	h.clientsMutex.Unlock()
	
	// Notificar a todos los clientes que alguien se conectó
	systemMessage := NewSystemMessage(fmt.Sprintf("%s se ha conectado", client.username))
	h.broadcast <- systemMessage
	
	log.Printf("Cliente %s conectado. Total de clientes: %d", client.username, len(h.clients))
}

// unregisterClient desregistra un cliente
func (h *Hub) unregisterClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		
		// Notificar a todos los clientes que alguien se desconectó
		systemMessage := NewSystemMessage(fmt.Sprintf("%s se ha desconectado", client.username))
		
		// Enviar la notificación después de liberar el mutex
		go func() {
			h.broadcast <- systemMessage
		}()
		
		log.Printf("Cliente %s desconectado. Total de clientes: %d", client.username, len(h.clients))
	}
}

// broadcastMessage difunde un mensaje a todos los clientes conectados
func (h *Hub) broadcastMessage(message *Message) {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()
	
	// Crear una copia de los clientes para evitar problemas de concurrencia
	clientsCopy := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clientsCopy = append(clientsCopy, client)
	}

	log.Printf("[broadcastMessage] Difundiendo mensaje a %d clientes: %+v", len(clientsCopy), message)
	// Enviar el mensaje a todos los clientes
	for _, client := range clientsCopy {
		log.Printf("[broadcastMessage] Intentando enviar a %s", client.username)
		select {
		case client.send <- message:
			log.Printf("[broadcastMessage] Mensaje enviado a %s", client.username)
		default:
			log.Printf("[broadcastMessage] Canal lleno o cerrado para %s, cerrando cliente", client.username)
			h.forceCloseClient(client)
		}
	}
}

// forceCloseClient cierra forzadamente un cliente que no responde
func (h *Hub) forceCloseClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		client.conn.Close()
		
		log.Printf("Cliente %s cerrado forzadamente", client.username)
	}
}

// GetClientCount retorna el número de clientes conectados
func (h *Hub) GetClientCount() int {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()
	return len(h.clients)
}