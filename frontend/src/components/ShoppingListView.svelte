<script lang="ts">
  import {
    ShoppingCart,
    Search,
    Check,
    CheckSquare,
    Square,
    Store,
    ArrowRight,
    Tag,
    Plus,
    Minus,
    AlertCircle,
    LayoutGrid,
    List,
    ArrowUpDown,
    ArrowDownAZ,
    ArrowUpAZ,
    Calendar
  } from '@lucide/svelte';
  import type { User, CatalogItemView, SelectedShoppingItem } from '../types';
  import { api } from '../lib/api';
  import ShoppingListModal from './ShoppingListModal.svelte';

  interface Props {
    users: User[];
  }

  let { users }: Props = $props();

  let catalog = $state<CatalogItemView[]>([]);
  let loading = $state(true);
  let errorMessage = $state('');

  // Busca e Filtros
  let searchQuery = $state('');
  let selectedCategory = $state('all');

  // Modo de visualização: Grade (cards) ou Lista (tabela/linhas compactas)
  let viewMode = $state<'grid' | 'list'>('grid');

  // Ordenação: Alfabética, Data, Menor Preço, Compras
  type SortOption = 'alpha-asc' | 'alpha-desc' | 'date-desc' | 'date-asc' | 'price-asc' | 'purchases-desc';
  let sortBy = $state<SortOption>('alpha-asc');

  // Mapa de itens selecionados: id -> SelectedShoppingItem
  let selectedMap = $state<Record<string, SelectedShoppingItem>>({});

  // Controle do modal de conferência / envio
  let isModalOpen = $state(false);

  // Carrega catálogo com preços otimizados
  async function loadCatalog() {
    loading = true;
    errorMessage = '';
    try {
      catalog = await api.getCatalogWithPrices();
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao carregar catálogo de produtos.';
    } finally {
      loading = false;
    }
  }

  // Carrega ao inicializar
  $effect(() => {
    loadCatalog();
  });

  // Categorias únicas encontradas no catálogo
  const categories = $derived.by(() => {
    const set = new Set<string>();
    for (const item of catalog) {
      if (item.category && item.category.trim()) {
        set.add(item.category.trim());
      }
    }
    return Array.from(set).sort();
  });

  // Retorna timestamp da data de compra ou criação do item
  function getItemDate(item: CatalogItemView): number {
    const d = item.last_purchased_at || item.cheapest_purchase_date || item.created_at;
    return d ? new Date(d).getTime() : 0;
  }

  // Formata data amigável em pt-BR
  function formatDate(dateStr?: string): string {
    if (!dateStr) return 'Sem registro';
    try {
      const d = new Date(dateStr);
      if (isNaN(d.getTime())) return 'Sem registro';
      return d.toLocaleDateString('pt-BR');
    } catch {
      return 'Sem registro';
    }
  }

  // Itens filtrados por busca, categoria e ordenados conforme seleção do usuário
  const filteredCatalog = $derived.by(() => {
    let list = catalog.filter((item) => {
      // Filtro por texto
      const q = searchQuery.trim().toLowerCase();
      const matchQuery =
        !q ||
        item.name.toLowerCase().includes(q) ||
        (item.category && item.category.toLowerCase().includes(q)) ||
        (item.cheapest_store_name && item.cheapest_store_name.toLowerCase().includes(q));

      // Filtro por categoria
      const matchCat =
        selectedCategory === 'all' ||
        (item.category && item.category.trim() === selectedCategory);

      return matchQuery && matchCat;
    });

    // Ordenação selecionada
    return list.sort((a, b) => {
      if (sortBy === 'alpha-asc') {
        return a.name.localeCompare(b.name, 'pt-BR');
      }
      if (sortBy === 'alpha-desc') {
        return b.name.localeCompare(a.name, 'pt-BR');
      }
      if (sortBy === 'date-desc') {
        return getItemDate(b) - getItemDate(a);
      }
      if (sortBy === 'date-asc') {
        return getItemDate(a) - getItemDate(b);
      }
      if (sortBy === 'price-asc') {
        const pa = a.lowest_price_per_unit > 0 ? a.lowest_price_per_unit : Number.MAX_VALUE;
        const pb = b.lowest_price_per_unit > 0 ? b.lowest_price_per_unit : Number.MAX_VALUE;
        return pa - pb;
      }
      if (sortBy === 'purchases-desc') {
        return (b.total_purchases || 0) - (a.total_purchases || 0);
      }
      return 0;
    });
  });

  // Lista de itens atualmente selecionados
  const selectedItemsList = $derived(Object.values(selectedMap));

  // Subtotal estimado
  const totalEstimated = $derived.by(() => {
    return selectedItemsList.reduce((sum, item) => {
      const price = item.product.lowest_price_per_unit > 0 ? item.product.lowest_price_per_unit : 0;
      return sum + price * item.desired_quantity;
    }, 0);
  });

  // Alterna ordenação de forma amigável
  function toggleSort(type: 'alpha' | 'date' | 'price' | 'purchases') {
    if (type === 'alpha') {
      sortBy = sortBy === 'alpha-asc' ? 'alpha-desc' : 'alpha-asc';
    } else if (type === 'date') {
      sortBy = sortBy === 'date-desc' ? 'date-asc' : 'date-desc';
    } else if (type === 'price') {
      sortBy = 'price-asc';
    } else if (type === 'purchases') {
      sortBy = 'purchases-desc';
    }
  }

  // Alterna seleção de um produto
  function toggleItem(product: CatalogItemView) {
    if (selectedMap[product.id]) {
      const updated = { ...selectedMap };
      delete updated[product.id];
      selectedMap = updated;
    } else {
      selectedMap = {
        ...selectedMap,
        [product.id]: {
          product,
          desired_quantity: 1,
        }
      };
    }
  }

  // Atualiza quantidade desejada de um item
  function updateQuantity(productId: string, qty: number) {
    if (qty <= 0) {
      removeItem(productId);
      return;
    }
    if (selectedMap[productId]) {
      selectedMap = {
        ...selectedMap,
        [productId]: {
          ...selectedMap[productId],
          desired_quantity: qty,
        }
      };
    }
  }

  // Remove item da seleção
  function removeItem(productId: string) {
    if (selectedMap[productId]) {
      const updated = { ...selectedMap };
      delete updated[productId];
      selectedMap = updated;
    }
  }

  // Selecionar todos os filtrados
  function selectAllFiltered() {
    const updated = { ...selectedMap };
    for (const item of filteredCatalog) {
      if (!updated[item.id]) {
        updated[item.id] = { product: item, desired_quantity: 1 };
      }
    }
    selectedMap = updated;
  }

  // Desmarcar todos
  function clearAllSelected() {
    selectedMap = {};
  }
