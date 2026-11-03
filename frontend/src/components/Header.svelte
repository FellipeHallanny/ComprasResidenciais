<script lang="ts">
  import {
    ShoppingCart,
    BarChart3,
    ReceiptText,
    Users,
    Store as StoreIcon,
    PlusCircle,
    Sparkles
  } from '@lucide/svelte';

  interface Props {
    activeTab: 'dashboard' | 'purchases' | 'household' | 'stores';
    onselectTab: (tab: 'dashboard' | 'purchases' | 'household' | 'stores') => void;
    onopenPurchaseModal: () => void;
  }

  let { activeTab, onselectTab, onopenPurchaseModal }: Props = $props();
</script>

<header class="border-b border-slate-800 bg-slate-900/90 backdrop-blur sticky top-0 z-40">
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
    <!-- Brand -->
    <div class="flex items-center gap-3">
      <div class="h-10 w-10 rounded-xl bg-gradient-to-tr from-emerald-500 to-teal-400 flex items-center justify-center shadow-lg shadow-emerald-500/20 text-slate-950">
        <ShoppingCart class="w-5 h-5 stroke-[2.5]" />
      </div>
      <div>
        <div class="flex items-center gap-2">
          <span class="font-bold text-lg text-white tracking-tight">Compras Residenciais</span>
          <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Sparkles class="w-3 h-3" /> IA & pgvector
          </span>
        </div>
        <p class="text-xs text-slate-400 hidden sm:block">Controle e Inteligência de Preços Domésticos</p>
      </div>
    </div>

    <!-- Navigation Tabs -->
    <nav class="flex items-center gap-1 sm:gap-2">
      <button
        onclick={() => onselectTab('dashboard')}
        class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors {activeTab === 'dashboard' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'}"
      >
        <BarChart3 class="w-4 h-4" />
        <span>Preços & Oportunidades</span>
      </button>

      <button
        onclick={() => onselectTab('purchases')}
        class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors {activeTab === 'purchases' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'}"
      >
        <ReceiptText class="w-4 h-4" />
        <span>Compras</span>
      </button>

      <button
        onclick={() => onselectTab('household')}
        class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors {activeTab === 'household' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'}"
      >
        <Users class="w-4 h-4" />
        <span>Moradores</span>
      </button>

      <button
        onclick={() => onselectTab('stores')}
        class="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors {activeTab === 'stores' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'}"
      >
        <StoreIcon class="w-4 h-4" />
        <span>Lojas</span>
      </button>
    </nav>

    <!-- Action Button -->
    <div class="flex items-center gap-3">
      <button
        onclick={() => onopenPurchaseModal()}
        class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold text-sm shadow-md shadow-emerald-500/20 transition-all cursor-pointer hover:scale-[1.02] active:scale-[0.98]"
      >
        <PlusCircle class="w-4 h-4 stroke-[2.5]" />
        <span>Registrar Compra</span>
      </button>
    </div>
  </div>
</header>
