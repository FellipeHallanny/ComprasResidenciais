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
  import ShoppingListView from './components/ShoppingListView.svelte';
  import { Users, Store as StoreIcon, CheckCircle2 } from '@lucide/svelte';

  // Active view tab
  let activeTab = $state<'dashboard' | 'purchases' | 'shopping-list' | 'household' | 'stores'>('dashboard');

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
  }

  // Refresh helper functions
  async function refreshUsers() {
    users = await api.getUsers().catch(() => []);
  }

  async function refreshStores() {
    stores = await api.getStores().catch(() => []);
  }

  async function refreshPurchases() {
    purchases = await api.getPurchases(50).catch(() => []);
    summary = await api.getSummary().catch(() => null);
  }

  // Handle product selection from search
  async function handleProductSelect(product: CanonicalProduct) {
    selectedProduct = product;
    loadingProductDetails = true;
    try {
      const [compRes, oscRes] = await Promise.all([
        api.getPriceComparison(product.id),
        api.getMonthlyOscillation(product.id),
      ]);
      priceComparison = compRes;
      priceOscillation = oscRes;
    } catch (err) {
      console.error('Erro ao carregar detalhes do produto:', err);
    } finally {
      loadingProductDetails = false;
    }
  }

  // When purchase is created or confirmed successfully
  function handlePurchaseSuccess(purchase: Purchase) {
    isPurchaseModalOpen = false;
    globalNotification = `Compra no valor de ${new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(purchase.total_amount)} registrada e confirmada com sucesso!`;
    refreshPurchases();
    refreshStores();
    if (selectedProduct) {
      handleProductSelect(selectedProduct);
    }
    setTimeout(() => {
      globalNotification = null;
    }, 6000);
  }

  // Reset database (Only Admin)
  async function handleResetDatabase() {
    if (!auth.isAdmin) {
      alert('Apenas administradores podem reiniciar o banco de dados.');
      return;
    }
    if (confirm('ATENÇÃO: Deseja realmente zerar todos os dados do banco de dados? Esta ação é irreversível e apagará todas as compras, notas, estabelecimentos e moradores.')) {
      try {
        await api.resetDatabase();
        alert('Banco de dados reiniciado do zero com sucesso!');
        auth.logout(); // desconecta após reset
        window.location.reload();
      } catch (err: any) {
        alert('Falha ao reiniciar o banco: ' + (err.message || 'Erro desconhecido'));
      }
    }
  }

  onMount(async () => {
    // 1. Inicializa o estado de autenticação
    await auth.init();

    // 2. Se autenticado, carrega os dados globais
    if (auth.isAuthenticated) {
      await loadInitialData();
      // Se não houver produto selecionado, busca o primeiro produto disponível
      try {
        const prods = await api.searchProducts('');
        if (prods.length > 0) {
          handleProductSelect(prods[0]);
        }
      } catch {
        // ignore
      }
    }
  });

  // Reage a mudanças de autenticação
  $effect(() => {
    if (auth.isAuthenticated && users.length === 0 && !loadingUsers) {
      loadInitialData();
    }
  });
</script>

