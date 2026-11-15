// Package clipboard expone el portapapeles de Android a través de termux-api.
// Nivel de seguridad: "safe".
package clipboard

import (
	"fmt"
	"os/exec"
	"strings"

	"termux-forjanet/internal/server"
)

func Register(s *server.Server) {
	s.Register(server.Tool{
		Name:        "get_clipboard",
		Description: "Lee el contenido actual del portapapeles de Android.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(args map[string]any) (string, error) {
			out, err := exec.Command("termux-clipboard-get").CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("termux-clipboard-get falló: %w (%s)", err, out)
			}
			return strings.TrimSpace(string(out)), nil
		},
	})

	s.Register(server.Tool{
		Name:        "set_clipboard",
		Description: "Escribe texto en el portapapeles de Android.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"texto": map[string]any{"type": "string"}},
			"required":   []string{"texto"},
		},
		Handler: func(args map[string]any) (string, error) {
			texto, _ := args["texto"].(string)
			cmd := exec.Command("termux-clipboard-set")
			cmd.Stdin = strings.NewReader(texto)
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("termux-clipboard-set falló: %w (%s)", err, out)
			}
			return "portapapeles actualizado", nil
		},
	})
}
