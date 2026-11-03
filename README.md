<div align="center">
  <img src="frontend/public/favicon.svg" width="80" height="80" alt="Compras Residenciais Logo" />

  # Compras Residenciais

  **Intelligent residential grocery tracking, price normalization, and market intelligence platform.**

  [![Go Version](https://img.shields.io/badge/Go-1.27-00ADD8?style=flat-square&logo=go)](https://golang.org)
  [![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?style=flat-square&logo=svelte)](https://svelte.dev)
  [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16%2B_pgvector-4169E1?style=flat-square&logo=postgresql)](https://www.postgresql.org/)
  [![Model Context Protocol](https://img.shields.io/badge/MCP-Enabled-9B51E0?style=flat-square)](https://modelcontextprotocol.io/)
  [![Tailwind CSS](https://img.shields.io/badge/TailwindCSS-v4-06B6D4?style=flat-square&logo=tailwindcss)](https://tailwindcss.com)

  [English](README.md) • [Português (Brasil)](README.pt-BR.md)

  <p>
    <a href="#overview">Overview</a> •
    <a href="#key-features">Key Features</a> •
    <a href="#architecture">Architecture</a> •
    <a href="#tech-stack">Tech Stack</a> •
    <a href="#getting-started">Getting Started</a> •
    <a href="#model-context-protocol-mcp">MCP Agent Integration</a> •
    <a href="#api-reference">API Reference</a> •
    <a href="#configuration">Configuration</a>
  </p>
</div>

---

## Overview

In multi-person households, groceries and supplies are routinely purchased across different supermarkets, wholesalers, neighborhood markets, and pharmacies. Traditional budgeting apps require manual item entry and track total expenditures without granular product intelligence.

**Compras Residenciais** solves this problem by automating receipt capture and normalizing item pricing (e.g., standardizing different packaging sizes into R$/kg, R$/L, or R$/unit). This allows households to accurately determine:
1. Which store offers a given product at the lowest historical or recent unit price.
2. How prices fluctuate across different periods of the month (beginning, middle, or end).
3. Whether larger packages (e.g., 5 kg bag) actually yield savings compared to multi-packs of smaller sizes.
4. Total household spending split across individual members.

The application exposes both a modern **Svelte 5** web dashboard and an autonomous **Model Context Protocol (MCP)** server, enabling native interaction from AI agents such as Claude Desktop and Cursor.

---

## Key Features

- **Automated Receipt Ingestion (Multimodal AI)**: Upload photos or PDFs of receipts and invoices. Google Gemini extracts store details, purchase date, item descriptions, package sizes, quantities, and totals into an editable draft.
- **Unit Price Normalization**: Automatically converts varied units and package formats into baseline units (R$/kg, R$/L, R$/unit) for direct, unbiased comparison across brands and stores.
- **Semantic Product Grouping (pgvector)**: Groups raw store receipts items (e.g., `"ARROZ TP1 5KG"`) into canonical products (`"Arroz Branco"`) using 768-dimensional vector embeddings and HNSW cosine similarity.
- **Price Trend & Oscillation Analytics**: Tracks price variations over calendar periods (days 1–7, 8–14, 15–21, 22–31) to identify optimal purchase timing.
- **Store Opportunity Detection**: Highlights price differences and potential savings percentages for every tracked product.
- **Household Member Attribution**: Associates purchases with specific residents, tracking expense distributions while preserving data integrity via soft-blocking.
- **Dual Surface (Web & MCP)**: Full REST API with a responsive Svelte 5 single-page application, plus an STDIO MCP server for agentic automation.

---

## Architecture

The backend follows a strict **Hexagonal Architecture** (Ports and Adapters), keeping the core business logic completely decoupled from external frameworks, databases, and network transports.

```mermaid
flowchart TD
    subgraph DrivingAdapters["Driving Adapters (Primary)"]
        HTTP["Gin HTTP Handler\n(/api/v1/...)"]
        MCP["MCP Server (STDIO)\n(mark3labs/mcp-go)"]
    end

    subgraph CoreDomain["Core Domain & Services (Hexagon)"]
        PUC["PurchaseService"]
        AUC["AnalyticsService"]
        UUC["UserService"]
    end

    subgraph DrivenAdapters["Driven Adapters (Secondary)"]
        PG["PostgreSQL + pgvector\n(pgx/v5 & HNSW Index)"]
        GEM["Google Gemini\n(Multimodal & text-embedding-004)"]
        SFZ["SEFAZ Fiscal Adapter\n(mTLS Stub)"]
    end

    HTTP --> PUC & AUC & UUC
    MCP --> PUC & AUC & UUC
    PUC & AUC & UUC --> PG
    PUC --> GEM
    PUC --> SFZ
```

### Directory Structure

```
ComprasResidenciais/
├── backend/
│   ├── cmd/
│   │   ├── server/               # Gin REST HTTP server entrypoint
│   │   └── mcp/                  # STDIO MCP Server entrypoint
│   ├── internal/
│   │   ├── core/
│   │   │   ├── domain/           # Entities, value objects, and views
│   │   │   ├── ports/            # Inbound & outbound interface contracts
│   │   │   └── services/         # Application use cases & pricing calculations
│   │   ├── adapters/
│   │   │   ├── handlers/         # HTTP (Gin) and MCP controllers
│   │   │   ├── repositories/     # PostgreSQL storage & pgvector queries
│   │   │   ├── ai/               # Google Gemini multimodal & embeddings
│   │   │   └── sefaz/            # Fiscal invoice communication adapter
│   │   └── config/               # Environment & configuration loading
│   ├── migrations/               # PostgreSQL schema & HNSW index migrations
│   └── docker-compose.yml        # PostgreSQL with pgvector container definition
├── frontend/
│   ├── src/
│   │   ├── components/           # Svelte 5 runes components
│   │   ├── lib/                  # HTTP client & API abstraction
│   │   └── types.ts              # TypeScript domain types
│   ├── public/                   # Static icons and assets
│   └── vite.config.ts            # Vite build configuration & API proxy
└── docs/                         # Architecture & design specifications
```

---

## Tech Stack

| Layer | Technologies |
| :--- | :--- |
| **Backend Core** | Go 1.27, Hexagonal Architecture, `gin-gonic/gin` |
| **Agent Protocol** | `mark3labs/mcp-go` (STDIO transport) |
| **Database** | PostgreSQL 16+, `pgvector` extension, `jackc/pgx/v5` |
| **AI / Machine Learning** | Google Gemini (`gemini-1.5-flash` / `gemini-2.0`, `text-embedding-004`) |
| **Frontend Framework** | Svelte 5 (Runes: `$state`, `$derived`, `$props`, `$effect`), TypeScript |
| **Styling & Icons** | Tailwind CSS v4, Lucide Svelte (`@lucide/svelte`) |
| **Build & Tooling** | Vite 8, Docker Compose |

---

## Getting Started

### Prerequisites

Ensure you have the following installed locally:
- [Go 1.27+](https://golang.org/dl/)
- [Node.js 20+](https://nodejs.org/) and npm
- [Docker & Docker Compose](https://www.docker.com/)
- (Optional) [Google Gemini API Key](https://aistudio.google.com/) for automated receipt extraction and vector search

### 1. Start the Database

Launch PostgreSQL 16 with the `pgvector` extension enabled:

```bash
docker compose up -d
```

> [!NOTE]
> Database migrations in `backend/migrations/` (including table definitions and HNSW vector indexing) are applied automatically on application startup.

### 2. Configure Environment Variables

Create your environment configuration inside the `backend` directory:

```bash
cp backend/.env.example backend/.env
```

Edit `backend/.env` with your settings:

```env
PORT=8080
DATABASE_URL=postgres://postgres:postgrespassword@localhost:5432/compras_residenciais?sslmode=disable
GEMINI_API_KEY=your_gemini_api_key_here
```

> [!TIP]
> If `GEMINI_API_KEY` is not provided, manual purchase entry and price analytics remain fully functional; multimodal image extraction and semantic embeddings will operate in fallback mode.

### 3. Run the Backend Server

```bash
cd backend
go run cmd/server/main.go
```

The REST API will start on `http://localhost:8080`.

### 4. Run the Frontend Application

In a separate terminal:

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173` in your browser. The Vite development server automatically proxies `/api` calls to the backend on port `8080`.

---

## Model Context Protocol (MCP)

The project includes a native STDIO Model Context Protocol server, enabling AI assistants like Claude Desktop or Cursor to query price records, extract receipts, and register purchases on your behalf.

### Connecting to Claude Desktop

Add the server to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "compras-residenciais": {
      "command": "C:/Projetos/Go/ComprasResidenciais/backend/bin/mcp.exe",
      "env": {
        "DATABASE_URL": "postgres://postgres:postgrespassword@localhost:5432/compras_residenciais?sslmode=disable",
        "GEMINI_API_KEY": "your_gemini_api_key_here"
      }
    }
  }
}
```

> [!TIP]
> You can also run the MCP server directly using `go run`:
> ```json
> "command": "go",
> "args": ["run", "cmd/mcp/main.go"]
> ```

### Available MCP Tools

| Tool | Description |
| :--- | :--- |
| `parse_receipt_image` | Parses a base64-encoded receipt photo or PDF into an itemized draft purchase. |
| `compare_item_prices` | Finds the cheapest store and pricing history for a given canonical product. |
| `get_monthly_price_trends` | Returns price oscillation averages across month periods (days 1–7, 8–14, 15–21, 22–31). |
| `confirm_purchase` | Confirms a draft purchase and binds it to a specific household resident. |
| `list_household_users` | Retrieves the list of active residents. |
| `delete_household_user` | Soft-blocks or removes a resident depending on historical purchase associations. |
| `get_global_monthly_summary` | Returns monthly spending totals, resident breakdown, and top visited stores. |
| `create_manual_purchase` | Registers a multi-item purchase manually. |

---

## API Reference

The backend provides a RESTful HTTP API under `/api/v1`:

### Health Check
- `GET /api/v1/health` - Verify server and service status.

### Household Users
- `GET /api/v1/users` - List all residents.
- `POST /api/v1/users` - Create a new resident (`{ "name": "...", "email": "..." }`).
- `DELETE /api/v1/users/:id` - Delete or soft-block a resident.

### Stores
- `GET /api/v1/stores` - List all registered stores.
- `POST /api/v1/stores` - Create a new store record.

### Purchases
- `GET /api/v1/purchases` - List recent purchases with item breakdowns.
- `GET /api/v1/purchases/:id` - Retrieve a specific purchase by UUID.
- `POST /api/v1/purchases/upload` - Upload receipt image or PDF (`multipart/form-data`).
- `POST /api/v1/purchases/manual` - Submit a manually recorded purchase.
- `PUT /api/v1/purchases/:id/confirm` - Confirm a draft purchase and assign resident.

### Analytics & Intelligence
- `GET /api/v1/analytics/search-products?q={query}` - Semantic product search.
- `GET /api/v1/analytics/price-comparison/:productId` - Unit price comparison across stores.
- `GET /api/v1/analytics/price-oscillation/:productId` - Monthly price oscillation data.
- `GET /api/v1/analytics/summary` - Global monthly expenditure summary.

---

## Configuration

| Environment Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port for the Gin HTTP server. |
| `DATABASE_URL` | `postgres://postgres:postgrespassword@localhost:5432/compras_residenciais?sslmode=disable` | PostgreSQL connection string. |
| `GEMINI_API_KEY` | _(empty)_ | Google Gemini API key for OCR and text embeddings. |
| `SEFAZ_CERT_PATH` | _(empty)_ | Path to A1 PKCS#12 certificate file (optional). |
| `SEFAZ_CERT_PASSWORD` | _(empty)_ | Password for the A1 certificate (optional). |
