package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// ── JSON-RPC 2.0 envelopes ────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── ACP method param/result types ────────────────────────────────────────────

type initializeParams struct {
	ProtocolVersion int `json:"protocolVersion"`
}

type initializeResult struct {
	ProtocolVersion    int             `json:"protocolVersion"`
	AgentCapabilities  agentCaps       `json:"agentCapabilities"`
	AgentInfo          agentInfo       `json:"agentInfo"`
	AuthMethods        []interface{}   `json:"authMethods"`
}

type agentCaps struct {
	LoadSession bool `json:"loadSession"`
}

type agentInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
}

type sessionNewParams struct {
	Cwd string `json:"cwd"`
}

type sessionNewResult struct {
	SessionID string `json:"sessionId"`
}

type sessionPromptParams struct {
	SessionID string         `json:"sessionId"`
	Messages  []promptMessage `json:"messages"`
}

type promptMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type sessionPromptResult struct {
	StopReason string `json:"stopReason"`
}

type sessionCancelParams struct {
	SessionID string `json:"sessionId"`
}

// ── Session/update notification helpers ──────────────────────────────────────

type sessionUpdateParams struct {
	SessionID string      `json:"sessionId"`
	Update    interface{} `json:"update"`
}

type agentMessageChunk struct {
	SessionUpdate string       `json:"sessionUpdate"`
	Content       contentBlock `json:"content"`
}

// ── Server state ─────────────────────────────────────────────────────────────

var (
	writeMu sync.Mutex
	writer  = bufio.NewWriter(os.Stdout)
)

// runACPServer reads newline-delimited JSON-RPC from stdin and dispatches requests.
func runACPServer() {
	scanner := bufio.NewScanner(os.Stdin)
	// Allow large prompts (up to 4 MB).
	buf := make([]byte, 4*1024*1024)
	scanner.Buffer(buf, cap(buf))

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			writeError(nil, -32700, "Parse error")
			continue
		}

		// Notifications have no ID — fire and forget, no response.
		if req.ID == nil {
			handleNotification(req)
			continue
		}

		handleRequest(req)
	}
}

// ── Dispatcher ───────────────────────────────────────────────────────────────

func handleRequest(req rpcRequest) {
	switch req.Method {
	case "initialize":
		handleInitialize(req)
	case "session/new":
		handleSessionNew(req)
	case "session/prompt":
		handleSessionPrompt(req)
	default:
		writeError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func handleNotification(req rpcRequest) {
	// session/cancel — kill any in-flight agy process (best-effort, no response).
	// Currently not tracked; agy --print is synchronous so cancel is a no-op.
	_ = req
}

// ── initialize ───────────────────────────────────────────────────────────────

func handleInitialize(req rpcRequest) {
	result := initializeResult{
		ProtocolVersion: 1,
		AgentCapabilities: agentCaps{
			LoadSession: false,
		},
		AgentInfo: agentInfo{
			Name:    "agy-acp-bridge",
			Title:   "agy ACP Bridge",
			Version: Version,
		},
		AuthMethods: []interface{}{},
	}
	writeResult(req.ID, result)
}

// ── session/new ──────────────────────────────────────────────────────────────

func handleSessionNew(req rpcRequest) {
	var params sessionNewParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(req.ID, -32602, "Invalid params for session/new")
		return
	}

	cwd := params.Cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			cwd = "/"
		}
	}

	sess := store.NewSession(cwd)
	writeResult(req.ID, sessionNewResult{SessionID: sess.ID})
}

// ── session/prompt ────────────────────────────────────────────────────────────

func handleSessionPrompt(req rpcRequest) {
	var params sessionPromptParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(req.ID, -32602, "Invalid params for session/prompt")
		return
	}

	sess := store.Get(params.SessionID)
	if sess == nil {
		writeError(req.ID, -32602, fmt.Sprintf("Unknown sessionId: %s", params.SessionID))
		return
	}

	// Extract text from the last user message.
	prompt := extractPromptText(params.Messages)
	if prompt == "" {
		writeError(req.ID, -32602, "No text content found in messages")
		return
	}

	// Run agy and collect the full response.
	result := RunPrompt(sess, prompt)
	if result.Err != nil {
		writeError(req.ID, -32603, fmt.Sprintf("agy error: %s", result.Err.Error()))
		return
	}

	store.MarkHistory(params.SessionID)

	// Emit agent_message_chunk notification with the full response.
	writeNotification("session/update", sessionUpdateParams{
		SessionID: params.SessionID,
		Update: agentMessageChunk{
			SessionUpdate: "agent_message_chunk",
			Content: contentBlock{
				Type: "text",
				Text: result.Output,
			},
		},
	})

	// Respond to the session/prompt request.
	writeResult(req.ID, sessionPromptResult{StopReason: "end_turn"})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func extractPromptText(messages []promptMessage) string {
	// Use the last message with role "user".
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != "user" {
			continue
		}
		for _, block := range msg.Content {
			if block.Type == "text" && block.Text != "" {
				return block.Text
			}
		}
	}
	return ""
}

func writeResult(id *json.RawMessage, result interface{}) {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	writeJSON(resp)
}

func writeError(id *json.RawMessage, code int, message string) {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
	writeJSON(resp)
}

func writeNotification(method string, params interface{}) {
	notif := rpcNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	writeJSON(notif)
}

func writeJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	writer.Write(data)
	writer.WriteByte('\n')
	writer.Flush()
}
