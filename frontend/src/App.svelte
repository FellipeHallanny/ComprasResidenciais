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

  import Header from './components/Header.svelte';
  import GlobalSummary from './components/GlobalSummary.svelte';
  import ProductSearch from './components/ProductSearch.svelte';
  import OpportunityCard from './components/OpportunityCard.svelte';
  import PriceOscillationChart from './components/PriceOscillationChart.svelte';
  import ProductHistoryTable from './components/ProductHistoryTable.svelte';
  import PurchaseModal from './components/PurchaseModal.svelte';
  import PurchaseList from './components/PurchaseList.svelte';
  import HouseholdUsers from './components/HouseholdUsers.svelte';
  import StoresView from './components/StoresView.svelte';

  // Active view tab
  let activeTab = $state<'dashboard' | 'purchases' | 'household' | 'stores'>('dashboard');

  // Modal open state
  let isPurchaseModalOpen = $state(false);

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
      users = usersRes;
      stores = storesRes;
      purchases = purchasesRes;
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
      purchases = await api.getPurchases(50);
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
      users = await api.getUsers();
    } catch (err) {
      console.error(err);
    } finally {
      loadingUsers = false;
    }
  }

  async function refreshStores() {
    loadingStores = true;
    try {
      stores = await api.getStores();
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

  onMount(() => {
    loadInitialData();
  });
</script>

<div class="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans">
  <!-- Header / Navigation Bar -->
  <Header
    {activeTab}
    onselectTab={(t) => (activeTab = t)}
    onopenPurchaseModal={() => (isPurchaseModalOpen = true)}
  />

  <!-- Main Content Container -->
  <main class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8">
    {#if activeTab === 'dashboard'}
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
      <span class="flex items-center gap-1.5 text-slate-400">
        <span class="w-2 h-2 rounded-full bg-emerald-400 inline-block animate-ping"></span>
        PostgreSQL 16 com busca semântica pgvector
      </span>
    </div>
  </footer>
</div>
