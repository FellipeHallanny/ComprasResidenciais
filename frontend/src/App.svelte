<script lang="ts">
  import { onMount } from 'svelte';
  import type {
    User,
    Store,
    Purchase,
    CanonicalProduct,
    PriceComparisonView,
    MonthlyPriceOscillation,
    GlobalSummaryView
  } from './types';
  import { api } from './lib/api';
  import { auth } from './lib/auth.svelte';

  import Header from './components/Header.svelte';
  import LoginView from './components/LoginView.svelte';
  import GlobalSummary from './components/GlobalSummary.svelte';
  import ProductSearch from './components/ProductSearch.svelte';
  import OpportunityCard from './components/OpportunityCard.svelte';
  import PriceOscillationChart from './components/PriceOscillationChart.svelte';
  import ProductHistoryTable from './components/ProductHistoryTable.svelte';
  import PurchaseModal from './components/PurchaseModal.svelte';
  import PurchaseList from './components/PurchaseList.svelte';
  import HouseholdUsers from './components/HouseholdUsers.svelte';
  import StoresView from './components/StoresView.svelte';
  import { Users, Store as StoreIcon, Sparkles, CheckCircle2 } from '@lucide/svelte';

  // Active view tab
  let activeTab = $state<'dashboard' | 'purchases' | 'household' | 'stores'>('dashboard');

  // Modal open state
  let isPurchaseModalOpen = $state(false);
  let globalNotification = $state<string | null>(null);

  // Data states
  let summary = $state<GlobalSummaryView | null>(null);
  let users = $state<User[]>([]);
  let stores = $state<Store[]>([]);
  let purchases = $state<Purchase[]>([]);
  let loadingSummary = $state(false);
  let loadingPurchases = $state(false);
  let loadingUsers = $state(false);
  let loadingStores = $state(false);

  // Selected product & Analytics states
  let selectedProduct = $state<CanonicalProduct | null>(null);
  let priceComparison = $state<PriceComparisonView | null>(null);
  let priceOscillation = $state<MonthlyPriceOscillation | null>(null);
  let loadingProductDetails = $state(false);

  // Load initial global data
  async function loadInitialData() {
    loadingSummary = true;
    loadingUsers = true;
    loadingStores = true;
    loadingPurchases = true;

    try {
      const [sumRes, usersRes, storesRes, purchasesRes] = await Promise.all([
        api.getSummary().catch(() => null),
        api.getUsers().catch(() => []),
        api.getStores().catch(() => []),
        api.getPurchases(50).catch(() => []),
      ]);

      summary = sumRes;
      users = Array.isArray(usersRes) ? usersRes : [];
      stores = Array.isArray(storesRes) ? storesRes : [];
      purchases = Array.isArray(purchasesRes) ? purchasesRes : [];
    } finally {
      loadingSummary = false;
      loadingUsers = false;
      loadingStores = false;
      loadingPurchases = false;
    }

    // Try finding a default product to showcase
    try {
      const defaultProducts = await api.searchProducts('Arroz');
      if (defaultProducts && defaultProducts.length > 0) {
        handleProductSelect(defaultProducts[0]);
      } else {
        const anyProducts = await api.searchProducts('');
        if (anyProducts && anyProducts.length > 0) {
          handleProductSelect(anyProducts[0]);
        }
      }
    } catch {
      // ignore
    }
  }

  // Load details whenever a product is selected
  async function handleProductSelect(product: CanonicalProduct) {
    selectedProduct = product;
    loadingProductDetails = true;
    try {
      const [comp, osc] = await Promise.all([
        api.getPriceComparison(product.id).catch(() => null),
        api.getMonthlyOscillation(product.id).catch(() => null),
      ]);
      priceComparison = comp;
      priceOscillation = osc;
    } catch (err) {
      console.error('Erro ao carregar detalhes do produto:', err);
    } finally {
      loadingProductDetails = false;
    }
  }

  // Refresh handlers
  async function refreshPurchases() {
    loadingPurchases = true;
    try {
      const res = await api.getPurchases(50);
      purchases = Array.isArray(res) ? res : [];
      summary = await api.getSummary().catch(() => null);
      if (selectedProduct) {
        handleProductSelect(selectedProduct);
      }
    } catch (err) {
      console.error(err);
    } finally {
      loadingPurchases = false;
    }
  }

  async function refreshUsers() {
    loadingUsers = true;
    try {
      const res = await api.getUsers();
      users = Array.isArray(res) ? res : [];
    } catch (err) {
      console.error(err);
    } finally {
      loadingUsers = false;
    }
  }

  async function refreshStores() {
    loadingStores = true;
    try {
      const res = await api.getStores();
      stores = Array.isArray(res) ? res : [];
    } catch (err) {
      console.error(err);
    } finally {
      loadingStores = false;
    }
  }

  function handlePurchaseSuccess(p: Purchase) {
    isPurchaseModalOpen = false;
    refreshPurchases();
    refreshStores();
  }

  async function handleResetDatabase() {
    if (!auth.isAdmin) {
      alert('Apenas administradores podem zerar o banco de dados.');
      return;
    }

    const confirmation = confirm(
      '⚠️ ATENÇÃO: Deseja realmente zerar todo o banco de dados?\n\nIsso removerá permanentemente todas as compras, itens, moradores e lojas cadastradas, deixando o sistema completamente limpo.'
    );
    if (!confirmation) return;

    try {
      const res = await api.resetDatabase();
      globalNotification = res.message || 'Banco de dados reiniciado do zero com sucesso!';
      selectedProduct = null;
      priceComparison = null;
      priceOscillation = null;
      auth.logout();
      await auth.init();
    } catch (err: any) {
      alert(`Erro ao reiniciar banco: ${err.message}`);
    }
  }

  onMount(async () => {
    await auth.init();
    if (auth.isAuthenticated) {
      loadInitialData();
    }
  });
