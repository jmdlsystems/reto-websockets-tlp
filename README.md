# Chat en Tiempo Real con WebSockets — Explicación y Trazabilidad de Requerimientos

## Cumplimiento del Reto #4: Aplicación de Chat en Tiempo Real con WebSockets

### 1. Servidor WebSockets en Go
- **Archivo principal:** `main.go`
- Se utiliza `net/http` para levantar el servidor HTTP y la ruta `/ws` es manejada por la función `ServeWS` (definida en `client.go`), que realiza el upgrade a WebSocket usando `github.com/gorilla/websocket`.
- **Justificación de la librería:** Se eligió `gorilla/websocket` por su robustez, manejo de errores y soporte de ping/pong automático (ver sección "Decisiones de Diseño").

### 2. Gestión de Clientes Conectados
- **Archivos:** `hub.go` y `client.go`
- El `Hub` mantiene un mapa `clients map[*Client]bool` protegido por `sync.RWMutex`.
- Cada cliente (`Client`) tiene dos goroutines: `goroutineLectura()` para leer mensajes entrantes y `goroutineEscritura()` para enviar mensajes salientes.
- El registro y desregistro de clientes se hace mediante canales (`register`, `unregister`).

### 3. Difusión de Mensajes (Broadcast)
- **Archivos:** `hub.go`, `message.go`
- Cuando un cliente envía un mensaje, este se recibe en su `goroutineLectura()` y se envía al canal `broadcast` del `Hub`.
- El `Hub` recibe el mensaje y lo reenvía a todos los clientes activos mediante su canal `send`.
- La struct de mensaje (`Message`) incluye `Username`, `MessageContent`, `Timestamp` y soporte para imágenes (`ImagenData`, `ImagenType`).
- Soporte para mensajes de texto, sistema e imágenes con validación de tipos MIME.

### 4. Manejo de Eventos de Conexión/Desconexión
- **Archivos:** `hub.go`, `client.go`
- Al conectar/desconectar un cliente, el `Hub` difunde un mensaje de sistema: "Usuario X se ha conectado/desconectado".
- El manejo de desconexión se realiza con `defer` y canales, asegurando limpieza de recursos.

### 5. Concurrencia Segura
- **Archivo:** `hub.go`
- El acceso al mapa de clientes está protegido por `sync.RWMutex`.
- Todas las operaciones de registro, desregistro y difusión usan canales y locks para evitar condiciones de carrera.
- La comunicación entre goroutines se realiza exclusivamente mediante canales (`register`, `unregister`, `broadcast`).

### 6. Front-end Básico (Opcional, pero implementado)
- **Archivo:** `index.html`
- HTML + JavaScript puro para conectarse al WebSocket, enviar y recibir mensajes.
- Interfaz responsiva y moderna, sin frameworks.
- Muestra mensajes de sistema, de usuario e imágenes con vista previa y ampliación.
- Valida nombres duplicados mostrando un error si corresponde.
- Soporte para subir imágenes con validación de tipo y tamaño (máximo 5MB).

### 7. Manejo de Errores
- **Archivos:** `client.go`, `hub.go`, `index.html`
- Manejo de errores de WebSocket (conexión cerrada, errores de red).
- Validación de mensajes malformados.
- El frontend muestra alertas si hay errores de conexión o si el nombre de usuario está en uso.

### 8. Restricciones y Cumplimiento
- No se usan frameworks de chat ni pub/sub externos.
- El chat es efímero y de una sola sala global.
- Toda la lógica de concurrencia y difusión está implementada con goroutines, canales y mutexes de Go.

### 9. Decisiones de Diseño y Reflexión
- **Modelo de conexión:** Una goroutine para leer (`goroutineLectura`) y otra para escribir (`goroutineEscritura`) por cliente. Comunicación con el hub mediante canales.
- **Difusión segura:** El hub usa un lock de lectura (`RLock`) para iterar sobre los clientes al difundir mensajes.
- **Manejo de desconexiones:** Uso de `defer` y canales para limpiar recursos y cerrar conexiones.
- **Canales para mensajes:** Canal `broadcast` sin buffer para mensajes globales. Canal `send` con buffer por cliente para evitar bloqueos.
- **Comunicación inter-goroutine:** Los mensajes de los clientes se envían al hub por el canal `broadcast`, y el hub los reenvía a todos los clientes.

