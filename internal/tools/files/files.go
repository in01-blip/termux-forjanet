// Package files da acceso de lectura/escritura a ficheros, mantenidos dentro
// de config.FileRoot: toda ruta se resuelve a su forma absoluta y canónica
// (symlinks incluidos) y se rechaza si escapa de la raíz. Nivel: "sensible".
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"termux-forjanet/internal/config"
	"termux-forjanet/internal/server"
)

func resolve(root, rel string) (string, error) {
	joined := filepath.Join(root, rel)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	// filepath.Join ya limpia ".." pero lo comprobamos explícitamente otra vez
	// por si el symlink resuelto apunta fuera de la raíz.
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		resolved, err = filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil {
			resolved = filepath.Dir(abs)
		}
		resolved = filepath.Join(resolved, filepath.Base(abs))
	}
	rootResolved, _ := filepath.EvalSymlinks(root)
	if rootResolved == "" {
		rootResolved = root
	}
	if resolved != rootResolved && !strings.HasPrefix(resolved, rootResolved+string(os.PathSeparator)) {
		return "", fmt.Errorf("ruta fuera del sandbox: %s", rel)
	}
	return abs, nil
}

func Register(s *server.Server, cfg config.Config) {
	root := cfg.FileRoot
	_ = os.MkdirAll(root, 0700)

	s.Register(server.Tool{
		Name:        "list_directory",
		Description: "Lista el contenido de un directorio dentro del sandbox de ficheros.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"ruta": map[string]any{"type": "string", "description": "ruta relativa a la raíz del sandbox"}},
		},
		Handler: func(args map[string]any) (string, error) {
			rel, _ := args["ruta"].(string)
			abs, err := resolve(root, rel)
			if err != nil {
				return "", err
			}
			entries, err := os.ReadDir(abs)
			if err != nil {
				return "", err
			}
			var sb strings.Builder
			for _, e := range entries {
				tipo := "fichero"
				if e.IsDir() {
					tipo = "directorio"
				}
				fmt.Fprintf(&sb, "%s\t%s\n", tipo, e.Name())
			}
			return sb.String(), nil
		},
	})

	s.Register(server.Tool{
		Name:        "read_file",
		Description: "Lee el contenido de un fichero de texto dentro del sandbox.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"ruta": map[string]any{"type": "string"}},
			"required":   []string{"ruta"},
		},
		Handler: func(args map[string]any) (string, error) {
			rel, _ := args["ruta"].(string)
			abs, err := resolve(root, rel)
			if err != nil {
				return "", err
			}
			data, err := os.ReadFile(abs)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	})

	s.Register(server.Tool{
		Name:        "write_file",
		Description: "Escribe (o crea) un fichero de texto dentro del sandbox.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ruta":      map[string]any{"type": "string"},
				"contenido": map[string]any{"type": "string"},
			},
			"required": []string{"ruta", "contenido"},
		},
		Handler: func(args map[string]any) (string, error) {
			rel, _ := args["ruta"].(string)
			contenido, _ := args["contenido"].(string)
			abs, err := resolve(root, rel)
			if err != nil {
				return "", err
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
				return "", err
			}
			if err := os.WriteFile(abs, []byte(contenido), 0600); err != nil {
				return "", err
			}
			return "escrito: " + rel, nil
		},
	})
}
