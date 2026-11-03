<div align="center">
  <img src="frontend/public/favicon.svg" width="80" height="80" alt="Compras Residenciais Logo" />

  # Compras Residenciais

  **Plataforma inteligente de controle de compras domésticas, normalização de preços e análise de mercado.**

  [![Versão Go](https://img.shields.io/badge/Go-1.27-00ADD8?style=flat-square&logo=go)](https://golang.org)
  [![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?style=flat-square&logo=svelte)](https://svelte.dev)
  [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16%2B_pgvector-4169E1?style=flat-square&logo=postgresql)](https://www.postgresql.org/)
  [![Model Context Protocol](https://img.shields.io/badge/MCP-Habilitado-9B51E0?style=flat-square)](https://modelcontextprotocol.io/)
  [![Tailwind CSS](https://img.shields.io/badge/TailwindCSS-v4-06B6D4?style=flat-square&logo=tailwindcss)](https://tailwindcss.com)

  [English](README.md) • [Português (Brasil)](README.pt-BR.md)

  <p>
    <a href="#visão-geral">Visão Geral</a> •
    <a href="#principais-funcionalidades">Principais Funcionalidades</a> •
    <a href="#arquitetura">Arquitetura</a> •
    <a href="#stack-tecnológica">Stack Tecnológica</a> •
    <a href="#primeiros-passos">Primeiros Passos</a> •
    <a href="#integração-com-agentes-mcp">Agentes e MCP</a> •
    <a href="#referência-da-api">Referência da API</a> •
    <a href="#configuração">Configuração</a>
  </p>
</div>

---

## Visão Geral

Nas compras domésticas do dia a dia, diferentes moradores adquirem itens essenciais em diversos estabelecimentos: supermercados, atacados, feiras livres e farmácias. Aplicativos financeiros comuns exigem a digitação manual item a item e focam apenas no montante gasto, sem inteligência sobre produtos ou preços.

O **Compras Residenciais** elimina esse atrito automatizando a leitura de notas fiscais e aplicando normalização de preços (padronizando diferentes tamanhos de embalagens em R$/kg, R$/L ou R$/unidade). Dessa forma, a residência consegue responder de imediato:
1. Em qual estabelecimento determinado produto tem o menor preço histórico ou recente.
2. Como os preços oscilam ao longo das épocas do mês (início, meio ou fim).
3. Se embalagens maiores (ex: pacote de 5 kg) realmente compensam em relação a múltiplos pacotes menores (ex: 5 pacotes de 1 kg), independente da marca.
4. Como os gastos totais estão distribuídos entre os membros da casa.

O sistema conta com um painel web moderno em **Svelte 5** e um servidor nativo **Model Context Protocol (MCP)**, permitindo que assistentes de IA (Claude Desktop, Cursor) consultem dados e registrem compras autonomamente via linguagem natural.

---

## Principais Funcionalidades

- **Captura Automática de Cupons (IA Multimodal)**: Envio de foto ou PDF da nota fiscal com extração via Google Gemini (identifica estabelecimento, data, itens brutos, quantidades, embalagens e valores) para conferência em rascunho.
- **Normalização de Preços por Unidade**: Converte automaticamente embalagens e formatos variados em unidades-base (R$/kg, R$/L, R$/unidade), garantindo comparações justas entre marcas e tamanhos de pacote.
- **Agrupamento Semântico (pgvector)**: Associa descrições brutas de cupom (ex: `"ARROZ TP1 5KG"`) a produtos canônicos (`"Arroz Branco"`) utilizando embeddings de 768 dimensões e busca aproximada por cosseno (HNSW).
- **Análise Temporal de Oscilação de Preços**: Monitora médias de preços divididas em períodos do mês (dias 1–7, 8–14, 15–21, 22–31) para identificar os melhores momentos para compras.
- **Detecção de Oportunidades**: Exibe o estabelecimento vencedor em economia e a porcentagem de variação entre o menor e o maior preço registrado.
- **Gestão de Moradores e Gastos**: Vincula quem realizou a compra e preserva o histórico de gastos com bloqueio suave (*soft-block*) ao invés de exclusão definitiva de registros vinculados.
- **Interface Dupla (Web e Agentes MCP)**: Aplicação web reativa em Svelte 5 e servidor MCP via STDIO para integração direta com agentes autônomos de IA.

---

## Arquitetura

O backend adota estritamente a **Arquitetura Hexagonal** (Ports and Adapters), isolando as regras de negócio de detalhes de infraestrutura, bancos de dados e protocolos externos.

```mermaid
flowchart TD
    subgraph DrivingAdapters["Adaptadores Primários (Entrada)"]
        HTTP["Handler HTTP Gin\n(/api/v1/...)"]
        MCP["Servidor MCP (STDIO)\n(mark3labs/mcp-go)"]
    end

    subgraph CoreDomain["Núcleo de Domínio & Casos de Uso"]
        PUC["PurchaseService"]
        AUC["AnalyticsService"]
        UUC["UserService"]
    end

    subgraph DrivenAdapters["Adaptadores Secundários (Saída)"]
        PG["PostgreSQL + pgvector\n(pgx/v5 com Índice HNSW)"]
        GEM["Google Gemini\n(Multimodal & text-embedding-004)"]
        SFZ["Adaptador Fiscal SEFAZ\n(Stub mTLS)"]
    end

    HTTP --> PUC & AUC & UUC
    MCP --> PUC & AUC & UUC
    PUC & AUC & UUC --> PG
    PUC --> GEM
    PUC --> SFZ
```

### Estrutura de Diretórios

```
ComprasResidenciais/
├── backend/
│   ├── cmd/
│   │   ├── server/               # Ponto de entrada do servidor HTTP Gin
│   │   └── mcp/                  # Ponto de entrada do servidor MCP STDIO
│   ├── internal/
│   │   ├── core/
│   │   │   ├── domain/           # Entidades, views e objetos de valor
│   │   │   ├── ports/            # Contratos de interfaces primárias e secundárias
│   │   │   └── services/         # Casos de uso e cálculos de normalização
│   │   ├── adapters/
│   │   │   ├── handlers/         # Controladores HTTP (Gin) e MCP
│   │   │   ├── repositories/     # Repositórios PostgreSQL e buscas vetoriais
│   │   │   ├── ai/               # Integração Gemini multimodal e embeddings
│   │   │   └── sefaz/            # Comunicação com serviços fiscais
│   │   └── config/               # Carregamento de variáveis de ambiente
│   ├── migrations/               # Migrações SQL e índices HNSW pgvector
│   └── docker-compose.yml        # Definição do container PostgreSQL com pgvector
├── frontend/
│   ├── src/
│   │   ├── components/           # Componentes Svelte 5 com Runes
│   │   ├── lib/                  # Cliente HTTP e integração com a API
│   │   └── types.ts              # Tipagens de domínio em TypeScript
│   ├── public/                   # Favicon e ícones estáticos
│   └── vite.config.ts            # Configuração do Vite e proxy reverso
└── docs/                         # Especificações de design e arquitetura
```

---

## Stack Tecnológica

| Camada | Tecnologias Utilizadas |
| :--- | :--- |
| **Backend** | Go 1.27, Arquitetura Hexagonal, `gin-gonic/gin` |
| **Protocolo de IA** | `mark3labs/mcp-go` (transporte STDIO) |
| **Banco de Dados** | PostgreSQL 16+, extensão `pgvector`, driver `jackc/pgx/v5` |
| **Inteligência Artificial** | Google Gemini (`gemini-1.5-flash` / `gemini-2.0`, `text-embedding-004`) |
| **Frontend** | Svelte 5 (Runes: `$state`, `$derived`, `$props`, `$effect`), TypeScript |
| **Estilização & Ícones** | Tailwind CSS v4, Lucide Svelte (`@lucide/svelte`) |
| **Ferramentas de Build** | Vite 8, Docker Compose |

---

## Primeiros Passos

### Pré-requisitos

Certifique-se de ter instalado em sua máquina:
- [Go 1.27+](https://golang.org/dl/)
- [Node.js 20+](https://nodejs.org/) e npm
- [Docker e Docker Compose](https://www.docker.com/)
- (Opcional) [Chave de API do Google Gemini](https://aistudio.google.com/) para extração inteligente de notas e busca vetorial

### 1. Iniciar o Banco de Dados

Suba o PostgreSQL 16 com suporte nativo a `pgvector`:

```bash
docker compose up -d
```

> [!NOTE]
> As migrações contidas em `backend/migrations/` (tabelas e índice vetorial HNSW) são aplicadas de forma automática na inicialização da aplicação.

### 2. Configurar Variáveis de Ambiente

Crie o arquivo de configuração no diretório `backend`:

```bash
cp backend/.env.example backend/.env
```

Edite o arquivo `backend/.env` com as configurações do seu ambiente:

```env
PORT=8080
DATABASE_URL=postgres://postgres:postgrespassword@localhost:5432/compras_residenciais?sslmode=disable
GEMINI_API_KEY=sua_chave_gemini_aqui
```

> [!TIP]
> Caso a `GEMINI_API_KEY` não seja informada, os cadastros manuais e painéis analíticos continuam funcionando normalmente; a extração por imagem e busca vetorial utilizarão correspondência textual simples.

### 3. Executar o Servidor Backend

```bash
cd backend
go run cmd/server/main.go
```

A API REST estará acessível em `http://localhost:8080`.

### 4. Executar o Frontend

Em outro terminal:

```bash
cd frontend
npm install
npm run dev
```

Acesse `http://localhost:5173` no navegador. O Vite redirecionará automaticamente chamadas `/api` para a porta `8080`.

---

## Integração com Agentes (MCP)

O projeto disponibiliza um servidor nativo Model Context Protocol via STDIO, permitindo que agentes como Claude Desktop e Cursor consultem preços e registrem compras diretamente por conversa.

### Configuração no Claude Desktop

Adicione a configuração ao seu arquivo `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "compras-residenciais": {
      "command": "C:/Projetos/Go/ComprasResidenciais/backend/bin/mcp.exe",
      "env": {
        "DATABASE_URL": "postgres://postgres:postgrespassword@localhost:5432/compras_residenciais?sslmode=disable",
        "GEMINI_API_KEY": "sua_chave_gemini_aqui"
      }
    }
  }
}
```

> [!TIP]
> Também é possível rodar o servidor MCP dinamicamente via `go run`:
> ```json
> "command": "go",
> "args": ["run", "cmd/mcp/main.go"]
> ```

### Ferramentas MCP Disponíveis

| Ferramenta | Descrição |
| :--- | :--- |
| `parse_receipt_image` | Analisa foto ou PDF de cupom fiscal em Base64 e cria rascunho de compra com itens extraídos. |
| `compare_item_prices` | Localiza onde determinado produto foi comprado mais barato (preço normalizado) e histórico de valores. |
| `get_monthly_price_trends` | Retorna a oscilação do preço médio por períodos do mês (dias 1–7, 8–14, 15–21, 22–31). |
| `confirm_purchase` | Confirma um rascunho de compra associando o morador comprador da residência. |
| `list_household_users` | Lista todos os moradores cadastrados na casa. |
| `delete_household_user` | Bloqueia ou remove um morador com base na existência de compras associadas a ele. |
| `get_global_monthly_summary` | Retorna o total de compras do mês, divisão por morador e mercados mais visitados. |
| `create_manual_purchase` | Registra manualmente uma compra contendo múltiplos itens. |

---

## Referência da API

A API RESTful está estruturada sob o prefixo `/api/v1`:

### Verificação de Saúde
- `GET /api/v1/health` - Status operacional do serviço.

### Moradores
- `GET /api/v1/users` - Lista todos os moradores cadastrados.
- `POST /api/v1/users` - Cadastra um novo morador (`{ "name": "...", "email": "..." }`).
- `DELETE /api/v1/users/:id` - Remove ou bloqueia um morador.

### Estabelecimentos
- `GET /api/v1/stores` - Lista os estabelecimentos comerciais registrados.
- `POST /api/v1/stores` - Cadastra um novo estabelecimento.

### Compras
- `GET /api/v1/purchases` - Lista as compras recentes com itens e valores.
- `GET /api/v1/purchases/:id` - Obtém os detalhes de uma compra específica por UUID.
- `POST /api/v1/purchases/upload` - Envio de imagem ou PDF da nota fiscal (`multipart/form-data`).
- `POST /api/v1/purchases/manual` - Registro manual de compra com itens.
- `PUT /api/v1/purchases/:id/confirm` - Confirmação do rascunho e associação do morador.

### Inteligência & Preços
- `GET /api/v1/analytics/search-products?q={termo}` - Busca semântica de produtos canônicos.
- `GET /api/v1/analytics/price-comparison/:productId` - Comparativo de preço por unidade entre estabelecimentos.
- `GET /api/v1/analytics/price-oscillation/:productId` - Curva de oscilação do preço ao longo do mês.
- `GET /api/v1/analytics/summary` - Resumo financeiro consolidado do mês.

---

## Configuração

| Variável de Ambiente | Padrão | Descrição |
| :--- | :--- | :--- |
| `PORT` | `8080` | Porta do servidor HTTP Gin. |
| `DATABASE_URL` | `postgres://postgres:postgrespassword@localhost:5432/compras_residenciais?sslmode=disable` | String de conexão com o PostgreSQL. |
| `GEMINI_API_KEY` | _(vazio)_ | Chave da API Google Gemini para OCR e vetorização semântica. |
| `SEFAZ_CERT_PATH` | _(vazio)_ | Caminho para o certificado A1 PKCS#12 (opcional). |
| `SEFAZ_CERT_PASSWORD` | _(vazio)_ | Senha do certificado digital A1 (opcional). |