{#if auth.isLoading}
  <!-- Splash / Loading Inicial -->
  <div class="min-h-screen bg-[#0a0d14] flex flex-col items-center justify-center text-slate-400">
    <div class="w-7 h-7 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin mb-3"></div>
    <span class="text-xs font-medium text-slate-400">Carregando Compras Residenciais…</span>
  </div>
{:else if !auth.isAuthenticated}
  <!-- Tela de Login / Setup Inicial do Administrador -->
  <LoginView
    onsuccess={() => {
      loadInitialData();
    }}
  />
{:else}
  <!-- Aplicação Autenticada -->
  <div class="min-h-screen bg-[#0a0d14] text-slate-100 flex flex-col font-sans selection:bg-emerald-500/30 selection:text-emerald-200">
    <!-- Skip to content link for accessibility -->
    <a
      href="#main-content"
      class="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:px-4 focus:py-2 focus:bg-emerald-500 focus:text-slate-950 focus:rounded-lg focus:font-semibold focus:shadow-md focus:outline-none"
    >
      Pular para o conteúdo principal
    </a>

    <!-- Header de Navegação Superior -->
    <Header
      {activeTab}
      onselectTab={(t) => (activeTab = t)}
      onopenPurchaseModal={() => (isPurchaseModalOpen = true)}
      onresetDatabase={handleResetDatabase}
    />

    <!-- Global Alert Notification -->
    {#if globalNotification}
      <div class="max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 mt-4" aria-live="polite">
        <div class="p-3 rounded-lg bg-[#0e131f] border border-emerald-500/30 text-emerald-300 text-xs flex items-center justify-between">
          <div class="flex items-center gap-2">
            <CheckCircle2 class="w-4 h-4 text-emerald-400" aria-hidden="true" />
            <span>{globalNotification}</span>
          </div>
          <button
            onclick={() => (globalNotification = null)}
            class="text-emerald-400 hover:text-emerald-200 focus:outline-none rounded p-1 cursor-pointer"
            aria-label="Fechar notificação"
          >
            &times;
          </button>
        </div>
      </div>
    {/if}

    <!-- Main Content Container -->
    <main id="main-content" class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8">
      {#if activeTab === 'dashboard'}
        <!-- Zero Data Welcome Banner -->
        {#if !loadingUsers && !loadingStores && users.length === 0 && stores.length === 0}
          <div class="mb-8 p-5 sm:p-6 rounded-xl bg-[#0e131f] border border-[#1b2234]">
            <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
              <div>
                <h3 class="text-base sm:text-lg font-semibold text-white">Bem-vindo ao Compras Residenciais</h3>
                <p class="text-xs text-slate-400 max-w-xl mt-1">
                  Para iniciar o acompanhamento inteligente de gastos e variação de preços, adicione os primeiros moradores e supermercados da sua rotina.
                </p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <button
                  onclick={() => (activeTab = 'household')}
                  class="px-3.5 py-1.5 rounded-lg bg-[#141b29] hover:bg-[#1b2438] border border-[#212c42] text-slate-200 text-xs font-medium flex items-center gap-1.5 transition-colors cursor-pointer focus:outline-none"
                >
                  <Users class="w-3.5 h-3.5" aria-hidden="true" />
                  <span>Cadastrar Morador</span>
                </button>
                <button
                  onclick={() => (activeTab = 'stores')}
                  class="px-3.5 py-1.5 rounded-lg bg-[#141b29] hover:bg-[#1b2438] border border-[#212c42] text-slate-200 text-xs font-medium flex items-center gap-1.5 transition-colors cursor-pointer focus:outline-none"
                >
                  <StoreIcon class="w-3.5 h-3.5" aria-hidden="true" />
                  <span>Cadastrar Loja</span>
                </button>
              </div>
            </div>
          </div>
        {/if}

        <!-- KPIs Globais da Casa -->
        <GlobalSummary {summary} loading={loadingSummary} />

        <!-- Barra de Busca Semântica -->
        <ProductSearch
          {selectedProduct}
          onselect={handleProductSelect}
        />

        <!-- Card de Oportunidade -->
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

      {:else if activeTab === 'shopping-list'}
        <!-- Catálogo de Itens & Gerador de Lista de Compras Inteligente -->
        <ShoppingListView
          {users}
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
    <footer class="border-t border-[#1b2234] bg-[#0c101a] py-5 text-center text-xs text-slate-500">
      <div class="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-3">
        <span>Compras Residenciais &bull; Gestão e Inteligência de Gastos Domésticos</span>
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1.5 text-slate-400">
            <span class="w-2 h-2 rounded-full bg-emerald-500 inline-block" aria-hidden="true"></span>
            Dados atualizados localmente
          </span>
        </div>
      </div>
    </footer>
  </div>
{/if}
