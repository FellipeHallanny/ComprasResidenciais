<script lang="ts">
  import { Search, Sparkles, X, Tag } from '@lucide/svelte';
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

  // Sugestões populares para busca rápida
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
    }, 250);
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
  <div class="text-center mb-4">
    <h2 class="text-xl sm:text-2xl font-bold text-white tracking-tight">
      Inteligência de Mercado e Normalização
    </h2>
    <p class="text-sm text-slate-400 mt-1">
      Compare onde itens essenciais estão mais baratos em <span class="text-emerald-400 font-mono font-medium">R$/kg</span>, <span class="text-emerald-400 font-mono font-medium">R$/litro</span> ou <span class="text-emerald-400 font-mono font-medium">R$/unidade</span>.
    </p>
  </div>

  <!-- Search Input Container -->
  <div class="relative">
    <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none text-slate-400">
      <Search class="w-5 h-5 text-emerald-400" />
    </div>

    <input
      type="text"
      value={query}
      oninput={handleInput}
      onfocus={() => { if (searchResults.length > 0) showDropdown = true; }}
      placeholder="Pesquisar produto canônico (ex: Arroz, Detergente, Leite Integral)..."
      class="w-full pl-12 pr-12 py-3.5 bg-slate-900 border border-slate-700/80 rounded-2xl text-slate-100 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 transition-all text-base shadow-lg shadow-black/20"
    />

    <div class="absolute inset-y-0 right-0 pr-4 flex items-center gap-2">
      {#if isSearching}
        <div class="w-4 h-4 border-2 border-emerald-500/30 border-t-emerald-400 rounded-full animate-spin"></div>
      {:else if query}
        <button
          onclick={handleClear}
          class="text-slate-400 hover:text-slate-200 transition cursor-pointer"
          title="Limpar busca"
        >
          <X class="w-4 h-4" />
        </button>
      {/if}
      <span class="text-[10px] uppercase font-semibold px-2 py-0.5 rounded bg-slate-800 text-slate-400 border border-slate-700 hidden sm:inline-block">
        pgvector
      </span>
    </div>
  </div>

  <!-- Autocomplete Dropdown -->
  {#if showDropdown && searchResults.length > 0}
    <div class="absolute z-50 left-0 right-0 mt-2 bg-slate-900 border border-slate-700 rounded-xl shadow-2xl overflow-hidden max-h-80 overflow-y-auto">
      <div class="p-2 border-b border-slate-800 flex items-center justify-between text-xs text-slate-400 font-medium">
        <span class="flex items-center gap-1.5 text-emerald-400">
          <Sparkles class="w-3.5 h-3.5" /> Produtos Encontrados via Busca Semântica
        </span>
        <span>{searchResults.length} {searchResults.length === 1 ? 'resultado' : 'resultados'}</span>
      </div>

      <div class="divide-y divide-slate-800/60">
        {#each searchResults as product}
          <button
            onclick={() => handleSelect(product)}
            class="w-full text-left px-4 py-3 hover:bg-slate-800/70 transition flex items-center justify-between group cursor-pointer"
          >
            <div>
              <div class="font-medium text-slate-200 group-hover:text-emerald-400 transition-colors">
                {product.name}
              </div>
              <div class="text-xs text-slate-500 flex items-center gap-2 mt-0.5">
                <span>{product.category || 'Geral'}</span>
                <span>•</span>
                <span class="text-slate-400">Unidade base: <strong class="text-slate-300 font-mono">{product.base_unit}</strong></span>
              </div>
            </div>
            <span class="text-xs font-semibold px-2.5 py-1 rounded-full bg-slate-800 group-hover:bg-emerald-500/10 group-hover:text-emerald-400 text-slate-400 transition-colors">
              Selecionar
            </span>
          </button>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Quick suggestions tags -->
  <div class="flex items-center gap-2 mt-3 flex-wrap justify-center text-xs">
    <span class="text-slate-500 flex items-center gap-1">
      <Tag class="w-3 h-3" /> Sugestões:
    </span>
    {#each quickSuggestions as suggestion}
      <button
        onclick={() => handleQuickSearch(suggestion)}
        class="px-2.5 py-1 rounded-lg bg-slate-800/70 hover:bg-slate-800 text-slate-400 hover:text-emerald-400 border border-slate-700/60 transition cursor-pointer"
      >
        {suggestion}
      </button>
    {/each}
  </div>

  <!-- Selected product badge indicator -->
  {#if selectedProduct}
    <div class="mt-4 flex items-center justify-between p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-xs sm:text-sm">
      <div class="flex items-center gap-2">
        <Sparkles class="w-4 h-4 text-emerald-400 shrink-0" />
        <span>
          Produto Ativo: <strong class="text-white font-semibold">{selectedProduct.name}</strong> 
          <span class="text-emerald-400/80">({selectedProduct.category || 'Geral'} • R$/{selectedProduct.base_unit})</span>
        </span>
      </div>
      <button
        onclick={() => handleClear()}
        class="text-xs text-emerald-400 hover:underline cursor-pointer"
      >
        Trocar produto
      </button>
    </div>
  {/if}
</div>