</script>

<div class="space-y-6">
  <!-- Header Bar -->
  <div class="bg-[#0e131f] border border-[#1b2234] p-5 rounded-xl flex flex-col md:flex-row md:items-center justify-between gap-4 shadow-xs">
    <div class="flex items-center gap-3">
      <div class="h-10 w-10 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 flex items-center justify-center shrink-0">
        <ShoppingCart class="w-5 h-5 stroke-[2.2]" aria-hidden="true" />
      </div>
      <div>
        <h2 class="text-base sm:text-lg font-semibold text-white flex items-center gap-2">
          <span>Catálogo de Itens & Lista de Compras</span>
        </h2>
        <p class="text-xs text-slate-400 mt-0.5 text-pretty">
          Selecione os produtos necessários. O sistema analisa os preços históricos e indica os locais mais econômicos.
        </p>
      </div>
    </div>

    <!-- Quick Action / Selection Badge -->
    <div class="flex items-center gap-3 shrink-0">
      {#if selectedItemsList.length > 0}
        <button
          type="button"
          onclick={() => { isModalOpen = true; }}
          class="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold text-xs shadow-xs transition-all flex items-center gap-2 cursor-pointer active:translate-y-px focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
        >
          <ShoppingCart class="w-3.5 h-3.5 stroke-[2.5]" aria-hidden="true" />
          <span>Gerar Lista (<span class="tabular-nums font-mono">{selectedItemsList.length}</span>)</span>
          <ArrowRight class="w-3.5 h-3.5" aria-hidden="true" />
        </button>
      {/if}
    </div>
  </div>

  <!-- Messages -->
  {#if errorMessage}
    <div role="alert" aria-live="polite" class="p-3.5 rounded-lg bg-rose-500/10 border border-rose-500/25 text-rose-300 text-xs flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <AlertCircle class="w-4 h-4 shrink-0" aria-hidden="true" />
        <span>{errorMessage}</span>
      </div>
      <button
        type="button"
        onclick={loadCatalog}
        class="underline font-medium hover:text-white cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-rose-400 rounded px-1"
      >
        Tentar novamente
      </button>
    </div>
  {/if}

  <!-- Filters, Sorting & View Mode Toolbar -->
  <div class="bg-[#0e131f] border border-[#1b2234] p-4 rounded-xl space-y-3">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-3">
      <!-- Search Input -->
      <div class="relative flex-1 max-w-md">
        <label for="catalog-search-input" class="sr-only">Buscar produtos no catálogo</label>
        <input
          id="catalog-search-input"
          type="search"
          autocomplete="off"
          spellcheck={false}
          bind:value={searchQuery}
          placeholder="Buscar no catálogo (ex: Arroz, Café, Sabão)…"
          class="w-full pl-9 pr-3 py-2 bg-[#090d16] border border-[#1b2234] focus:border-emerald-500/50 rounded-lg text-slate-100 text-xs focus-visible:outline-none placeholder-slate-500 transition-colors"
        />
        <Search class="w-3.5 h-3.5 text-slate-500 absolute left-3 top-2.5 pointer-events-none" aria-hidden="true" />
      </div>

      <!-- Controls Group: Sorting, View Mode & Batch Actions -->
      <div class="flex flex-wrap items-center gap-2">
        <!-- Ordenação Dropdown -->
        <div class="flex items-center gap-1.5 bg-[#090d16] border border-[#1b2234] px-2.5 py-1.5 rounded-lg">
          <ArrowUpDown class="w-3.5 h-3.5 text-slate-400 shrink-0" aria-hidden="true" />
          <label for="catalog-sort-select" class="text-xs text-slate-400 font-medium hidden sm:inline">Ordenar:</label>
          <select
            id="catalog-sort-select"
            aria-label="Ordenar catálogo"
            bind:value={sortBy}
            class="bg-transparent text-xs text-slate-200 font-medium focus-visible:outline-none cursor-pointer pr-1"
          >
            <option value="alpha-asc" class="bg-[#0e131f] text-slate-200">Alfabética (A → Z)</option>
            <option value="alpha-desc" class="bg-[#0e131f] text-slate-200">Alfabética (Z → A)</option>
            <option value="date-desc" class="bg-[#0e131f] text-slate-200">Data (Mais recentes)</option>
            <option value="date-asc" class="bg-[#0e131f] text-slate-200">Data (Mais antigas)</option>
            <option value="price-asc" class="bg-[#0e131f] text-slate-200">Menor Preço</option>
            <option value="purchases-desc" class="bg-[#0e131f] text-slate-200">Mais Comprados</option>
          </select>
        </div>

        <!-- Alternador de Visualização (Grade / Lista) -->
        <div class="flex items-center bg-[#090d16] border border-[#1b2234] rounded-lg p-0.5" role="group" aria-label="Modo de visualização">
          <button
            type="button"
            aria-pressed={viewMode === 'grid'}
            onclick={() => { viewMode = 'grid'; }}
            class="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {viewMode === 'grid' ? 'bg-[#182030] text-emerald-400 shadow-sm border border-[#26334d]' : 'text-slate-400 hover:text-slate-200'}"
            title="Visualização em Grade de Cards"
          >
            <LayoutGrid class="w-3.5 h-3.5" aria-hidden="true" />
            <span class="hidden sm:inline">Cards</span>
          </button>

          <button
            type="button"
            aria-pressed={viewMode === 'list'}
            onclick={() => { viewMode = 'list'; }}
            class="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {viewMode === 'list' ? 'bg-[#182030] text-emerald-400 shadow-sm border border-[#26334d]' : 'text-slate-400 hover:text-slate-200'}"
            title="Visualização em Lista / Tabela"
          >
            <List class="w-3.5 h-3.5" aria-hidden="true" />
            <span class="hidden sm:inline">Lista</span>
          </button>
        </div>

        <!-- Action Buttons -->
        <div class="flex items-center gap-1.5">
          <button
            type="button"
            onclick={selectAllFiltered}
            disabled={filteredCatalog.length === 0}
            class="px-2.5 py-1.5 rounded-lg border border-[#1b2234] hover:border-[#27324b] bg-[#090d16] text-slate-300 hover:text-white text-xs font-medium transition-colors cursor-pointer flex items-center gap-1.5 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400"
            title="Marcar todos os itens visíveis"
          >
            <CheckSquare class="w-3.5 h-3.5 text-slate-400" aria-hidden="true" />
            <span class="hidden sm:inline">Marcar Todos</span>
          </button>

          {#if selectedItemsList.length > 0}
            <button
              type="button"
              onclick={clearAllSelected}
              class="px-2.5 py-1.5 rounded-lg border border-[#1b2234] hover:border-rose-500/40 bg-[#090d16] hover:bg-rose-500/10 text-slate-400 hover:text-rose-300 text-xs font-medium transition-colors cursor-pointer flex items-center gap-1.5 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-rose-400"
              title="Desmarcar todos os itens selecionados"
            >
              <Square class="w-3.5 h-3.5" aria-hidden="true" />
              <span>Desmarcar (<span class="tabular-nums font-mono">{selectedItemsList.length}</span>)</span>
            </button>
          {/if}
        </div>
      </div>
    </div>

    <!-- Category Pills -->
    {#if categories.length > 0}
      <div class="flex items-center gap-1.5 flex-wrap pt-1 text-xs">
        <span class="text-slate-500 text-[11px] mr-1">
          Categorias:
        </span>
        <button
          type="button"
          onclick={() => { selectedCategory = 'all'; }}
          class="px-2.5 py-1 rounded-md text-xs font-medium transition-all cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {selectedCategory === 'all' ? 'bg-emerald-500 text-slate-950 font-semibold' : 'bg-[#090d16] text-slate-400 hover:text-white border border-[#1b2234]'}"
        >
          Todas (<span class="tabular-nums font-mono">{catalog.length}</span>)
        </button>
        {#each categories as cat}
          <button
            type="button"
            onclick={() => { selectedCategory = cat; }}
            class="px-2.5 py-1 rounded-md text-xs font-medium transition-all cursor-pointer focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {selectedCategory === cat ? 'bg-emerald-500 text-slate-950 font-semibold' : 'bg-[#090d16] text-slate-400 hover:text-white border border-[#1b2234]'}"
          >
            {cat}
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Catalog Items Section -->
  {#if loading}
    <div class="p-16 text-center text-slate-400 text-xs sm:text-sm flex flex-col items-center justify-center gap-3" aria-live="polite">
      <div class="w-6 h-6 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
      <span>Carregando itens cadastrados e histórico de preços…</span>
    </div>
  {:else if filteredCatalog.length === 0}
    <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl p-12 text-center text-slate-400 space-y-3">
      <ShoppingCart class="w-8 h-8 text-slate-600 mx-auto" aria-hidden="true" />
      <h3 class="text-sm font-semibold text-white">Nenhum produto encontrado</h3>
      <p class="text-xs text-slate-500 max-w-sm mx-auto text-pretty">
        {#if searchQuery || selectedCategory !== 'all'}
          Nenhum item corresponde aos filtros selecionados. Tente alterar a busca.
        {:else}
          Nenhum produto cadastrado no catálogo ainda. Registre compras para iniciar o monitoramento.
        {/if}
      </p>
    </div>
  {:else if viewMode === 'grid'}
    <!-- ============================================== -->
    <!-- VISUALIZAÇÃO 1: GRADE DE CARDS                 -->
    <!-- ============================================== -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each filteredCatalog as item}
        {@const isSelected = !!selectedMap[item.id]}
        {@const selectedItem = selectedMap[item.id]}
        {@const itemDateStr = item.last_purchased_at || item.cheapest_purchase_date || item.created_at}
        <div
          class="border rounded-xl p-4 transition-all flex flex-col justify-between {isSelected ? 'bg-[#121927] border-emerald-500/40 shadow-xs' : 'bg-[#0e131f] border-[#1b2234] hover:border-[#27324b]'}"
        >
          <!-- Top Row: Checkbox + Product Name -->
          <div>
            <div class="flex items-start gap-3">
              <button
                type="button"
                role="checkbox"
                aria-checked={isSelected}
                aria-label={isSelected ? `Desmarcar ${item.name}` : `Marcar ${item.name}`}
                onclick={() => toggleItem(item)}
                class="mt-0.5 h-4.5 w-4.5 rounded border flex items-center justify-center transition-colors cursor-pointer shrink-0 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {isSelected ? 'bg-emerald-500 border-emerald-500 text-slate-950' : 'border-[#263148] bg-[#090d16] text-transparent hover:border-slate-500'}"
              >
                <Check class="w-3 h-3 stroke-[3]" aria-hidden="true" />
              </button>

              <div class="min-w-0 flex-1">
                <button
                  type="button"
                  onclick={() => toggleItem(item)}
                  class="text-left font-semibold text-white text-sm hover:text-emerald-400 transition-colors line-clamp-2 cursor-pointer focus-visible:outline-none rounded"
                >
                  {item.name}
                </button>
                <div class="flex items-center gap-2 mt-1">
                  <span class="text-[10px] font-medium px-1.5 py-0.5 rounded bg-[#141b29] text-slate-400 border border-[#212c42]">
                    {item.category || 'Geral' }
                  </span>
                  <span class="text-[10px] text-slate-500 font-mono">
                    Unidade: {item.base_unit}
                  </span>
                </div>
              </div>
            </div>

            <!-- Price & Store Intelligence Info -->
            <div class="mt-3 p-2.5 rounded-lg bg-[#090d16] border border-[#1b2234] space-y-1.5">
              {#if item.lowest_price_per_unit > 0}
                <div class="flex items-center justify-between text-xs">
                  <span class="text-slate-400">Menor Preço:</span>
                  <span class="font-semibold font-mono tabular-nums text-emerald-400">
                    {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(item.lowest_price_per_unit)}
                    <span class="text-[10px] text-slate-400 font-normal">/{item.base_unit}</span>
                  </span>
                </div>
                {#if item.cheapest_store_name}
                  <div class="flex items-center justify-between text-[11px] text-slate-400">
                    <span class="flex items-center gap-1 text-slate-500 truncate">
                      <Store class="w-3.5 h-3.5 text-emerald-400 shrink-0" aria-hidden="true" /> Loja:
                    </span>
                    <span class="font-medium text-slate-300 truncate max-w-[150px]" title={item.cheapest_store_name}>
                      {item.cheapest_store_name}
                    </span>
                  </div>
                {/if}
              {:else}
                <div class="text-[11px] text-slate-500 italic py-0.5 text-center">
                  Sem compras registradas
                </div>
              {/if}

              <!-- Informação de Data -->
              {#if itemDateStr}
                <div class="flex items-center justify-between text-[10px] text-slate-500 pt-1 border-t border-[#1b2234]">
                  <span class="flex items-center gap-1">
                    <Calendar class="w-3 h-3 text-slate-500" aria-hidden="true" /> Último registro:
                  </span>
                  <span class="font-mono tabular-nums text-slate-400">{formatDate(itemDateStr)}</span>
                </div>
              {/if}
            </div>
          </div>

          <!-- Bottom: Quantity Adjuster (if selected) or Select button -->
          <div class="mt-4 pt-3 border-t border-[#1b2234] flex items-center justify-between">
            {#if isSelected && selectedItem}
              <div class="flex items-center gap-1 text-xs text-emerald-400 font-medium">
                <Check class="w-3.5 h-3.5" aria-hidden="true" />
                <span>Na lista</span>
              </div>

              <!-- Quantity Controls -->
              <div class="flex items-center border border-emerald-500/40 rounded-lg overflow-hidden bg-[#090d16]">
                <button
                  type="button"
                  onclick={() => updateQuantity(item.id, selectedItem.desired_quantity - 1)}
                  class="px-2 py-1 text-slate-400 hover:text-white hover:bg-[#182030] transition-colors cursor-pointer focus-visible:outline-none"
                  aria-label="Diminuir quantidade de {item.name}"
                  title="Diminuir"
                >
                  <Minus class="w-3 h-3" aria-hidden="true" />
                </button>
                <input
                  type="number"
                  min="1"
                  aria-label="Quantidade de {item.name}"
                  value={selectedItem.desired_quantity}
                  oninput={(e) => updateQuantity(item.id, parseInt((e.target as HTMLInputElement).value) || 1)}
                  class="w-8 text-center font-mono tabular-nums font-semibold text-xs text-white bg-transparent focus-visible:outline-none"
                />
                <button
                  type="button"
                  onclick={() => updateQuantity(item.id, selectedItem.desired_quantity + 1)}
                  class="px-2 py-1 text-slate-400 hover:text-white hover:bg-[#182030] transition-colors cursor-pointer focus-visible:outline-none"
                  aria-label="Aumentar quantidade de {item.name}"
                  title="Aumentar"
                >
                  <Plus class="w-3 h-3" aria-hidden="true" />
                </button>
              </div>
            {:else}
              <span class="text-[11px] text-slate-500 tabular-nums font-mono">
                {item.total_purchases} {item.total_purchases === 1 ? 'compra' : 'compras'}
              </span>

              <button
                type="button"
                onclick={() => toggleItem(item)}
                class="px-2.5 py-1 rounded-md border border-[#1b2234] hover:border-emerald-500/40 bg-[#090d16] hover:bg-emerald-500/10 text-slate-300 hover:text-emerald-400 text-xs font-medium transition-colors cursor-pointer flex items-center gap-1 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400"
              >
                <Plus class="w-3 h-3" aria-hidden="true" />
                <span>Adicionar</span>
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <!-- ============================================== -->
    <!-- VISUALIZAÇÃO 2: LISTA COMPACTA / TABELA        -->
    <!-- ============================================== -->
    <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl overflow-hidden shadow-xs">
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse text-xs">
          <thead>
            <tr class="bg-[#090d16] border-b border-[#1b2234] text-slate-400 text-xs font-medium">
              <th class="py-3 px-4 w-12 text-center">Sel.</th>
              <th class="py-3 px-4">
                <button
                  type="button"
                  class="flex items-center gap-1.5 hover:text-white transition-colors cursor-pointer focus-visible:outline-none rounded"
                  onclick={() => toggleSort('alpha')}
                >
                  <span>Produto</span>
                  {#if sortBy === 'alpha-asc'}
                    <ArrowDownAZ class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
                  {:else if sortBy === 'alpha-desc'}
                    <ArrowUpAZ class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
                  {/if}
                </button>
              </th>
              <th class="py-3 px-4">
                <button
                  type="button"
                  class="flex items-center gap-1.5 hover:text-white transition-colors cursor-pointer focus-visible:outline-none rounded"
                  onclick={() => toggleSort('price')}
                >
                  <span>Menor Preço</span>
                  {#if sortBy === 'price-asc'}
                    <ArrowUpDown class="w-3 h-3 text-emerald-400" aria-hidden="true" />
                  {/if}
                </button>
              </th>
              <th class="py-3 px-4">Loja Mais Barata</th>
              <th class="py-3 px-4">
                <button
                  type="button"
                  class="flex items-center gap-1.5 hover:text-white transition-colors cursor-pointer focus-visible:outline-none rounded"
                  onclick={() => toggleSort('date')}
                >
                  <span>Último Registro</span>
                  {#if sortBy.startsWith('date')}
                    <Calendar class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
                  {/if}
                </button>
              </th>
              <th class="py-3 px-4 text-center">
                <button
                  type="button"
                  class="flex items-center justify-center gap-1.5 hover:text-white transition-colors cursor-pointer focus-visible:outline-none rounded mx-auto"
                  onclick={() => toggleSort('purchases')}
                >
                  <span>Compras</span>
                  {#if sortBy === 'purchases-desc'}
                    <ArrowUpDown class="w-3 h-3 text-emerald-400" aria-hidden="true" />
                  {/if}
                </button>
              </th>
              <th class="py-3 px-4 text-right min-w-[130px]">Adicionar</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[#182030]">
            {#each filteredCatalog as item}
              {@const isSelected = !!selectedMap[item.id]}
              {@const selectedItem = selectedMap[item.id]}
              {@const itemDateStr = item.last_purchased_at || item.cheapest_purchase_date || item.created_at}
              <tr class="transition-colors hover:bg-[#131926] {isSelected ? 'bg-[#121927]' : ''}">
                <!-- Checkbox -->
                <td class="py-3 px-4 text-center">
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked={isSelected}
                    aria-label={isSelected ? `Desmarcar ${item.name}` : `Marcar ${item.name}`}
                    onclick={() => toggleItem(item)}
                    class="h-4.5 w-4.5 rounded border flex items-center justify-center transition-colors cursor-pointer mx-auto focus-visible:outline-none {isSelected ? 'bg-emerald-500 border-emerald-500 text-slate-950' : 'border-[#263148] bg-[#090d16] text-transparent hover:border-slate-500'}"
                  >
                    <Check class="w-3 h-3 stroke-[3]" aria-hidden="true" />
                  </button>
                </td>

                <!-- Nome do Produto e Categoria -->
                <td class="py-3 px-4">
                  <div class="min-w-0">
                    <button
                      type="button"
                      onclick={() => toggleItem(item)}
                      class="text-left font-semibold text-slate-100 hover:text-emerald-400 transition-colors cursor-pointer focus-visible:outline-none rounded"
                    >
                      {item.name}
                    </button>
                    <div class="flex items-center gap-2 mt-0.5">
                      <span class="text-[10px] font-medium px-1.5 py-0.2 rounded bg-[#141b29] text-slate-400 border border-[#212c42]">
                        {item.category || 'Geral'}
                      </span>
                      <span class="text-[10px] text-slate-500 font-mono">
                        Base: {item.base_unit}
                      </span>
                    </div>
                  </div>
                </td>

                <!-- Menor Preço -->
                <td class="py-3 px-4 whitespace-nowrap">
                  {#if item.lowest_price_per_unit > 0}
                    <div class="font-semibold font-mono tabular-nums text-emerald-400 text-xs sm:text-sm">
                      {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(item.lowest_price_per_unit)}
                      <span class="text-[10px] text-slate-400 font-normal">/{item.base_unit}</span>
                    </div>
                    {#if item.cheapest_package_info}
                      <span class="text-[10px] text-slate-500 font-mono tabular-nums block">({item.cheapest_package_info})</span>
                    {/if}
                  {:else}
                    <span class="text-slate-500 italic text-[11px]">Sem preço</span>
                  {/if}
                </td>

                <!-- Local Mais Barato -->
                <td class="py-3 px-4">
                  {#if item.cheapest_store_name}
                    <div class="flex items-center gap-1.5">
                      <Store class="w-3.5 h-3.5 text-emerald-400 shrink-0" aria-hidden="true" />
                      <div class="min-w-0">
                        <span class="font-medium text-slate-200 block truncate max-w-[180px]" title={item.cheapest_store_name}>
                          {item.cheapest_store_name}
                        </span>
                        {#if item.cheapest_store_address}
                          <span class="text-[10px] text-slate-500 block truncate max-w-[180px]" title={item.cheapest_store_address}>
                            {item.cheapest_store_address}
                          </span>
                        {/if}
                      </div>
                    </div>
                  {:else}
                    <span class="text-slate-500 italic text-[11px]">-</span>
                  {/if}
                </td>

                <!-- Data Última Compra / Registro -->
                <td class="py-3 px-4 whitespace-nowrap">
                  <div class="flex items-center gap-1.5 text-slate-400 font-mono tabular-nums text-[11px]">
                    <Calendar class="w-3 h-3 text-slate-500 shrink-0" aria-hidden="true" />
                    <span>{formatDate(itemDateStr)}</span>
                  </div>
                </td>

                <!-- Total de Compras -->
                <td class="py-3 px-4 text-center whitespace-nowrap">
                  <span class="px-2 py-0.5 rounded-full text-[10px] font-semibold tabular-nums font-mono bg-[#141b29] border border-[#212c42] text-slate-400">
                    {item.total_purchases}
                  </span>
                </td>

                <!-- Ações / Quantidade -->
                <td class="py-3 px-4 text-right whitespace-nowrap">
                  {#if isSelected && selectedItem}
                    <div class="inline-flex items-center border border-emerald-500/40 rounded-lg overflow-hidden bg-[#090d16]">
                      <button
                        type="button"
                        onclick={() => updateQuantity(item.id, selectedItem.desired_quantity - 1)}
                        class="px-2 py-1 text-slate-400 hover:text-white hover:bg-[#182030] transition-colors cursor-pointer focus-visible:outline-none"
                        aria-label="Diminuir quantidade de {item.name}"
                        title="Diminuir"
                      >
                        <Minus class="w-3 h-3" aria-hidden="true" />
                      </button>
                      <input
                        type="number"
                        min="1"
                        aria-label="Quantidade de {item.name}"
                        value={selectedItem.desired_quantity}
                        oninput={(e) => updateQuantity(item.id, parseInt((e.target as HTMLInputElement).value) || 1)}
                        class="w-7 text-center font-mono tabular-nums font-semibold text-xs text-white bg-transparent focus-visible:outline-none"
                      />
                      <button
                        type="button"
                        onclick={() => updateQuantity(item.id, selectedItem.desired_quantity + 1)}
                        class="px-2 py-1 text-slate-400 hover:text-white hover:bg-[#182030] transition-colors cursor-pointer focus-visible:outline-none"
                        aria-label="Aumentar quantidade de {item.name}"
                        title="Aumentar"
                      >
                        <Plus class="w-3 h-3" aria-hidden="true" />
                      </button>
                    </div>
                  {:else}
                    <button
                      type="button"
                      onclick={() => toggleItem(item)}
                      class="px-2.5 py-1 rounded-md border border-[#1b2234] hover:border-emerald-500/40 bg-[#090d16] hover:bg-emerald-500/10 text-slate-300 hover:text-emerald-400 text-xs font-medium transition-colors cursor-pointer inline-flex items-center gap-1 focus-visible:outline-none"
                    >
                      <Plus class="w-3 h-3" aria-hidden="true" />
                      <span>Adicionar</span>
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}

  <!-- Sticky Bottom Summary Bar (when items are selected) -->
  {#if selectedItemsList.length > 0}
    <div class="sticky bottom-4 z-40 bg-[#0e131f]/95 backdrop-blur-md border border-[#232d44] rounded-xl p-4 shadow-xl flex flex-col sm:flex-row items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <div class="h-9 w-9 rounded-lg bg-emerald-500 text-slate-950 flex items-center justify-center font-bold text-xs font-mono tabular-nums">
          {selectedItemsList.length}
        </div>
        <div>
          <div class="text-xs text-slate-400">
            <span class="tabular-nums font-mono">{selectedItemsList.length}</span> {selectedItemsList.length === 1 ? 'item selecionado' : 'itens selecionados'} para a lista
          </div>
          <div class="text-base font-bold text-white font-mono tabular-nums">
            Total estimado: <span class="text-emerald-400">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(totalEstimated)}</span>
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2 w-full sm:w-auto">
        <button
          type="button"
          onclick={clearAllSelected}
          class="px-3 py-2 rounded-lg border border-[#1b2234] hover:border-[#27324b] text-slate-400 hover:text-slate-200 text-xs font-medium transition-colors cursor-pointer focus-visible:outline-none"
        >
          Limpar
        </button>

        <button
          type="button"
          onclick={() => { isModalOpen = true; }}
          class="flex-1 sm:flex-none px-5 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold text-xs shadow-xs transition-all flex items-center justify-center gap-2 cursor-pointer active:translate-y-px focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
        >
          <ShoppingCart class="w-3.5 h-3.5 stroke-[2.5]" aria-hidden="true" />
          <span>Gerar Lista de Compras</span>
          <ArrowRight class="w-3.5 h-3.5" aria-hidden="true" />
        </button>
      </div>
    </div>
  {/if}

  <!-- Review & Dispatch Modal -->
  {#if isModalOpen}
    <ShoppingListModal
      selectedItems={selectedItemsList}
      users={users}
      onclose={() => { isModalOpen = false; }}
      onremoveitem={removeItem}
      onupdatequantity={updateQuantity}
    />
  {/if}
</div>
