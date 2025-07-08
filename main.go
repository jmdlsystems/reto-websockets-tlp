package main

import (
	"log"
	"net/http"
	
)

func main() {
	// Crear el hub de chat
	hub := NewHub()
	
	// Iniciar el hub en una goroutine separada
	go hub.Run()
	
	// Configurar las rutas
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	})
	
	// Servir archivos estáticos (HTML, CSS, JS)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	
	log.Println("Servidor de chat iniciado en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}