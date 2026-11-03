# Design Doc: Sistema de Controle e Inteligência de Compras Residenciais

- **Data:** 2026-11-02
- **Status:** Implementado (Backend Go Hexagonal + Frontend Svelte 5)
- **Autor/Equipe:** DeepMind Antigravity & Usuário

---

## 1. Contexto e Motivação (Intent & The Why)

Nas compras domésticas do dia a dia, diferentes pessoas da casa compram itens essenciais em diversos estabelecimentos (supermercados, atacadistas, feiras de rua, farmácias). Atualmente, não há visibilidade clara de:
1. Em qual mercado determinado produto é vendido mais barato.
2. Em que época do mês (início, meio ou fim) os preços oscilam para mais ou para menos.
3. Se embalagens maiores (ex: 1 pacote de 5 kg) realmente compensam mais do que múltiplos pacotes menores (ex: 5 pacotes de 1 kg), independente da marca.

Além disso, cadastrar compras item por item de forma avulsa gera muito atrito, levando ao abandono do controle. É fundamental agrupar os itens por "compra/sessão/nota fiscal" e oferecer formas automatizadas de captura (leitura de notas fiscais via IA multimodal e integração fiscal), mantendo o sistema acessível tanto via interface web moderna quanto diretamente por agentes de IA (Claude Desktop, Cursor) via Model Context Protocol (MCP).

---

## 2. Metas, Não-Metas e Anti-Metas

### Metas (Goals)
- **Registro Ágil de Compras:** Permitir registrar compras completas contendo múltiplos itens de uma vez, através de:
  - Upload de foto ou PDF de cupom fiscal com extração inteligente via LLM multimodal.
  - Formulário manual rápido para compras sem nota fiscal (feiras, comércios locais).
  - Consulta estruturada por chave/número de NF-e/NFC-e (porta preparada para SEFAZ com certificado).
- **Conferência em Rascunho:** Possibilidade de revisar os itens extraídos e vincular o morador/comprador da casa antes de efetivar a compra.
- **Normalização de Preços por Unidade:** Calcular automaticamente o preço normalizado (R$/kg, R$/litro, R$/unidade), permitindo comparação fidedigna entre diferentes formatos de embalagem e diferentes marcas.
- **Inteligência de Mercado e Temporal:** Painel analítico que responde:
  - Onde comprar determinado item pelo menor preço histórico/recente.
  - Como o preço oscila ao longo dos períodos do mês (dividido em quinzenas ou semanas).
- **Arquitetura Hexagonal Estrita em Go:**
  - Núcleo de domínio desacoplado de frameworks e provedores externos.
  - Exposição dupla: API HTTP REST via Gin e Servidor MCP via STDIO (`mark3labs/mcp-go`) para agentes autônomos.
  - Persistência em PostgreSQL usando `pgx/v5` e busca semântica vetorial com `pgvector-go`.
- **Frontend Moderno em Svelte 5:** Aplicação em Svelte 5 (com novas Runes `$state`, `$derived`, `$props`, `$effect`), TypeScript (modo estrito), Tailwind CSS v4, Lucide Icons e Vite.
- **Separação de Projetos:** Estrutura isolada em diretórios `/backend` e `/frontend`.

### Não-Metas (Non-Goals)
- Não abrangerá conciliação bancária, extratos de cartão de crédito ou fluxo financeiro além do escopo de compras de produtos e estabelecimentos.
- Não exigirá certificado digital obrigatório no lançamento inicial (o adaptador SEFAZ ficará pronto em código, mas inativo por padrão até inserção do certificado A1).
- Não será um ERP empresarial; o foco é gestão doméstica/residencial.

### Anti-Metas (Anti-Goals)
- **Sem digitação forçada de item avulso:** Nunca forçar o usuário a cadastrar um produto antes de cadastrar uma compra.
- **Sem acoplamento entre regras de negócio e camada HTTP/MCP:** Gin e MCP STDIO não conterão regras de cálculo de preços ou normalização; ambos delegarão exclusivamente para Use Cases do domínio.

