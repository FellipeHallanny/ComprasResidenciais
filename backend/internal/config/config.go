package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DatabaseURL       string
	GeminiAPIKey      string
	SefazCertPath     string
	SefazCertPassword string
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

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/compras_residenciais?sslmode=disable"
	}

	return &Config{
		Port:              port,
		DatabaseURL:       dbURL,
		GeminiAPIKey:      os.Getenv("GEMINI_API_KEY"),
		SefazCertPath:     os.Getenv("SEFAZ_CERT_PATH"),
		SefazCertPassword: os.Getenv("SEFAZ_CERT_PASSWORD"),
	}
}
