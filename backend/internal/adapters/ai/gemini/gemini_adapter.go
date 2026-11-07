package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"ComprasResidenciais/internal/core/ports"

	"github.com/google/generative-ai-go/genai"
	"github.com/pgvector/pgvector-go"
	"google.golang.org/api/option"
)

type GeminiAdapter struct {
	client     *genai.Client
	apiKey     string
	httpClient *http.Client
}

type geminiEmbedPart struct {
	Text string `json:"text"`
}

type geminiEmbedContent struct {
	Parts []geminiEmbedPart `json:"parts"`
}

type geminiEmbedRequest struct {
	Content              geminiEmbedContent `json:"content"`
	OutputDimensionality int                `json:"outputDimensionality"`
}

func normalizeReceiptMIME(mimeType string) (string, error) {
	mimeType = strings.ToLower(strings.TrimSpace(mimeType))
	if mimeType == "" {
		return "image/jpeg", nil
	}

	switch mimeType {
	case "application/pdf",
		"image/jpeg",
		"image/png",
		"image/webp",
		"image/heic",
		"image/heif":
		return mimeType, nil
	default:
		return "", fmt.Errorf("unsupported receipt MIME type: %s", mimeType)
	}
}

// NewGeminiAdapter inicializa o cliente Gemini usando a chave de API fornecida
// OTIMIZAÇÃO: Transporte HTTP customizado com MaxIdleConnsPerHost: 25 e keep-alive para
// eliminar TLS Handshake repetidos nas chamadas à API do Gemini.
func NewGeminiAdapter(ctx context.Context, apiKey string) (*GeminiAdapter, error) {
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is required for Gemini adapter")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	customTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   25,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &GeminiAdapter{
		client: client,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: customTransport,
		},
	}, nil
}

// GenerateEmbedding gera vetor de 768 dimensões com gemini-embedding-001
// OTIMIZAÇÃO: Payload com tipagem estática sem boxing em interface{} e streaming com
// json.NewDecoder para zero buffers intermediários na memória.
func (a *GeminiAdapter) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent"

	reqBody := geminiEmbedRequest{
		Content: geminiEmbedContent{
			Parts: []geminiEmbedPart{{Text: text}},
		},
		OutputDimensionality: 768,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return pgvector.Vector{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return pgvector.Vector{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", a.apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to call gemini embedding API: %w", err)
	}
	defer func() {
		// Drena o restante do body para permitir reuso de conexão TCP no pool Keep-Alive
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return pgvector.Vector{}, fmt.Errorf("gemini embedding API returned status %d: %s", resp.StatusCode, string(errBytes))
	}

	var res struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return pgvector.Vector{}, fmt.Errorf("failed to decode gemini embedding response: %w", err)
	}

	if len(res.Embedding.Values) == 0 {
		return pgvector.Vector{}, errors.New("empty embedding values returned from Gemini")
	}

	return pgvector.NewVector(res.Embedding.Values), nil
}

// ParseReceipt envia a imagem ou PDF do cupom fiscal para o Gemini extrair os dados estruturados
func (a *GeminiAdapter) ParseReceipt(ctx context.Context, fileBytes []byte, mimeType string) (*ports.ParsedReceipt, error) {
	model := a.client.GenerativeModel("gemini-3.8-flash")
	model.SetTemperature(0.1)
	model.ResponseMIMEType = "application/json"

	prompt := `
Você é um especialista em extração de dados fiscais de cupons e notas fiscais de supermercado brasileiras (NFC-e / Cupom SAT).
Analise a imagem/documento e extraia rigorosamente as informações no formato JSON com a seguinte estrutura:

{
  "store_name": "Nome Fantasia ou Razão Social do Mercado",
  "store_cnpj": "00.000.000/0000-00",
  "purchased_at": "2026-10-18T14:30:00Z",
  "total_amount": 150.40,
  "fiscal_key": "44 dígitos se visível, senão vazio",
  "items": [
    {
      "raw_description": "Descrição exata do item na nota",
      "quantity": 1.0,
      "package_size": 5.0,
      "package_unit": "kg",
      "total_price": 23.95
    }
  ]
}

Regras:
1. Para cada item, identifique o tamanho da embalagem e unidade:
- Se comprou "ARROZ TIO JOAO 5KG", package_size: 5.0, package_unit: "kg", quantity: 1.0.
- Se comprou "5 pct de ARROZ 1KG", package_size: 1.0, package_unit: "kg", quantity: 5.0.
- Unidades permitidas para package_unit: "kg", "g", "L", "ml", "un".
2. Se a data não estiver legível, retorne a data atual em ISO 8601.
`

	mimeType, err := normalizeReceiptMIME(mimeType)
	if err != nil {
		return nil, err
	}

	partData := genai.Blob{
		MIMEType: mimeType,
		Data:     fileBytes,
	}
	resp, err := model.GenerateContent(ctx, partData, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini vision error: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("no response generated from gemini")
	}

	// OTIMIZAÇÃO: strings.Builder com preallocation para evitar N alocações de concatenação de strings
	var sb strings.Builder
	sb.Grow(4096)
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			sb.WriteString(string(txt))
		}
	}

	cleaned := strings.TrimSpace(sb.String())
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var parsed ports.ParsedReceipt
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gemini response to JSON: %w (raw: %s)", err, cleaned)
	}

	if parsed.PurchasedAt.IsZero() {
		parsed.PurchasedAt = time.Now()
	}

	return &parsed, nil
}
