package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Configuración del upgrader WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Permitir conexiones desde cualquier origen
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

// goroutineLectura maneja la lectura de mensajes del cliente
func (c *Client) goroutineLectura() {
	defer func() {
		log.Printf("[goroutineLectura] Cliente %s desconectado, cerrando conexión", c.username)
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
				log.Printf("[goroutineLectura] error: %v", err)
			}
			break
		}
		log.Printf("[goroutineLectura] Mensaje recibido de %s: %s", c.username, string(messageBytes))
		// Parsear el mensaje JSON
		var rawMessage map[string]interface{}
		if err := json.Unmarshal(messageBytes, &rawMessage); err != nil {
			log.Printf("[goroutineLectura] Error al Parsear Mensaje: %v", err)
			continue
		}
		// Crear el mensaje con el username del cliente
		var message *Message
		
		// Verificar si es un mensaje con imagen
		if imagenData, hasImage := rawMessage["imagen_data"].(string); hasImage && imagenData != "" {
			imagenType := rawMessage["imagen_type"].(string)
			content := ""
			if contentVal, ok := rawMessage["message_content"].(string); ok {
				content = contentVal
			}
			
			// Validar tipo de imagen
			if !validarTipoImagen(imagenType) {
				log.Printf("[goroutineLectura] Tipo de imagen no soportado: %s", imagenType)
				continue
			}
			
			message = envioImagen(c.username, content, imagenData, imagenType)
			log.Printf("[goroutineLectura] Enviando mensaje con imagen al hub: %+v", message)
		} else {
			// Mensaje de texto normal
			message = NewUserMessage(c.username, rawMessage["message_content"].(string))
			log.Printf("[goroutineLectura] Enviando mensaje al hub: %+v", message)
		}
		
		// Enviar al hub para broadcast
		c.hub.broadcast <- message
	}
}

// goroutineEscritura maneja el envío de mensajes al cliente
func (c *Client) goroutineEscritura() {
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
				log.Printf("[goroutineEscritura] Canal send cerrado para %s", c.username)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			log.Printf("[goroutineEscritura] Enviando mensaje a %s: %+v", c.username, message)
			// Enviar el mensaje como JSON
			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("[goroutineEscritura] Error al enviar mensaje: %v", err)
				return
			}
		case <-ticker.C:
			// Enviar ping para mantener la conexión viva
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[goroutineEscritura] Error enviando ping: %v", err)
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
	go client.goroutineEscritura()
	go client.goroutineLectura()

}
