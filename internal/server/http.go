// Transporte HTTP, alternativo al de stdio en mcp.go. Expone un único
// endpoint POST que recibe un mensaje JSON-RPC por petición y devuelve la
// respuesta como JSON — el subconjunto mínimo del transporte "Streamable
// HTTP" de MCP que necesitan nuestros métodos (initialize, tools/list,
// tools/call), ya que este servidor nunca envía mensajes por iniciativa
// propia y no necesita el canal de eventos (SSE) de la variante completa.
package server

import (
	"io"
	"log"
	"net/http"
	"strings"
)

// ServeHTTP arranca un servidor HTTP en addr (p. ej. ":8765") y bloquea.
// Si token no está vacío, exige la cabecera "Authorization: Bearer <token>"
// en cada petición — algo prácticamente obligatorio en cuanto el puerto se
// expone a internet a través de un túnel, porque sin él cualquiera con la
// URL puede invocar tus herramientas (leer el portapapeles, batería, etc.,
// y más si activaste los módulos sensibles/peligrosos).
func (s *Server) ServeHTTP(addr, token string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != token {
				http.Error(w, `{"error":"no autorizado"}`, http.StatusUnauthorized)
				return
			}
		}
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"usa POST"}`, http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB de margen, de sobra para estos mensajes
		if err != nil {
			http.Error(w, `{"error":"no se pudo leer el cuerpo"}`, http.StatusBadRequest)
			return
		}
		respBytes, hasResp := s.HandleMessage(body)
		w.Header().Set("Content-Type", "application/json")
		if !hasResp {
			w.WriteHeader(http.StatusAccepted) // notificación: aceptada, sin cuerpo que devolver
			return
		}
		_, _ = w.Write(respBytes)
	})

	if token == "" {
		log.Println("AVISO: TERMUXFORJANET_TOKEN no está definido — el endpoint /mcp queda SIN autenticación. No lo expongas a internet así.")
	}
	log.Printf("termux-forjanet escuchando HTTP en %s (endpoint: /mcp)\n", addr)
	return http.ListenAndServe(addr, mux)
}
