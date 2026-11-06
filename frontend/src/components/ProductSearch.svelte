<script lang="ts">
  import { Search, X, Tag, Check, ArrowRight } from '@lucide/svelte';
  import type { CanonicalProduct } from '../types';
  import { api } from '../lib/api';

  interface Props {
    selectedProduct: CanonicalProduct | null;
    onselect: (product: CanonicalProduct) => void;
  }

  let { selectedProduct, onselect }: Props = $props();

  let query = $state('');
  let searchResults = $state<CanonicalProduct[]>([]);
  let isSearching = $state(false);
  let showDropdown = $state(false);

  // Sugestões populares para consulta rápida
  const quickSuggestions = ['Arroz', 'Feijão', 'Leite', 'Café', 'Azeite', 'Sabão em Pó'];

  let searchTimeout: any;

  function handleInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    query = val;
    clearTimeout(searchTimeout);

    if (val.trim().length === 0) {
      searchResults = [];
      showDropdown = false;
      return;
    }

    isSearching = true;
    showDropdown = true;
    searchTimeout = setTimeout(async () => {
      try {
        const results = await api.searchProducts(val.trim());
        searchResults = results;
      } catch (err) {
        console.error('Erro buscando produtos:', err);
      } finally {
        isSearching = false;
      }
    }, 220);
  }

  function handleSelect(product: CanonicalProduct) {
    query = product.name;
    showDropdown = false;
    onselect(product);
  }

  function handleQuickSearch(term: string) {
    query = term;
    isSearching = true;
    showDropdown = true;
    api.searchProducts(term)
      .then((results) => {
        searchResults = results;
        if (results.length > 0) {
          handleSelect(results[0]);
        }
      })
      .catch((err) => console.error(err))
      .finally(() => {
        isSearching = false;
      });
  }

  function handleClear() {
    query = '';
    searchResults = [];
    showDropdown = false;
  }
</script>

<div class="relative w-full max-w-3xl mx-auto mb-8">
  <div class="text-center mb-5">
    <h2 class="text-xl sm:text-2xl font-bold text-white tracking-tight text-pretty">
      Comparativo de Preços por Unidade
    </h2>
    <p class="text-xs sm:text-sm text-slate-400 mt-1 max-w-xl mx-auto">
      Descubra em qual loja o item sai mais em conta por <span class="text-emerald-400 font-mono font-medium">kg</span>, <span class="text-emerald-400 font-mono font-medium">litro</span> ou <span class="text-emerald-400 font-mono font-medium">unidade</span>.
    </p>
  </div>

  <!-- Search Input Container -->
  <div class="relative">
    <label for="product-search-input" class="sr-only">Pesquisar produto no catálogo</label>
    <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-slate-400">
      <Search class="w-4 h-4 text-slate-400" aria-hidden="true" />
    </div>

    <input
      id="product-search-input"
      type="text"
      autocomplete="off"
      value={query}
      oninput={handleInput}
      onfocus={() => { if (searchResults.length > 0) showDropdown = true; }}
      placeholder="Buscar por produto, marca ou categoria (ex: Arroz, Detergente, Café)…"
      class="w-full pl-11 pr-11 py-3 bg-[#0e131f] border border-[#1b2234] focus:border-emerald-500/60 focus:bg-[#111726] rounded-xl text-slate-100 placeholder-slate-500 focus:outline-none transition-all text-sm shadow-xs"
    />

    <div class="absolute inset-y-0 right-0 pr-3.5 flex items-center gap-2">
      {#if isSearching}
        <div class="w-4 h-4 border-2 border-emerald-500/30 border-t-emerald-400 rounded-full animate-spin" aria-hidden="true"></div>
      {:else if query}
        <button
          onclick={handleClear}
          class="text-slate-400 hover:text-slate-200 transition-colors cursor-pointer p-1 rounded-md hover:bg-[#1a2233] focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
          title="Limpar busca"
          aria-label="Limpar busca"
        >
          <X class="w-3.5 h-3.5" aria-hidden="true" />
        </button>
      {/if}
    </div>
  </div>

  <!-- Autocomplete Dropdown -->
  {#if showDropdown && searchResults.length > 0}
    <div
      role="listbox"
      aria-label="Resultados de produtos"
      aria-live="polite"
      class="absolute z-50 left-0 right-0 mt-2 bg-[#0e131f] border border-[#232d44] rounded-xl shadow-xl overflow-hidden max-h-80 overflow-y-auto"
    >
      <div class="px-4 py-2.5 border-b border-[#1b2234] flex items-center justify-between text-xs text-slate-400 font-medium">
        <span>Itens encontrados no catálogo</span>
        <span class="tabular-nums font-mono text-slate-500">{searchResults.length} {searchResults.length === 1 ? 'item' : 'itens'}</span>
      </div>

      <div class="divide-y divide-[#182030]">
        {#each searchResults as product}
          <button
            role="option"
            aria-selected={selectedProduct?.id === product.id}
            onclick={() => handleSelect(product)}
            class="w-full text-left px-4 py-3 hover:bg-[#151c2c] transition-colors flex items-center justify-between group cursor-pointer focus:outline-none focus-visible:bg-[#151c2c]"
          >
            <div>
              <div class="font-medium text-sm text-slate-200 group-hover:text-emerald-400 transition-colors">
                {product.name}
              </div>
              <div class="text-xs text-slate-500 flex items-center gap-2 mt-0.5">
                <span>{product.category || 'Geral'}</span>
                <span>•</span>
                <span class="text-slate-400">Unidade base: <span class="text-slate-300 font-mono">{product.base_unit}</span></span>
              </div>
            </div>
            <span class="text-xs font-medium px-2.5 py-1 rounded-md bg-[#182030] group-hover:bg-emerald-500/10 group-hover:text-emerald-400 text-slate-400 transition-colors flex items-center gap-1">
              <span>Selecionar</span>
              <ArrowRight class="w-3 h-3" aria-hidden="true" />
            </span>
          </button>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Quick suggestions tags -->
  <div class="flex items-center gap-1.5 mt-3 flex-wrap justify-center text-xs">
    <span class="text-slate-500 text-[11px] mr-1">
      Atalhos:
    </span>
    {#each quickSuggestions as suggestion}
      <button
        onclick={() => handleQuickSearch(suggestion)}
        class="px-2.5 py-1 rounded-md bg-[#0e131f] hover:bg-[#161d2d] text-slate-400 hover:text-slate-200 border border-[#1b2234] hover:border-[#27324b] transition-all cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400"
      >
        {suggestion}
      </button>
    {/each}
  </div>

  <!-- Selected product badge indicator -->
  {#if selectedProduct}
    <div class="mt-4 flex items-center justify-between p-3 rounded-lg bg-[#0e131f] border border-emerald-500/30 text-slate-300 text-xs sm:text-sm">
      <div class="flex items-center gap-2.5">
        <span class="w-2 h-2 rounded-full bg-emerald-400 shrink-0" aria-hidden="true"></span>
        <span>
          Produto em análise: <strong class="text-white font-semibold">{selectedProduct.name}</strong> 
          <span class="text-slate-400 font-normal">({selectedProduct.category || 'Geral'} • R$/{selectedProduct.base_unit})</span>
        </span>
      </div>
      <button
        onclick={() => handleClear()}
        class="text-xs text-emerald-400 hover:text-emerald-300 transition-colors cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 rounded px-1.5 py-0.5"
      >
        Alterar
      </button>
    </div>
  {/if}
</div>
