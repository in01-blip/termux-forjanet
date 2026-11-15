// Comando termux-forjanet: servidor MCP para Termux/Android, en español,
// con perfiles de rendimiento por arquitectura (ARM32/ARM64).
package main

import (
	"fmt"
	"os"
	osexec "os/exec"

	"termux-forjanet/internal/audit"
	"termux-forjanet/internal/config"
	"termux-forjanet/internal/server"
	"termux-forjanet/internal/tools/clipboard"
	"termux-forjanet/internal/tools/exec"
	"termux-forjanet/internal/tools/files"
	"termux-forjanet/internal/tools/system"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "doctor" {
		runDoctor()
		return
	}

	cfg := config.Load()
	s := server.New("termux-forjanet", "0.1.0") // nombre anunciado en el handshake MCP

	// Módulos "safe": siempre activos.
	system.Register(s, cfg)
	clipboard.Register(s)

	// Módulos "sensibles": solo si el usuario los activó explícitamente.
	if cfg.EnableSensitive {
		files.Register(s, cfg)
	}

	// Módulos "peligrosos": solo si el usuario los activó Y dio lista blanca.
	exec.Register(s, cfg)

	logger, err := audit.Open(os.Getenv("TERMUXFORJANET_AUDIT_LOG"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "aviso: no se pudo abrir el registro de auditoría:", err)
	}
	_ = logger // enganchado dentro de cada Handler en una futura iteración

	fmt.Fprintf(os.Stderr, "termux-forjanet arrancado — perfil=%s arquitectura=%s sensibles=%v peligrosos=%v\n",
		cfg.Profile, cfg.Arch, cfg.EnableSensitive, cfg.EnableDangerous)

	if err := s.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error en el servidor:", err)
		os.Exit(1)
	}
}

func runDoctor() {
	cfg := config.Load()
	fmt.Println("== termux-forjanet doctor ==")
	fmt.Println("arquitectura de compilación:", cfg.Arch)
	fmt.Println("perfil activo:             ", cfg.Profile)
	fmt.Println("raíz de ficheros:          ", cfg.FileRoot)
	fmt.Println("módulos sensibles:         ", cfg.EnableSensitive)
	fmt.Println("módulos peligrosos:        ", cfg.EnableDangerous)
	fmt.Println("lista blanca shell:        ", cfg.ShellAllowlist)
	for _, bin := range []string{"termux-battery-status", "termux-clipboard-get", "termux-wifi-connectioninfo"} {
		fmt.Printf("  %-28s", bin)
		if _, err := osexec.LookPath(bin); err == nil {
			fmt.Println("OK")
		} else {
			fmt.Println("no encontrado (instala termux-api)")
		}
	}
}
