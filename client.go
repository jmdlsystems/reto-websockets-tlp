package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"encoding/json"
	"log"


)

// Configuración del upgrader WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permitir todas las conexiones por simplicidad
		// En producción, esto debería ser más restrictivo
		return true
	},
}


type Client struct {
	// Hub de chat al que pertenece este cliente
	hub *Hub
	// Conexión WebSocket
	conn *websocket.Conn
	// Canal para enviar mensajes al cliente
	send chan *Message
	// Nombre de usuario del cliente
	username string
}

// NewClient crea un nuevo cliente
func NewClient(hub *Hub, conn *websocket.Conn, username string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan *Message, 256),
		username: username,
	}
}

// readPump maneja la lectura de mensajes del cliente
func (c *Client) readPump() {
	defer func() {
		log.Printf("[readPump] Cliente %s desconectado, cerrando conexión", c.username)
		// Notificar al hub que el cliente se desconectó
		c.hub.unregister <- c
		c.conn.Close()
	}()
	
	// Configurar timeouts para la conexión
	c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})
	
	for {
		// Leer mensaje del cliente
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[readPump] error: %v", err)
			}
			break
		}
		log.Printf("[readPump] Mensaje recibido de %s: %s", c.username, string(messageBytes))
		// Parsear el mensaje JSON
		var rawMessage map[string]interface{}
		if err := json.Unmarshal(messageBytes, &rawMessage); err != nil {
			log.Printf("[readPump] Error al Parsear Mensaje: %v", err)
			continue
		}
		// Crear el mensaje con el username del cliente
		message := NewUserMessage(c.username, rawMessage["message_content"].(string))
		log.Printf("[readPump] Enviando mensaje al hub: %+v", message)
		// Enviar al hub para broadcast
		c.hub.broadcast <- message
	}
}

// writePump maneja el envío de mensajes al cliente
func (c *Client) writePump() {
	// Ticker para mantener la conexión viva
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			// Configurar timeout de escritura
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// El canal send fue cerrado
				log.Printf("[writePump] Canal send cerrado para %s", c.username)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			log.Printf("[writePump] Enviando mensaje a %s: %+v", c.username, message)
			// Enviar el mensaje como JSON
			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("[writePump] Error al enviar mensaje: %v", err)
				return
			}
		case <-ticker.C:
			// Enviar ping para mantener la conexión viva
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[writePump] Error enviando ping: %v", err)
				return
			}
		}
	}
}

// ServeWS maneja las conexiones WebSocket
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Obtener el username desde los parámetros de la URL
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "Anónimo"
	}
	
	// Upgrade de HTTP a WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}
	
    // Crear el cliente
    client := NewClient(hub, conn, username)

    // Registrar el cliente en el hub ANTES de iniciar las goroutines
    client.hub.register <- client
    // Iniciar las goroutines para leer y escribir
    go client.writePump()
    go client.readPump()

	

}