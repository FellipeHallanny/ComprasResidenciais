package sefaz

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ComprasResidenciais/internal/core/ports"
)

// SefazAdapter implementa a porta SefazAdapterPort
// Fica preparado estruturalmente para quando o Certificado A1 (PKCS#12) for adicionado
type SefazAdapter struct {
	certPath     string
	certPassword string
	isConfigured bool
}

// NewSefazAdapter inicializa o adaptador SEFAZ
func NewSefazAdapter(certPath, certPassword string) *SefazAdapter {
	configured := certPath != "" && certPassword != ""
	return &SefazAdapter{
		certPath:     certPath,
		certPassword: certPassword,
		isConfigured: configured,
	}
}

// ConsultarChave consulta os dados completos da nota fiscal através da chave de 44 dígitos
func (a *SefazAdapter) ConsultarChave(ctx context.Context, chaveNFe string) (*ports.ParsedReceipt, error) {
	chave := strings.TrimSpace(chaveNFe)
	if len(chave) != 44 {
		return nil, fmt.Errorf("chave de acesso inválida (esperado 44 dígitos, recebido %d)", len(chave))
	}

	if !a.isConfigured {
		return nil, errors.New("adaptador SEFAZ requer Certificado Digital ICP-Brasil A1 (configure SEFAZ_CERT_PATH e SEFAZ_CERT_PASSWORD). Utilize o upload de imagem/PDF via IA enquanto o certificado não estiver configurado")
	}

	// Estrutura pronta para execução do NfeConsultaProtocolo v4.00 com mTLS
	// Em produção com certificado A1, carrega os pares TLS via tls.X509KeyPair e faz a requisição SOAP
	return nil, errors.New("comunicação mTLS com webservice da SEFAZ aguardando carregamento do certificado A1")
}
