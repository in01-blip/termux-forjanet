// Package server implementa lo justo del protocolo MCP (Model Context Protocol)
// sobre stdio sin tirar de ninguna librería externa: así el binario sigue siendo
// un único ejecutable estático, fácil de compilar de forma cruzada para ARM32/ARM64
// y sin pasos de "go mod download" que dependan de red en el propio dispositivo.
package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
)

// Tool describe una herramienta MCP y cómo ejecutarla.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(args map[string]any) (string, error)
}

// Server es un servidor MCP mínimo sobre stdio.
type Server struct {
	Name    string
	Version string
	tools   map[string]Tool
	order   []string
}

func New(name, version string) *Server {
	return &Server{Name: name, Version: version, tools: map[string]Tool{}}
}

// Register añade una herramienta al catálogo. El orden de registro se conserva
// para que `tools/list` sea determinista.
func (s *Server) Register(t Tool) {
	if _, exists := s.tools[t.Name]; !exists {
		s.order = append(s.order, t.Name)
	}
	s.tools[t.Name] = t
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Serve bloquea leyendo peticiones JSON-RPC línea a línea de in y escribiendo
// las respuestas en out, tal y como espera un cliente MCP sobre stdio.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	enc := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue // notificación/basura: se ignora, no se rompe la sesión
		}
		resp := s.dispatch(req)
		if resp == nil {
			continue // era una notificación (sin id); no se responde
		}
		if err := enc.Encode(resp); err != nil {
			log.Printf("error escribiendo respuesta: %v", err)
		}
	}
	return scanner.Err()
}

func (s *Server) dispatch(req rpcRequest) *rpcResponse {
	switch req.Method {
	case "initialize":
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]any{"name": s.Name, "version": s.Version},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		}}
	case "notifications/initialized":
		return nil
	case "tools/list":
		list := make([]map[string]any, 0, len(s.order))
		for _, name := range s.order {
			t := s.tools[name]
			list = append(list, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": t.InputSchema,
			})
		}
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": list}}
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "método no soportado: " + req.Method}}
	}
}

func (s *Server) handleToolsCall(req rpcRequest) *rpcResponse {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "parámetros inválidos"}}
	}
	t, ok := s.tools[params.Name]
	if !ok {
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: fmt.Sprintf("herramienta desconocida: %s", params.Name)}}
	}
	text, err := t.Handler(params.Arguments)
	if err != nil {
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"isError": true,
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
		}}
	}
	return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}}
}
