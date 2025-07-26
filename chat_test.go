package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"sync"
)

// TestHubCreation prueba la creación del hub
func TestHubCreation(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.clients == nil {
		t.Error("Hub clients map is nil")
	}

	if hub.broadcast == nil {
		t.Error("Hub broadcast channel is nil")
	}

	if hub.register == nil {
		t.Error("Hub register channel is nil")
	}

	if hub.unregister == nil {
		t.Error("Hub unregister channel is nil")
	}

	if hub.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", hub.GetClientCount())
	}
}

// TestMessageCreation prueba la creación de mensajes
func TestMessageCreation(t *testing.T) {
	username := "testuser"
	content := "test message"

	userMsg := NewUserMessage(username, content)
	if userMsg.Username != username {
		t.Errorf("Expected username %s, got %s", username, userMsg.Username)
	}

	if userMsg.MessageContent != content {
		t.Errorf("Expected content %s, got %s", content, userMsg.MessageContent)
	}

	if userMsg.Type != "user" {
		t.Errorf("Expected type 'user', got %s", userMsg.Type)
	}

	systemMsg := NewSystemMessage(content)
	if systemMsg.Username != "Sistema" {
		t.Errorf("Se esperaba el nombre de usuario 'Sistema', obtuve %s", systemMsg.Username)
	}

	if systemMsg.Type != "system" {
		t.Errorf("Se esperaba un tipo 'sistema', obtuve%s", systemMsg.Type)
	}
}

// TestClientRegistration prueba el registro de clientes
func TestClientRegistration(t *testing.T) {
	hub := NewHub()

	// Iniciar el hub en una goroutine
	go hub.Run()

	// Crear un servidor de test
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	}))
	defer server.Close()

	// Convertir la URL HTTP a WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?username=testuser"

	// Conectar al WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Error de conexión al WebSocket: %v", err)
	}
	defer conn.Close()

	// Esperar un poco para que el cliente se registre
	time.Sleep(100 * time.Millisecond)

	// Verificar que el cliente se registró
	if hub.GetClientCount() != 1 {
		t.Errorf("Se esperaban 1 cliente, obtuvimos %d", hub.GetClientCount())
	}
}

// TestMessageBroadcast prueba la difusión de mensajes
func TestMessageBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Conectar múltiples clientes
	var conns []*websocket.Conn
	numClients := 3

	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL+fmt.Sprintf("?username=user%d", i), nil)
		if err != nil {
			t.Fatalf("Error de conexión del cliente %d: %v", i, err)
		}
		conns = append(conns, conn)
	}

	// Cerrar todas las conexiones al final
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	// Esperar a que todos los clientes se conecten
	time.Sleep(200 * time.Millisecond)

	// Verificar que todos los clientes están conectados
	if hub.GetClientCount() != numClients {
		t.Errorf("Se esperaban %d clientes, obtuvimos %d", numClients, hub.GetClientCount())
	}

	// Enviar mensaje desde el primer cliente
	testMessage := map[string]interface{}{
		"message_content": "Hola desde el cliente 0",
	}

	messageBytes, _ := json.Marshal(testMessage)
	err := conns[0].WriteMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		t.Fatalf("Error al enviar el mensaje: %v", err)
	}

	// Verificar que todos los clientes reciben el mensaje
	for i, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, receivedBytes, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Error al leer el mensaje del cliente %d: %v", i, err)
		}

		var receivedMessage Message
		err = json.Unmarshal(receivedBytes, &receivedMessage)
		if err != nil {
			t.Fatalf("Error al deserializar el mensaje: %v", err)
		}

		if receivedMessage.Type == "user" && receivedMessage.MessageContent == "Hola desde el cliente 0" {
			// Este es el mensaje que esperamos
			continue
		}

		if receivedMessage.Type != "system" {
			t.Errorf("Mensaje de sistema o mensaje de usuario esperado, se obtuvo tipo %s", receivedMessage.Type)
		}
	}
}

