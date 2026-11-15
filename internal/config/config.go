// Package config centraliza toda la configuración del servidor.
// Todo se controla por variables de entorno con prefijo TERMUXFORJANET_,
// siguiendo el modelo "default-deny" de proyectos como DroidMCP y termux-mcp:
// nada sensible se activa salvo que el usuario lo pida explícitamente.
package config

import (
	"os"
	"strconv"
	"strings"
)

// Config contiene todo lo que el servidor necesita para arrancar.
type Config struct {
	// Perfil de rendimiento detectado o forzado: "alto", "medio", "bajo", "patata".
	// Lo rellena el script de instalación según la arquitectura/RAM del dispositivo.
	Profile string

	// Arquitectura real del binario en ejecución (informativa, viene del linker).
	Arch string

	// Raíz del sistema de archivos permitida para el módulo de ficheros.
	// Nunca debe quedar en "/" — eso anularía el sandbox.
	FileRoot string

	// Lista blanca de comandos permitidos para el módulo shell. Vacía = shell desactivado.
	ShellAllowlist []string

	// Activa los módulos "sensibles" (SMS, contactos, cámara...). Apagado por defecto.
	EnableSensitive bool

	// Activa los módulos "peligrosos" (automatización de UI, control de tareas). Apagado por defecto.
	EnableDangerous bool

	// Token de autenticación opcional para el transporte HTTP/LAN.
	AuthToken string
}

func boolEnv(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// Load construye la configuración a partir del entorno.
func Load() Config {
	root := os.Getenv("TERMUXFORJANET_ROOT")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = home + "/storage/downloads/termux-forjanet-workspace"
	}

	allow := os.Getenv("TERMUXFORJANET_SHELL_ALLOWLIST")
	var allowlist []string
	if allow != "" {
		for _, c := range strings.Split(allow, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				allowlist = append(allowlist, c)
			}
		}
	}

	profile := os.Getenv("TERMUXFORJANET_PROFILE")
	if profile == "" {
		profile = "medio"
	}

	return Config{
		Profile:         profile,
		Arch:            BuildArch,
		FileRoot:        root,
		ShellAllowlist:  allowlist,
		EnableSensitive: boolEnv("TERMUXFORJANET_ENABLE_SENSITIVE", false),
		EnableDangerous: boolEnv("TERMUXFORJANET_ENABLE_DANGEROUS", false),
		AuthToken:       os.Getenv("TERMUXFORJANET_TOKEN"),
	}
}

// BuildArch se fija en tiempo de compilación con -ldflags "-X ...BuildArch=arm64",
// de modo que el binario sabe para qué perfil fue construido sin tener que adivinarlo
// de nuevo en tiempo de ejecución.
var BuildArch = "desconocido"
