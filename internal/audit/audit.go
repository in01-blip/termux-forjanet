// Package audit escribe un registro JSONL de cada llamada a una herramienta,
// redactando los argumentos que puedan contener datos sensibles (tokens,
// contraseñas, cuerpos de SMS...). Mismo principio que termux-mcp/DroidMCP.
package audit

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

var redactKeys = []string{"token", "password", "contrasena", "body", "mensaje", "sms"}

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func Open(path string) (*Logger, error) {
	if path == "" {
		return &Logger{}, nil // auditoría desactivada
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

func redact(args map[string]any) map[string]any {
	clean := map[string]any{}
	for k, v := range args {
		lower := strings.ToLower(k)
		hit := false
		for _, r := range redactKeys {
			if strings.Contains(lower, r) {
				hit = true
				break
			}
		}
		if hit {
			clean[k] = "[REDACTADO]"
		} else {
			clean[k] = v
		}
	}
	return clean
}

// Log registra una llamada. Nunca bloquea la ejecución de la herramienta si falla.
func (l *Logger) Log(tool string, args map[string]any, ok bool) {
	if l == nil || l.file == nil {
		return
	}
	entry := map[string]any{
		"ts":   time.Now().UTC().Format(time.RFC3339),
		"tool": tool,
		"args": redact(args),
		"ok":   ok,
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	enc := json.NewEncoder(l.file)
	_ = enc.Encode(entry)
}