// TestConcurrentClients prueba la seguridad concurrente con múltiples clientes
func TestConcurrentClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Número de clientes concurrentes
	numClients := 10
	numMessages := 5

	var wg sync.WaitGroup
	var mu sync.Mutex
	receivedMessages := make(map[string]int)

	// Función para simular un cliente
	clientFunc := func(clientID int) {
		defer wg.Done()

		conn, _, err := websocket.DefaultDialer.Dial(wsURL+fmt.Sprintf("?username=user%d", clientID), nil)
		if err != nil {
			t.Errorf("Error de conexión del cliente %d: %v", clientID, err)
			return
		}
		defer conn.Close()

		// Goroutine para leer mensajes
		go func() {
			for {
				_, messageBytes, err := conn.ReadMessage()
				if err != nil {
					break
				}

				var message Message
				if err := json.Unmarshal(messageBytes, &message); err != nil {
					continue
				}

				mu.Lock()
				receivedMessages[message.MessageContent]++
				mu.Unlock()
			}
		}()

		// Enviar múltiples mensajes
		for i := 0; i < numMessages; i++ {
			testMessage := map[string]interface{}{
				"message_content": fmt.Sprintf("Mensaje %d del cliente %d", i, clientID),
			}

			messageBytes, _ := json.Marshal(testMessage)
			err := conn.WriteMessage(websocket.TextMessage, messageBytes)
			if err != nil {
				t.Errorf("Error al enviar el mensaje desde el cliente %d: %v", clientID, err)
				return
			}

			// Pequeña pausa entre mensajes
			time.Sleep(10 * time.Millisecond)
		}

		// Mantener la conexión abierta un poco más
		time.Sleep(500 * time.Millisecond)
	}

	// Lanzar todos los clientes concurrentemente
	wg.Add(numClients)
	for i := 0; i < numClients; i++ {
		go clientFunc(i)
	}

	wg.Wait()

	// Esperar un poco más para que todos los mensajes se procesen
	time.Sleep(1 * time.Second)

	// Verificar que no hay condiciones de carrera
	// (el test -race detectará automáticamente las condiciones de carrera)
}

// TestClientDisconnection prueba el manejo de desconexiones
func TestClientDisconnection(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?username=testuser"

	// Conectar un cliente
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Error de conexión: %v", err)
	}

	// Esperar a que se registre
	time.Sleep(100 * time.Millisecond)

	// Verificar que el cliente está conectado
	if hub.GetClientCount() != 1 {
		t.Errorf("Se esperaban 1 cliente, obtuvimos %d", hub.GetClientCount())
	}

	// Cerrar la conexión
	conn.Close()

	// Esperar a que se desregistre
	time.Sleep(100 * time.Millisecond)

	// Verificar que el cliente se desconectó
	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes, obtuvimos %d", hub.GetClientCount())
	}
}

// TestRaceConditions prueba específicamente las condiciones de carrera
func TestRaceConditions(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Simular registro y desregistro concurrente de clientes
	var wg sync.WaitGroup
	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Simular un cliente ficticio
			client := &Client{
				hub:      hub,
				conn:     nil, // No necesitamos una conexión real para esta prueba
				send:     make(chan *Message, 256),
				username: fmt.Sprintf("Usuario%d", id),
			}

			// Registrar el cliente
			hub.register <- client

			// Esperar un poco
			time.Sleep(10 * time.Millisecond)

			// Enviar algunos mensajes
			for j := 0; j < 5; j++ {
				message := NewUserMessage(client.username, fmt.Sprintf("Mensaje %d", j))
				hub.broadcast <- message
				time.Sleep(1 * time.Millisecond)
			}

			// Desregistrar el cliente
			hub.unregister <- client
		}(i)
	}

	wg.Wait()

	// Esperar a que se procesen todos los eventos
	wg.Wait()
	// Esperar más tiempo para que el hub procese los desregistros
	time.Sleep(500 * time.Millisecond)
	if hub.GetClientCount() != 0 {
		t.Errorf("Se esperaban 0 clientes, obtuvimos %d", hub.GetClientCount())
	}
}

// BenchmarkMessageBroadcast benchmarks para medir el rendimiento
func BenchmarkMessageBroadcast(b *testing.B) {
	hub := NewHub()
	go hub.Run()
	// Simular algunos clientes
	numClients := 100
	clients := make([]*Client, numClients)

	for i := 0; i < numClients; i++ {
		client := &Client{
			hub:      hub,
			conn:     nil,
			send:     make(chan *Message, 256),
			username: fmt.Sprintf("Usuario%d", i),
		}
		clients[i] = client
		hub.register <- client
	}

	// Esperar a que se registren todos los clientes
	time.Sleep(100 * time.Millisecond)

	// Consumir mensajes de los canales send para evitar bloqueos
	for _, client := range clients {
		go func(c *Client) {
			for range c.send {
				// Consumir mensajes
			}
		}(client)
	}

	b.ResetTimer()

	// Benchmark de difusión de mensajes
	for i := 0; i < b.N; i++ {
		message := NewUserMessage("benchuser", fmt.Sprintf("Benchmark message %d", i))
		hub.broadcast <- message
	}

	// Limpiar
	for _, client := range clients {
		hub.unregister <- client
	}
}
