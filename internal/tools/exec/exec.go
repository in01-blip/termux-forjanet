// Package exec ejecuta comandos de shell, pero únicamente los que aparecen
// (por nombre exacto del binario) en la lista blanca de configuración.
// Sin lista blanca, este módulo no registra ninguna herramienta: el servidor
// arranca igualmente, simplemente sin capacidad de ejecutar shell.
// Nivel de seguridad: "peligroso".
package exec

import (
	"context"
	gexec "os/exec"
	"strings"
	"time"

	"termux-forjanet/internal/config"
	"termux-forjanet/internal/server"
)

func allowed(cmd string, list []string) bool {
	for _, c := range list {
		if c == cmd {
			return true
		}
	}
	return false
}

func Register(s *server.Server, cfg config.Config) {
	if !cfg.EnableDangerous || len(cfg.ShellAllowlist) == 0 {
		return // por diseño: sin opt-in explícito, esta herramienta no existe
	}

	s.Register(server.Tool{
		Name:        "run_command",
		Description: "Ejecuta un binario de la lista blanca configurada (TERMUXFORJANET_SHELL_ALLOWLIST).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"comando":    map[string]any{"type": "string"},
				"argumentos": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"comando"},
		},
		Handler: func(args map[string]any) (string, error) {
			comando, _ := args["comando"].(string)
			if !allowed(comando, cfg.ShellAllowlist) {
				return "", &notAllowedError{comando}
			}
			var argv []string
			if raw, ok := args["argumentos"].([]any); ok {
				for _, a := range raw {
					if str, ok := a.(string); ok {
						argv = append(argv, str)
					}
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			out, err := gexec.CommandContext(ctx, comando, argv...).CombinedOutput()
			result := strings.TrimSpace(string(out))
			if err != nil {
				return result, err
			}
			return result, nil
		},
	})
}

type notAllowedError struct{ cmd string }

func (e *notAllowedError) Error() string {
	return "comando no permitido por la lista blanca: " + e.cmd
}
