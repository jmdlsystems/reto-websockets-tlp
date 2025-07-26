package main

import (
	"strings"
	"time"
)

type Message struct {
	Username       string    `json:"username"`
	MessageContent string    `json:"message_content"`
	Timestamp      time.Time `json:"timestamp"`
	Type           string    `json:"type"` // "user" o "system"
	ImagenData     string    `json:"imagen_data,omitempty"` 
	ImagenType     string    `json:"imagen_type,omitempty"`
}

func NewUserMessage(username, content string) *Message {
	return &Message{
		Username:       username,
		MessageContent: content,
		Timestamp:      time.Now(),
		Type:           "user",
	}
}

func NewSystemMessage(content string) *Message {
	return &Message{
		Username:       "Sistema",
		MessageContent: content,
		Timestamp:      time.Now(),
		Type:           "system",
	}
}

func envioImagen(username, content, imagenData, imagenType string) *Message {
	return &Message{
		Username:       username,
		MessageContent: content,
		Timestamp:      time.Now(),
		Type:           "user",
		ImagenData:     imagenData,
		ImagenType:     imagenType,
	}
}

// validarTipoImagen verifica si el tipo de imagen es soportado
func validarTipoImagen(tipoImagen string) bool {
	tiposSoportados := []string{"image/jpeg", "image/jpg", "image/png"}
	for _, tipo := range tiposSoportados {
		if strings.ToLower(tipoImagen) == tipo {
			return true
		}
	}
	return false
}

// obtenerExtensionImagen obtiene la extensión del archivo basado en el tipo MIME
func obtenerExtensionImagen(tipoImagen string) string {
	switch strings.ToLower(tipoImagen) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}