### 10. Estructura de Archivos y Modularidad
- `main.go`: Arranque del servidor y rutas HTTP.
- `hub.go`: Lógica central del chat, registro y difusión.
- `client.go`: Abstracción y gestión de cada cliente WebSocket.
- `message.go`: Estructura de los mensajes y funcionalidad de imágenes.
- `index.html`: Cliente web para pruebas y uso real.
- `chat_test.go`: Pruebas unitarias y de concurrencia.
- `pruebas_imagen.go`: Pruebas específicas para funcionalidad de imágenes.

### 11. Pruebas y Robustez
- **Archivo:** `chat_test.go`
- Pruebas unitarias para registro/desregistro, difusión y concurrencia.
- **Archivo:** `pruebas_imagen.go`
- Pruebas específicas para funcionalidad de imágenes: validación de tipos, creación de mensajes, manejo de base64.
- Uso de `go test -race` para detectar condiciones de carrera.

### 12. Funcionalidad Extra y UX
- **Frontend responsivo y accesible.**
- **Validación de nombres duplicados tanto en backend como frontend.**
- **Mensajes de sistema claros y feedback visual inmediato.**
- **Soporte para imágenes con botón único de adjuntar y enviar.**
- **Validación de tipos de imagen (JPEG, PNG) y tamaño máximo (5MB).**

### 13. Funcionalidad de Imágenes (Nueva)
- **Backend (Go):**
  - Estructura `Message` extendida con campos `ImagenData` (base64) y `ImagenType` (MIME).
  - Función `envioImagen()` para crear mensajes con imágenes.
  - Validación de tipos MIME soportados (`validarTipoImagen()`).
  - Función `obtenerExtensionImagen()` para obtener extensiones de archivo.
- **Frontend (HTML/JavaScript):**
  - Botón único para adjuntar y enviar imágenes automáticamente.
  - Validación de tipo y tamaño de archivo (máximo 5MB).
  - Vista previa de imágenes en el chat con ampliación modal.
  - Soporte para JPEG, PNG y JPG.
- **Pruebas:**
  - Archivo `pruebas_imagen.go` con pruebas específicas para funcionalidad de imágenes.

---

## Tabla de Trazabilidad de Requerimientos

| Requerimiento | Archivo(s) | Función/Sección relevante |
|---------------|------------|--------------------------|
| Servidor WebSocket y ruta `/ws` | main.go, client.go | http.HandleFunc, ServeWS |
| Gestión de clientes | hub.go, client.go | Hub struct, registerClient, unregisterClient |
| Goroutines por cliente | client.go | goroutineLectura, goroutineEscritura |
| Registro seguro de clientes | hub.go | sync.RWMutex, canales |
| Difusión de mensajes | hub.go | broadcastMessage |
| Estructura de mensaje | message.go | Message struct |
| Mensajes de sistema | hub.go | NewSystemMessage, registerClient, unregisterClient |
| Manejo de desconexiones | client.go, hub.go | defer, unregisterClient |
| Frontend básico | index.html | Todo el archivo |
| Validación de duplicados | hub.go, index.html | registerClient, displayMessage |
| Funcionalidad de imágenes | message.go, client.go, index.html | envioImagen, validarTipoImagen, enviarImagen |
| Pruebas de imágenes | pruebas_imagen.go | Todo el archivo |
| Pruebas unitarias y de concurrencia | chat_test.go | Todo el archivo |
| Documentación y justificación | README.md | Secciones de arquitectura y decisiones |

---

Esta explicación y tabla te permiten defender y demostrar el cumplimiento de todos los puntos del reto, con referencias claras a tu código y decisiones de diseño.