---

## 3. Restrições Técnicas (Constraints)

- **Linguagem & Backend:** Go 1.27, padrão Ports and Adapters (Hexagonal), localizado na pasta `/backend`.
- **Banco de Dados:** PostgreSQL 16+ com extensão `pgvector`.
- **Driver de BD:** `github.com/jackc/pgx/v5` com conexão via `pgxpool`.
- **Driver Vetorial:** `github.com/pgvector/pgvector-go`.
- **Servidor MCP:** `github.com/mark3labs/mcp-go` operando sobre transporte STDIO.
- **Motor HTTP:** `github.com/gin-gonic/gin`.
- **Provedor de IA (Backend):** Google Gemini API (modelos `gemini-1.5-flash` / `gemini-2.0` para extração multimodal e `text-embedding-004` para vetorização semântica).
- **Frontend:** Svelte 5 (Runes `$state`, `$derived`, `$props`, `$effect`, `$bindable`), Vite, TypeScript (`strict: true`), Tailwind CSS v4, `@lucide/svelte`, localizado na pasta `/frontend`.
- **Ambiente Operacional:** Windows, PowerShell.

---

## 4. Estrutura de Diretórios Separada

```
ComprasResidenciais/
├── backend/                  # Backend Go Hexagonal + MCP Server
│   ├── cmd/
│   │   ├── server/           # Entrada HTTP Gin
│   │   └── mcp/              # Entrada Servidor MCP STDIO
│   ├── internal/
│   │   ├── core/             # Domínio e Use Cases
│   │   └── adapters/         # PostgreSQL, Gemini, SEFAZ, Gin, MCP
│   ├── migrations/           # Scripts SQL (pgvector HNSW)
│   ├── docker-compose.yml
│   └── go.mod & go.sum
└── frontend/                 # Frontend SPA Svelte 5 + Vite
    ├── src/
    │   ├── components/       # Componentes Svelte 5 com Runes
    │   ├── lib/              # api.ts
    │   ├── types.ts          # Contratos TypeScript
    │   └── App.svelte
    ├── vite.config.ts        # Proxy para porta 8080
    └── package.json
```

---

## 5. Decisões de Arquitetura e Como o Sistema se Comporta

### 5.1. Modelagem de Domínio (Entidades e Relações)

```
[Morador/Usuário] (1) ── (N) [Compra / Sessão]
                                │ (1)
                                ├── (N) [Item de Compra (Bruto)]
                                │           │ (N)
                                │           ▼
                                └── (1) [Produto Canônico Normalizado]
                                            - Nome Genérico (ex: "Arroz Branco")
                                            - Unidade Padrão (kg, L, un)
                                            - Embedding Vetorial (pgvector)
```

1. **Morador (User):** Identificador, Nome, Cor/Avatar, Status de Bloqueio.
2. **Estabelecimento (Store):** Nome fantasia, CNPJ (opcional), Localização/Endereço.
3. **Compra (Purchase):**
   - ID, Data da compra, Morador ID, Estabelecimento ID, Valor total da nota, Origem (Manual, LLM_Upload, SEFAZ), Status (Draft, Confirmed).
4. **Item de Compra (PurchaseItem):**
   - Descrição original da nota (ex: "ARROZ TIPO 1 TJ 5KG").
   - Quantidade comprada (ex: 1).
   - Embalagem/Multiplicador (ex: 5).
   - Unidade da embalagem (ex: "kg").
   - Quantidade total normalizada (ex: 5.0 kg).
   - Preço total pago no item (ex: R$ 23,95).
   - Preço unitário original (ex: R$ 23,95).
   - **Preço normalizado calculado (ex: R$ 4,79 / kg).**
   - Produto Canônico ID (vínculo semântico).
