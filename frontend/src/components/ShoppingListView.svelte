<script lang="ts">
  import {
    ShoppingCart,
    Search,
    Check,
    CheckSquare,
    Square,
    Store,
    Sparkles,
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
    Calendar,
    Clock,
    MapPin
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
    const q = searchQuery.toLowerCase().trim();
    const list = catalog.filter(item => {
      const matchQuery = !q || item.name.toLowerCase().includes(q) || (item.category && item.category.toLowerCase().includes(q));
      const matchCat = selectedCategory === 'all' || item.category === selectedCategory;
      return matchQuery && matchCat;
    });

    return list.sort((a, b) => {
      switch (sortBy) {
        case 'alpha-asc':
          return a.name.localeCompare(b.name, 'pt-BR');
        case 'alpha-desc':
          return b.name.localeCompare(a.name, 'pt-BR');
        case 'date-desc': {
          const diff = getItemDate(b) - getItemDate(a);
          return diff !== 0 ? diff : a.name.localeCompare(b.name, 'pt-BR');
        }
        case 'date-asc': {
          const diff = getItemDate(a) - getItemDate(b);
          return diff !== 0 ? diff : a.name.localeCompare(b.name, 'pt-BR');
        }
        case 'price-asc': {
          const pa = a.lowest_price_per_unit > 0 ? a.lowest_price_per_unit : Number.MAX_VALUE;
          const pb = b.lowest_price_per_unit > 0 ? b.lowest_price_per_unit : Number.MAX_VALUE;
          return pa - pb;
        }
        case 'purchases-desc':
          return b.total_purchases - a.total_purchases;
        default:
          return 0;
      }
    });
  });

  // Array de itens selecionados para a lista
  const selectedItemsList = $derived(Object.values(selectedMap));

  // Valor total estimado dos itens selecionados
  const totalEstimated = $derived.by(() => {
    return selectedItemsList.reduce((acc, it) => acc + (it.product.lowest_price_per_unit || 0) * it.desired_quantity, 0);
  });

  // Alterna ordenação ao clicar no cabeçalho
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
  <div class="bg-slate-900 border border-slate-800 p-5 rounded-3xl flex flex-col md:flex-row md:items-center justify-between gap-4 shadow-lg shadow-black/20">
    <div class="flex items-center gap-3">
      <div class="h-12 w-12 rounded-2xl bg-gradient-to-tr from-emerald-500 to-teal-400 text-slate-950 flex items-center justify-center shadow-lg shadow-emerald-500/20 shrink-0">
        <ShoppingCart class="w-6 h-6 stroke-[2.5]" />
      </div>
      <div>
        <h2 class="text-xl font-bold text-white flex items-center gap-2">
          <span>Catálogo de Itens & Lista de Compras</span>
          <span class="inline-flex items-center gap-1 text-[11px] font-semibold px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Sparkles class="w-3 h-3" /> Otimizador de Preços
          </span>
        </h2>
        <p class="text-xs text-slate-400 mt-0.5">
          Selecione os produtos necessários. O sistema analisa o histórico de compras e organiza a lista indicando os locais mais baratos.
        </p>
      </div>
    </div>

    <!-- Quick Action / Selection Badge -->
    <div class="flex items-center gap-3 shrink-0">
      {#if selectedItemsList.length > 0}
        <button
          onclick={() => { isModalOpen = true; }}
          class="px-5 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs shadow-lg shadow-emerald-500/20 transition-all flex items-center gap-2 cursor-pointer hover:scale-[1.02] active:scale-[0.98]"
        >
          <ShoppingCart class="w-4 h-4 stroke-[2.5]" />
          <span>Gerar Lista de Compras ({selectedItemsList.length})</span>
          <ArrowRight class="w-4 h-4" />
        </button>
      {/if}
    </div>
  </div>

  <!-- Messages -->
  {#if errorMessage}
    <div class="p-4 rounded-2xl bg-rose-500/15 border border-rose-500/30 text-rose-300 text-xs flex items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <AlertCircle class="w-4 h-4 shrink-0" />
        <span>{errorMessage}</span>
      </div>
      <button onclick={loadCatalog} class="underline font-semibold hover:text-white cursor-pointer">
        Tentar novamente
      </button>
    </div>
  {/if}

  <!-- Filters, Sorting & View Mode Toolbar -->
  <div class="bg-slate-900 border border-slate-800 p-4 rounded-2xl space-y-3">
    <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-3">
      <!-- Search Input -->
      <div class="relative flex-1 max-w-md">
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Buscar no catálogo (ex: Arroz, Café, Fralda)..."
          class="w-full pl-9 pr-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none placeholder-slate-500"
        />
        <Search class="w-4 h-4 text-slate-500 absolute left-3 top-2.5 pointer-events-none" />
      </div>

      <!-- Controls Group: Sorting, View Mode & Batch Actions -->
      <div class="flex flex-wrap items-center gap-2.5">
        <!-- Ordenação Dropdown -->
        <div class="flex items-center gap-1.5 bg-slate-950 border border-slate-700 px-2.5 py-1 rounded-xl">
          <ArrowUpDown class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
          <span class="text-[11px] text-slate-400 font-semibold hidden sm:inline">Ordenar:</span>
          <select
            bind:value={sortBy}
            class="bg-transparent text-xs text-slate-200 font-medium focus:outline-none cursor-pointer pr-1"
          >
            <option value="alpha-asc" class="bg-slate-900 text-slate-200">Ordem Alfabética (A → Z)</option>
            <option value="alpha-desc" class="bg-slate-900 text-slate-200">Ordem Alfabética (Z → A)</option>
            <option value="date-desc" class="bg-slate-900 text-slate-200">Data (Mais recentes)</option>
            <option value="date-asc" class="bg-slate-900 text-slate-200">Data (Mais antigas)</option>
            <option value="price-asc" class="bg-slate-900 text-slate-200">Menor Preço</option>
            <option value="purchases-desc" class="bg-slate-900 text-slate-200">Mais Comprados</option>
          </select>
        </div>

        <!-- Alternador de Visualização (Grade / Lista) -->
        <div class="flex items-center bg-slate-950 border border-slate-700 rounded-xl p-0.5">
          <button
            type="button"
            onclick={() => { viewMode = 'grid'; }}
            class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-semibold transition cursor-pointer {viewMode === 'grid' ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : 'text-slate-400 hover:text-slate-200'}"
            title="Visualização em Grade de Cards"
          >
            <LayoutGrid class="w-3.5 h-3.5" />
            <span class="hidden sm:inline">Cards</span>
          </button>

          <button
            type="button"
            onclick={() => { viewMode = 'list'; }}
            class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-semibold transition cursor-pointer {viewMode === 'list' ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : 'text-slate-400 hover:text-slate-200'}"
            title="Visualização em Lista / Tabela"
          >
            <List class="w-3.5 h-3.5" />
            <span class="hidden sm:inline">Lista</span>
          </button>
        </div>

        <!-- Action Buttons -->
        <div class="flex items-center gap-1.5">
          <button
            onclick={selectAllFiltered}
            disabled={filteredCatalog.length === 0}
            class="px-3 py-1.5 rounded-xl border border-slate-700 hover:border-slate-600 bg-slate-950 text-slate-300 hover:text-white text-xs font-semibold transition cursor-pointer flex items-center gap-1.5 disabled:opacity-50"
            title="Marcar todos os itens visíveis"
          >
            <CheckSquare class="w-3.5 h-3.5 text-emerald-400" />
            <span class="hidden sm:inline">Marcar Todos</span>
          </button>

          {#if selectedItemsList.length > 0}
            <button
              onclick={clearAllSelected}
              class="px-3 py-1.5 rounded-xl border border-slate-800 hover:border-rose-500/40 bg-slate-950 hover:bg-rose-500/10 text-slate-400 hover:text-rose-300 text-xs font-semibold transition cursor-pointer flex items-center gap-1.5"
              title="Desmarcar todos os itens selecionados"
            >
              <Square class="w-3.5 h-3.5" />
              <span>Desmarcar ({selectedItemsList.length})</span>
            </button>
          {/if}
        </div>
      </div>
    </div>

    <!-- Category Pills -->
    {#if categories.length > 0}
      <div class="flex items-center gap-1.5 flex-wrap pt-1 text-xs">
        <span class="text-slate-500 flex items-center gap-1 mr-1 text-[11px]">
          <Tag class="w-3 h-3" /> Categorias:
        </span>
        <button
          onclick={() => { selectedCategory = 'all'; }}
          class="px-2.5 py-1 rounded-lg text-xs font-semibold transition cursor-pointer {selectedCategory === 'all' ? 'bg-emerald-500 text-slate-950 font-bold' : 'bg-slate-950 text-slate-400 hover:text-white border border-slate-800'}"
        >
          Todas ({catalog.length})
        </button>
        {#each categories as cat}
          <button
            onclick={() => { selectedCategory = cat; }}
            class="px-2.5 py-1 rounded-lg text-xs font-semibold transition cursor-pointer {selectedCategory === cat ? 'bg-emerald-500 text-slate-950 font-bold' : 'bg-slate-950 text-slate-400 hover:text-white border border-slate-800'}"
          >
            {cat}
          </button>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Catalog Items Section -->
  {#if loading}
    <div class="p-16 text-center text-slate-400 text-sm flex flex-col items-center justify-center gap-3">
      <div class="w-6 h-6 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin"></div>
      <span>Carregando itens cadastrados e histórico de preços...</span>
    </div>
  {:else if filteredCatalog.length === 0}
    <div class="bg-slate-900 border border-slate-800 rounded-2xl p-12 text-center text-slate-400 space-y-3">
      <ShoppingCart class="w-10 h-10 text-slate-600 mx-auto" />
      <h3 class="text-base font-bold text-white">Nenhum produto encontrado</h3>
      <p class="text-xs text-slate-500 max-w-md mx-auto">
        {#if searchQuery || selectedCategory !== 'all'}
          Nenhum item corresponde aos filtros selecionados. Tente alterar a busca ou filtros.
        {:else}
          Nenhum produto cadastrado no catálogo ainda. Registre compras ou adicione itens manualmente para iniciar o catálogo inteligente.
        {/if}
      </p>
    </div>
  {:else if viewMode === 'grid'}
    <!-- ============================================== -->
    <!-- VISUALIZAÇÃO 1: GRADE DE CARDS (EXISTENTE)      -->
    <!-- ============================================== -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each filteredCatalog as item}
        {@const isSelected = !!selectedMap[item.id]}
        {@const selectedItem = selectedMap[item.id]}
        {@const itemDateStr = item.last_purchased_at || item.cheapest_purchase_date || item.created_at}
        <div
          class="border rounded-2xl p-4 transition-all flex flex-col justify-between {isSelected ? 'bg-emerald-950/20 border-emerald-500/60 shadow-lg shadow-emerald-500/5 ring-1 ring-emerald-500/30' : 'bg-slate-900/80 border-slate-800 hover:border-slate-700'}"
        >
          <!-- Top Row: Checkbox + Product Name -->
          <div>
            <div class="flex items-start gap-3">
              <button
                type="button"
                onclick={() => toggleItem(item)}
                class="mt-0.5 h-5 w-5 rounded-md border flex items-center justify-center transition cursor-pointer shrink-0 {isSelected ? 'bg-emerald-500 border-emerald-500 text-slate-950' : 'border-slate-700 bg-slate-950 text-transparent hover:border-slate-500'}"
              >
                <Check class="w-3.5 h-3.5 stroke-[3]" />
              </button>

              <div class="min-w-0 flex-1">
                <button
                  type="button"
                  onclick={() => toggleItem(item)}
                  class="text-left font-bold text-white text-sm hover:text-emerald-400 transition-colors line-clamp-2 cursor-pointer"
                >
                  {item.name}
                </button>
                <div class="flex items-center gap-2 mt-1">
                  <span class="text-[10px] font-semibold px-2 py-0.5 rounded bg-slate-950 text-slate-400 border border-slate-800">
                    {item.category || 'Geral'}
                  </span>
                  <span class="text-[10px] text-slate-500 font-mono">
                    Unidade: {item.base_unit}
                  </span>
                </div>
              </div>
            </div>

            <!-- Price & Store Intelligence Info -->
            <div class="mt-3 p-2.5 rounded-xl bg-slate-950/70 border border-slate-800/80 space-y-1.5">
              {#if item.lowest_price_per_unit > 0}
                <div class="flex items-center justify-between text-xs">
                  <span class="text-slate-400">Menor Preço:</span>
                  <span class="font-bold font-mono text-emerald-400">
                    {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(item.lowest_price_per_unit)}
                    <span class="text-[10px] text-slate-400 font-normal">/{item.base_unit}</span>
                  </span>
                </div>
                {#if item.cheapest_store_name}
                  <div class="flex items-center justify-between text-[11px] text-slate-400">
                    <span class="flex items-center gap-1 text-slate-500 truncate">
                      <Store class="w-3 h-3 text-emerald-400 shrink-0" /> Local:
                    </span>
                    <span class="font-semibold text-slate-300 truncate max-w-[150px]" title={item.cheapest_store_name}>
                      {item.cheapest_store_name}
                    </span>
                  </div>
                {/if}
              {:else}
                <div class="text-[11px] text-slate-500 italic py-0.5 text-center">
                  Sem compras recentes registradas
                </div>
              {/if}

              <!-- Informação de Data -->
              {#if itemDateStr}
                <div class="flex items-center justify-between text-[10px] text-slate-500 pt-1 border-t border-slate-800/60">
                  <span class="flex items-center gap-1">
                    <Calendar class="w-3 h-3 text-slate-500" /> Último registro:
                  </span>
                  <span class="font-mono text-slate-400">{formatDate(itemDateStr)}</span>
                </div>
              {/if}
            </div>
          </div>

          <!-- Bottom: Quantity Adjuster (if selected) or Select button -->
          <div class="mt-4 pt-3 border-t border-slate-800/60 flex items-center justify-between">
            {#if isSelected && selectedItem}
              <div class="flex items-center gap-1 text-xs text-emerald-400 font-semibold">
                <Check class="w-3.5 h-3.5" />
                <span>Na Lista:</span>
              </div>

              <!-- Quantity Controls -->
              <div class="flex items-center border border-emerald-500/40 rounded-xl overflow-hidden bg-slate-950">
                <button
                  type="button"
                  onclick={() => updateQuantity(item.id, selectedItem.desired_quantity - 1)}
                  class="px-2 py-1 text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
                  title="Diminuir"
                >
                  <Minus class="w-3 h-3" />
                </button>
                <input
                  type="number"
                  min="1"
                  value={selectedItem.desired_quantity}
                  oninput={(e) => updateQuantity(item.id, parseInt((e.target as HTMLInputElement).value) || 1)}
                  class="w-10 text-center font-mono font-bold text-xs text-white bg-transparent focus:outline-none"
                />
                <button
                  type="button"
                  onclick={() => updateQuantity(item.id, selectedItem.desired_quantity + 1)}
                  class="px-2 py-1 text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
                  title="Aumentar"
                >
                  <Plus class="w-3 h-3" />
                </button>
              </div>
            {:else}
              <span class="text-[11px] text-slate-500">
                {item.total_purchases} {item.total_purchases === 1 ? 'compra registrada' : 'compras registradas'}
              </span>

              <button
                type="button"
                onclick={() => toggleItem(item)}
                class="px-3 py-1.5 rounded-xl border border-slate-700 hover:border-emerald-500/50 bg-slate-950 hover:bg-emerald-500/10 text-slate-300 hover:text-emerald-400 text-xs font-semibold transition cursor-pointer flex items-center gap-1"
              >
                <Plus class="w-3 h-3" />
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
    <div class="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-xl">
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse text-xs">
          <thead>
            <tr class="bg-slate-950/80 border-b border-slate-800 text-slate-400 text-[11px] uppercase tracking-wider font-semibold">
              <th class="py-3 px-4 w-12 text-center">Sel.</th>
              <th class="py-3 px-4 cursor-pointer hover:text-white transition" onclick={() => toggleSort('alpha')}>
                <div class="flex items-center gap-1.5">
                  <span>Produto</span>
                  {#if sortBy === 'alpha-asc'}
                    <ArrowDownAZ class="w-3.5 h-3.5 text-emerald-400" />
                  {:else if sortBy === 'alpha-desc'}
                    <ArrowUpAZ class="w-3.5 h-3.5 text-emerald-400" />
                  {/if}
                </div>
              </th>
              <th class="py-3 px-4 cursor-pointer hover:text-white transition" onclick={() => toggleSort('price')}>
                <div class="flex items-center gap-1.5">
                  <span>Menor Preço</span>
                  {#if sortBy === 'price-asc'}
                    <ArrowUpDown class="w-3 h-3 text-emerald-400" />
                  {/if}
                </div>
              </th>
              <th class="py-3 px-4">Local Mais Barato</th>
              <th class="py-3 px-4 cursor-pointer hover:text-white transition" onclick={() => toggleSort('date')}>
                <div class="flex items-center gap-1.5">
                  <span>Último Registro</span>
                  {#if sortBy.startsWith('date')}
                    <Calendar class="w-3.5 h-3.5 text-emerald-400" />
                  {/if}
                </div>
              </th>
              <th class="py-3 px-4 text-center cursor-pointer hover:text-white transition" onclick={() => toggleSort('purchases')}>
                <div class="flex items-center justify-center gap-1.5">
                  <span>Compras</span>
                  {#if sortBy === 'purchases-desc'}
                    <ArrowUpDown class="w-3 h-3 text-emerald-400" />
                  {/if}
                </div>
              </th>
              <th class="py-3 px-4 text-right min-w-[140px]">Adicionar à Lista</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            {#each filteredCatalog as item}
              {@const isSelected = !!selectedMap[item.id]}
              {@const selectedItem = selectedMap[item.id]}
              {@const itemDateStr = item.last_purchased_at || item.cheapest_purchase_date || item.created_at}
              <tr class="transition-colors hover:bg-slate-800/30 {isSelected ? 'bg-emerald-950/20' : ''}">
                <!-- Checkbox -->
                <td class="py-3 px-4 text-center">
                  <button
                    type="button"
                    onclick={() => toggleItem(item)}
                    class="h-5 w-5 rounded-md border flex items-center justify-center transition cursor-pointer mx-auto {isSelected ? 'bg-emerald-500 border-emerald-500 text-slate-950' : 'border-slate-700 bg-slate-950 text-transparent hover:border-slate-500'}"
                  >
                    <Check class="w-3.5 h-3.5 stroke-[3]" />
                  </button>
                </td>

                <!-- Nome do Produto e Categoria -->
                <td class="py-3 px-4">
                  <div class="min-w-0">
                    <button
                      type="button"
                      onclick={() => toggleItem(item)}
                      class="text-left font-semibold text-slate-100 hover:text-emerald-400 transition cursor-pointer"
                    >
                      {item.name}
                    </button>
                    <div class="flex items-center gap-2 mt-0.5">
                      <span class="text-[10px] font-medium px-1.5 py-0.2 rounded bg-slate-950 text-slate-400 border border-slate-800">
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
                    <div class="font-bold font-mono text-emerald-400 text-sm">
                      {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(item.lowest_price_per_unit)}
                      <span class="text-[10px] text-slate-400 font-normal">/{item.base_unit}</span>
                    </div>
                    {#if item.cheapest_package_info}
                      <span class="text-[10px] text-slate-500 font-mono block">({item.cheapest_package_info})</span>
                    {/if}
                  {:else}
                    <span class="text-slate-500 italic text-[11px]">Sem preço</span>
                  {/if}
                </td>

                <!-- Local Mais Barato -->
                <td class="py-3 px-4">
                  {#if item.cheapest_store_name}
                    <div class="flex items-center gap-1.5">
                      <Store class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
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
                  <div class="flex items-center gap-1.5 text-slate-400 font-mono text-[11px]">
                    <Calendar class="w-3 h-3 text-slate-500 shrink-0" />
                    <span>{formatDate(itemDateStr)}</span>
                  </div>
                </td>

                <!-- Total de Compras -->
                <td class="py-3 px-4 text-center whitespace-nowrap">
                  <span class="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-slate-950 border border-slate-800 text-slate-400">
                    {item.total_purchases}
                  </span>
                </td>

                <!-- Ações / Quantidade -->
                <td class="py-3 px-4 text-right whitespace-nowrap">
                  {#if isSelected && selectedItem}
                    <div class="inline-flex items-center border border-emerald-500/40 rounded-xl overflow-hidden bg-slate-950">
                      <button
                        type="button"
                        onclick={() => updateQuantity(item.id, selectedItem.desired_quantity - 1)}
                        class="px-2 py-1 text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
                        title="Diminuir"
                      >
                        <Minus class="w-3 h-3" />
                      </button>
                      <input
                        type="number"
                        min="1"
                        value={selectedItem.desired_quantity}
                        oninput={(e) => updateQuantity(item.id, parseInt((e.target as HTMLInputElement).value) || 1)}
                        class="w-8 text-center font-mono font-bold text-xs text-white bg-transparent focus:outline-none"
                      />
                      <button
                        type="button"
                        onclick={() => updateQuantity(item.id, selectedItem.desired_quantity + 1)}
                        class="px-2 py-1 text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
                        title="Aumentar"
                      >
                        <Plus class="w-3 h-3" />
                      </button>
                    </div>
                  {:else}
                    <button
                      type="button"
                      onclick={() => toggleItem(item)}
                      class="px-3 py-1 rounded-xl border border-slate-700 hover:border-emerald-500/50 bg-slate-950 hover:bg-emerald-500/10 text-slate-300 hover:text-emerald-400 text-xs font-semibold transition cursor-pointer inline-flex items-center gap-1"
                    >
                      <Plus class="w-3 h-3" />
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
    <div class="sticky bottom-4 z-40 bg-slate-900/95 backdrop-blur border border-emerald-500/40 rounded-2xl p-4 shadow-2xl flex flex-col sm:flex-row items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <div class="h-10 w-10 rounded-xl bg-emerald-500 text-slate-950 flex items-center justify-center font-bold text-sm">
          {selectedItemsList.length}
        </div>
        <div>
          <div class="text-xs text-slate-400">
            {selectedItemsList.length === 1 ? '1 item selecionado' : `${selectedItemsList.length} itens selecionados`} para a lista
          </div>
          <div class="text-lg font-black text-white font-mono">
            Subtotal: <span class="text-emerald-400">{new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(totalEstimated)}</span>
          </div>
        </div>
      </div>

      <div class="flex items-center gap-2 w-full sm:w-auto">
        <button
          onclick={clearAllSelected}
          class="px-3 py-2.5 rounded-xl border border-slate-700 text-slate-400 hover:text-slate-200 text-xs font-semibold transition cursor-pointer"
        >
          Limpar
        </button>

        <button
          onclick={() => { isModalOpen = true; }}
          class="flex-1 sm:flex-none px-6 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs shadow-lg shadow-emerald-500/20 transition-all flex items-center justify-center gap-2 cursor-pointer hover:scale-[1.02] active:scale-[0.98]"
        >
          <ShoppingCart class="w-4 h-4 stroke-[2.5]" />
          <span>Gerar Lista de Compras</span>
          <ArrowRight class="w-4 h-4" />
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
