// Package system expone herramientas de solo lectura sobre el estado del
// dispositivo, apoyándose en los binarios termux-api cuando existen.
// Nivel de seguridad: "safe" — no requieren activar módulos sensibles.
package system

import (
	"os/exec"
	"strings"

	"termux-forjanet/internal/config"
	"termux-forjanet/internal/server"
)

// run ejecuta un comando termux-api si está instalado; si no, explica qué falta
// en vez de fallar de forma confusa.
func run(name string, args ...string) (string, error) {
	if _, err := exec.LookPath(name); err != nil {
		return "", &missingBinaryError{name}
	}
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type missingBinaryError struct{ bin string }

func (e *missingBinaryError) Error() string {
	return "falta '" + e.bin + "' — instala el paquete termux-api y la app Termux:API (F-Droid)"
}

// Register añade las herramientas del módulo "system" al servidor.
func Register(s *server.Server, cfg config.Config) {
	s.Register(server.Tool{
		Name:        "get_battery",
		Description: "Nivel de batería, estado de carga y temperatura (requiere termux-api).",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(args map[string]any) (string, error) {
			return run("termux-battery-status")
		},
	})

	s.Register(server.Tool{
		Name:        "device_info",
		Description: "Perfil de rendimiento activo y arquitectura para la que se compiló este binario.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(args map[string]any) (string, error) {
			return "perfil=" + cfg.Profile + " arquitectura_compilacion=" + cfg.Arch, nil
		},
	})

	s.Register(server.Tool{
		Name:        "get_wifi_info",
		Description: "Información de la conexión WiFi actual (requiere termux-api).",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(args map[string]any) (string, error) {
			return run("termux-wifi-connectioninfo")
		},
	})

	s.Register(server.Tool{
		Name:        "list_tools",
		Description: "Lista las herramientas activas en este servidor y su estado (opt-in o no).",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		Handler: func(args map[string]any) (string, error) {
			sensible := "desactivados"
			if cfg.EnableSensitive {
				sensible = "activados"
			}
			peligroso := "desactivados"
			if cfg.EnableDangerous {
				peligroso = "activados"
			}
			return "módulos sensibles: " + sensible + " | módulos peligrosos: " + peligroso, nil
		},
	})
}