5. **Produto Canônico (CanonicalProduct):**
   - Nome padronizado (ex: "Arroz").
   - Categoria (ex: "Grãos & Cereais").
   - Unidade base canônica (ex: "kg").
   - Embedding vetorial (vetor float32 gerado via `text-embedding-004`).

### 5.2. Mecanismo de Inteligência e pgvector

- Ao registrar um item bruto, o sistema gera o embedding da descrição textual e busca no PostgreSQL com `pgvector` por produtos canônicos próximos (usando distância cosseno `<=>`).
- Se houver similaridade semântica acima de 0.82, o item é associado automaticamente ao produto canônico.
- Caso contrário, cria-se uma nova referência canônica.
- O cálculo do preço normalizado é realizado no domínio:
  $$\text{Preço Normalizado} = \frac{\text{Preço Total do Item}}{\text{Quantidade Total em Unidade Base}}$$

### 5.3. Portas e Adaptadores (Hexagonal Architecture)

#### Portas de Entrada (Driving/Primary Ports):
- `PurchaseUseCase`:
  - `ProcessReceipt(ctx, fileBytes, mimeType) -> DraftPurchase`
  - `ConfirmPurchase(ctx, command) -> PurchaseID`
  - `CreateManualPurchase(ctx, command) -> PurchaseID`
- `AnalyticsUseCase`:
  - `GetProductPriceComparison(ctx, canonicalProductID) -> PriceComparisonView`
  - `GetMonthlyPriceOscillation(ctx, canonicalProductID, period) -> OscillationView`
  - `GetGlobalSummary(ctx, month) -> GlobalSummaryView`
- `UserUseCase`:
  - `ListUsers(ctx) -> []User`
  - `CreateUser(ctx, name, email) -> User`
  - `DeleteUser(ctx, id) -> DeleteUserResult`

#### Adaptadores Primários (Driving Adapters):
- `Gin HTTP Handler`: rotas RESTful (`/api/v1/purchases`, `/api/v1/analytics`, `/api/v1/users`, `/api/v1/stores`).
- `MCP Server (STDIO)`: ferramentas expostas para agentes (Claude Desktop, Cursor).

#### Portas de Saída (Driven/Secondary Ports):
- `PurchaseRepository`: persistência de compras e itens.
- `ProductRepository`: persistência e busca vetorial de produtos canônicos via `pgvector`.
- `UserRepository`: persistência de moradores.
- `StoreRepository`: persistência e listagem de estabelecimentos.
- `ReceiptParser`: interface para interpretar notas (`GeminiMultimodalAdapter`, `SefazAdapterStub`).
- `EmbeddingProvider`: interface para vetorização de texto (`GeminiEmbeddingAdapter`).

### 5.4. Interface e Experiência Visual (Svelte 5)

Seguindo o layout aprovado no brainstorm (Opção A - Visão Foco em Produto e Normalização):
- **Barra de Busca Semântica:** Busca inteligente de produtos ("Arroz", "Leite", etc.) conectada ao backend com pgvector.
- **Card de Oportunidade:** Exibe menor preço normalizado registrado, maior preço, economia percentual e o estabelecimento vencedor.
- **Gráfico de Oscilação Temporal:** Variação do preço médio por períodos do mês (Dias 1-7, 8-14, 15-21, 22-31).
- **Tabela de Histórico:** Detalhe das últimas compras com Data, Estabelecimento, Morador/Comprador, Tamanho de Embalagem e Preço/kg ou Preço/L.
- **Modal de Entrada de Compra:**
  - Aba 1: Upload de foto/PDF do cupom fiscal com visualizador e extração automática via Gemini.
  - Aba 2: Chave da Nota Fiscal (com adapter preparado para SEFAZ).
  - Aba 3: Cadastro Manual item a item.
  - Tela de Conferência do Rascunho com seleção rápida do morador comprador antes de salvar.
