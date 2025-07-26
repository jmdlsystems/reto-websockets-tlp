package main

import (
	"encoding/base64"
	"testing"
)

// TestCreacionMensajeImagen verifica la creación de mensajes con imagen
func TestCreacionMensajeImagen(t *testing.T) {
	nombreUsuario := "usuario_prueba"
	contenido := "Mira esta imagen"
	datosImagen := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==" // 1x1 pixel PNG
	tipoImagen := "image/png"
	
	mensaje := envioImagen(nombreUsuario, contenido, datosImagen, tipoImagen)
	
	if mensaje.Username != nombreUsuario {
		t.Errorf("Se esperaba usuario %s, se obtuvo %s", nombreUsuario, mensaje.Username)
	}
	
	if mensaje.MessageContent != contenido {
		t.Errorf("Se esperaba contenido %s, se obtuvo %s", contenido, mensaje.MessageContent)
	}
	
	if mensaje.Type != "user" {
		t.Errorf("Se esperaba tipo 'user', se obtuvo %s", mensaje.Type)
	}
	
	if mensaje.ImagenData != datosImagen {
		t.Errorf("Se esperaban datos de imagen %s, se obtuvieron %s", datosImagen, mensaje.ImagenData)
	}
	
	if mensaje.ImagenType != tipoImagen {
		t.Errorf("Se esperaba tipo de imagen %s, se obtuvo %s", tipoImagen, mensaje.ImagenType)
	}
}

// TestValidacionTiposImagen verifica la validación de tipos de imagen
func TestValidacionTiposImagen(t *testing.T) {
	tiposValidos := []string{"image/jpeg", "image/jpg", "image/png"}
	tiposInvalidos := []string{"image/gif", "text/plain", "application/pdf", ""}
	
	for _, tipoImagen := range tiposValidos {
		if !validarTipoImagen(tipoImagen) {
			t.Errorf("Se esperaba que %s fuera válido", tipoImagen)
		}
	}
	
	for _, tipoImagen := range tiposInvalidos {
		if validarTipoImagen(tipoImagen) {
			t.Errorf("Se esperaba que %s fuera inválido", tipoImagen)
		}
	}
}

// TestObtenerExtensionImagen verifica la obtención de extensiones de archivo
func TestObtenerExtensionImagen(t *testing.T) {
	pruebas := []struct {
		entrada    string
		esperado   string
	}{
		{"image/jpeg", ".jpg"},
		{"image/jpg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ""},
		{"text/plain", ""},
		{"", ""},
	}
	
	for _, prueba := range pruebas {
		resultado := obtenerExtensionImagen(prueba.entrada)
		if resultado != prueba.esperado {
			t.Errorf("Para %s, se esperaba %s, se obtuvo %s", prueba.entrada, prueba.esperado, resultado)
		}
	}
}

// TestDatosImagenBase64 verifica el manejo de datos de imagen en base64
func TestDatosImagenBase64(t *testing.T) {
	// Simular datos de imagen en base64
	datosImagen := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
	
	// Verificar que es base64 válido
	_, err := base64.StdEncoding.DecodeString(datosImagen)
	if err != nil {
		t.Errorf("Datos base64 inválidos: %v", err)
	}
	
	// Crear mensaje con imagen
	mensaje := envioImagen("usuario_prueba", "Imagen de prueba", datosImagen, "image/png")
	
	if mensaje.ImagenData != datosImagen {
		t.Errorf("Los datos de imagen no se preservaron correctamente")
	}
}

// TestMensajeImagenSinContenido verifica mensajes de imagen sin texto
func TestMensajeImagenSinContenido(t *testing.T) {
	mensaje := envioImagen("usuario_prueba", "", "datosImagen", "image/png")
	
	if mensaje.MessageContent != "" {
		t.Errorf("Se esperaba contenido vacío, se obtuvo %s", mensaje.MessageContent)
	}
	
	if mensaje.Type != "user" {
		t.Errorf("Se esperaba tipo 'user', se obtuvo %s", mensaje.Type)
	}
}

// TestTiposImagenInsensiblesMayusculas verifica que la validación sea insensible a mayúsculas/minúsculas
func TestTiposImagenInsensiblesMayusculas(t *testing.T) {
	tipos := []string{"IMAGE/JPEG", "Image/Png", "image/jpg"}
	
	for _, tipoImagen := range tipos {
		if !validarTipoImagen(tipoImagen) {
			t.Errorf("Se esperaba que %s fuera válido (insensible a mayúsculas/minúsculas)", tipoImagen)
		}
	}
} 