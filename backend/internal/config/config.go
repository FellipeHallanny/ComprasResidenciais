package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	GeminiAPIKey      string
	SefazCertPath     string
	SefazCertPassword string
	JWTSecret         string
}

func LoadConfig() *Config {
	// Tenta carregar .env do diretório de trabalho ou subindo níveis
	locations := []string{".env", "../.env", "../../.env"}
	loaded := false
	found := false
	for _, loc := range locations {
		if abs, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(abs); err == nil {
				found = true
				if err := godotenv.Load(abs); err == nil {
					log.Printf("Arquivo de configuração .env carregado de: %s", abs)
					loaded = true
					break
				} else {
					log.Printf("Falha ao carregar arquivo de configuração .env em %s: %v", abs, err)
				}
			}
		}
	}

	if !loaded && !found {
		log.Println("Notice: Nenhum arquivo .env encontrado, utilizando variáveis de ambiente do sistema.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		log.Println("Aviso: DATABASE_URL não definida nas variáveis de ambiente.")
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		if os.Getenv("GIN_MODE") == "release" {
			log.Fatal("FATAL: Variável de ambiente JWT_SECRET é obrigatória em ambiente de produção.")
		}
		// Gera segredo aleatório efêmero de 256 bits para desenvolvimento seguro
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			log.Fatalf("Falha crítica ao gerar segredo seguro aleatório: %v", err)
		}
		jwtSecret = hex.EncodeToString(randomBytes)
		log.Println("Aviso de segurança: JWT_SECRET não configurado. Um segredo criptográfico aleatório foi gerado temporariamente para esta sessão.")
	} else if len(jwtSecret) < 32 {
		log.Println("Aviso de segurança: JWT_SECRET possui menos de 32 caracteres. Recomenda-se uma chave com pelo menos 32 caracteres (256 bits).")
	}

	return &Config{
		Port:              port,
		DatabaseURL:       dbURL,
		GeminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		SefazCertPath:     os.Getenv("SEFAZ_CERT_PATH"),
		SefazCertPassword: os.Getenv("SEFAZ_CERT_PASSWORD"),
		JWTSecret:         jwtSecret,
	}
}