</script>

{#if auth.isLoading}
  <div class="min-h-screen bg-slate-950 flex flex-col items-center justify-center p-4">
    <div class="w-8 h-8 border-3 border-emerald-500 border-t-transparent rounded-full animate-spin mb-3"></div>
    <p class="text-xs text-slate-400 font-medium">Carregando autenticação...</p>
  </div>
{:else if !auth.isAuthenticated}
  <LoginView onsuccess={loadInitialData} />
{:else}
  <div class="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans">
    <!-- Header / Navigation Bar -->
    <Header
      {activeTab}
      onselectTab={(t) => (activeTab = t)}
      onopenPurchaseModal={() => (isPurchaseModalOpen = true)}
      onresetDatabase={handleResetDatabase}
    />

    <!-- Global Alert Notification -->
    {#if globalNotification}
      <div class="max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 mt-4">
        <div class="p-3.5 rounded-xl bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 text-xs flex items-center justify-between">
          <div class="flex items-center gap-2">
            <CheckCircle2 class="w-4 h-4 text-emerald-400" />
            <span>{globalNotification}</span>
          </div>
          <button onclick={() => (globalNotification = null)} class="text-emerald-400 hover:text-emerald-200">
            &times;
          </button>
        </div>
      </div>
    {/if}

    <!-- Main Content Container -->
    <main class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {#if activeTab === 'dashboard'}
        <!-- Zero Data Welcome Banner -->
        {#if !loadingUsers && !loadingStores && users.length === 0 && stores.length === 0}
          <div class="mb-8 p-6 rounded-2xl bg-gradient-to-r from-slate-900 to-slate-800/90 border border-emerald-500/30 shadow-xl">
            <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
              <div>
                <div class="flex items-center gap-2 text-emerald-400 text-xs font-semibold uppercase tracking-wider mb-1">
                  <Sparkles class="w-4 h-4" /> Banco de Dados Limpo & Pronto
                </div>
                <h3 class="text-xl font-bold text-white">Bem-vindo ao Compras Residenciais!</h3>
                <p class="text-xs text-slate-300 max-w-xl mt-1">
                  Seu banco de dados foi iniciado do zero. Para começar o acompanhamento de preços, cadastre os moradores e supermercados da casa.
                </p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <button
                  onclick={() => (activeTab = 'household')}
                  class="px-4 py-2 rounded-xl bg-amber-500/15 hover:bg-amber-500/25 border border-amber-500/30 text-amber-300 text-xs font-semibold flex items-center gap-1.5 transition cursor-pointer"
                >
                  <Users class="w-3.5 h-3.5" />
                  <span>1. Cadastrar Morador</span>
                </button>
                <button
                  onclick={() => (activeTab = 'stores')}
                  class="px-4 py-2 rounded-xl bg-purple-500/15 hover:bg-purple-500/25 border border-purple-500/30 text-purple-300 text-xs font-semibold flex items-center gap-1.5 transition cursor-pointer"
                >
                  <StoreIcon class="w-3.5 h-3.5" />
                  <span>2. Cadastrar Loja</span>
                </button>
              </div>
            </div>
          </div>
        {/if}

        <!-- KPIs Globais da Casa -->
        <GlobalSummary {summary} loading={loadingSummary} />

        <!-- Barra de Busca Semântica com pgvector -->
        <ProductSearch
          {selectedProduct}
          onselect={handleProductSelect}
        />

        <!-- Card de Oportunidade (Opção A) -->
        <OpportunityCard
          comparison={priceComparison}
          loading={loadingProductDetails}
        />

        <!-- Gráfico de Oscilação Temporal no Mês -->
        <PriceOscillationChart
          oscillation={priceOscillation}
          loading={loadingProductDetails}
        />

        <!-- Tabela de Histórico de Compras do Produto Normalizado -->
        {#if selectedProduct && priceComparison}
          <ProductHistoryTable
            purchases={priceComparison.recent_purchases}
            baseUnit={selectedProduct.base_unit}
            productName={selectedProduct.name}
          />
        {/if}

      {:else if activeTab === 'purchases'}
        <!-- Lista Geral de Compras Registradas -->
        <PurchaseList
          {purchases}
          {stores}
          {users}
          loading={loadingPurchases}
          onrefresh={refreshPurchases}
        />

      {:else if activeTab === 'household'}
        <!-- Gestão de Moradores -->
        <HouseholdUsers
          {users}
          loading={loadingUsers}
          onrefresh={refreshUsers}
        />

      {:else if activeTab === 'stores'}
        <!-- Gestão de Estabelecimentos -->
        <StoresView
          {stores}
          loading={loadingStores}
          onrefresh={refreshStores}
        />
      {/if}
    </main>

    <!-- Purchase Modal (Upload, SEFAZ, Manual e Conferência) -->
    {#if isPurchaseModalOpen}
      <PurchaseModal
        {users}
        {stores}
        onclose={() => (isPurchaseModalOpen = false)}
        onsuccess={handlePurchaseSuccess}
      />
    {/if}

    <!-- Footer -->
    <footer class="border-t border-slate-900 bg-slate-950/80 py-6 text-center text-xs text-slate-500">
      <div class="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-3">
        <span>Controle e Inteligência de Compras Residenciais &bull; Svelte 5 &amp; Go Hexagonal</span>
        <div class="flex items-center gap-4">
          <span class="flex items-center gap-1.5 text-slate-400">
            <span class="w-2 h-2 rounded-full bg-emerald-400 inline-block animate-ping"></span>
            PostgreSQL 16 com pgvector
          </span>
        </div>
      </div>
    </footer>
  </div>
{/if}
