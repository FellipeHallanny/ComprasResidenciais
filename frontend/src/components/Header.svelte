<script lang="ts">
  import {
    ShoppingCart,
    BarChart3,
    ReceiptText,
    Users,
    Store as StoreIcon,
    PlusCircle,
    Sparkles,
    RotateCcw,
    LogOut,
    Shield,
    User as UserIcon,
    ListChecks
  } from '@lucide/svelte';
  import { auth } from '../lib/auth.svelte';

  interface Props {
    activeTab: 'dashboard' | 'purchases' | 'shopping-list' | 'household' | 'stores';
    onselectTab: (tab: 'dashboard' | 'purchases' | 'shopping-list' | 'household' | 'stores') => void;
    onopenPurchaseModal: () => void;
    onresetDatabase: () => void;
  }

  let { activeTab, onselectTab, onopenPurchaseModal, onresetDatabase }: Props = $props();
</script>

<header class="border-b border-slate-800 bg-slate-900/95 backdrop-blur sticky top-0 z-40">
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="h-16 flex items-center justify-between gap-3">
      <!-- Left: Brand -->
      <div class="flex items-center gap-2.5 shrink-0">
        <div class="h-9 w-9 rounded-xl bg-gradient-to-tr from-emerald-500 to-teal-400 flex items-center justify-center shadow-md shadow-emerald-500/20 text-slate-950 shrink-0">
          <ShoppingCart class="w-5 h-5 stroke-[2.5]" />
        </div>
        <div class="flex items-center gap-2">
          <span class="font-bold text-sm sm:text-base text-white tracking-tight">Compras Residenciais</span>
          <span class="hidden 2xl:inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Sparkles class="w-3 h-3" /> IA
          </span>
        </div>
      </div>

      <!-- Center: Navigation Tabs (Desktop md+) -->
      <nav class="hidden md:flex items-center bg-slate-950/70 border border-slate-800/80 p-1 rounded-xl">
        <button
          onclick={() => onselectTab('dashboard')}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer {activeTab === 'dashboard' ? 'bg-slate-800 text-emerald-400 shadow-sm border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'}"
        >
          <BarChart3 class="w-3.5 h-3.5" />
          <span>Preços</span>
        </button>

        <button
          onclick={() => onselectTab('purchases')}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer {activeTab === 'purchases' ? 'bg-slate-800 text-emerald-400 shadow-sm border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'}"
        >
          <ReceiptText class="w-3.5 h-3.5" />
          <span>Compras</span>
        </button>

        <button
          onclick={() => onselectTab('shopping-list')}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer {activeTab === 'shopping-list' ? 'bg-slate-800 text-emerald-400 shadow-sm border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'}"
        >
          <ListChecks class="w-3.5 h-3.5 text-emerald-400" />
          <span>Lista de Compras</span>
        </button>

        <button
          onclick={() => onselectTab('household')}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer {activeTab === 'household' ? 'bg-slate-800 text-emerald-400 shadow-sm border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'}"
        >
          <Users class="w-3.5 h-3.5" />
          <span>Moradores</span>
        </button>

        <button
          onclick={() => onselectTab('stores')}
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer {activeTab === 'stores' ? 'bg-slate-800 text-emerald-400 shadow-sm border border-slate-700' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'}"
        >
          <StoreIcon class="w-3.5 h-3.5" />
          <span>Lojas</span>
        </button>
      </nav>

      <!-- Right: User Info & Actions -->
      <div class="flex items-center gap-2 sm:gap-2.5 shrink-0">
        {#if auth.currentUser}
          <!-- User Profile Badge -->
          <div class="flex items-center gap-2 px-2.5 py-1 rounded-xl bg-slate-950/80 border border-slate-800">
            <div class="w-6 h-6 rounded-lg {auth.isAdmin ? 'bg-amber-500/15 text-amber-400 border border-amber-500/20' : 'bg-slate-800 text-slate-300'} flex items-center justify-center font-bold text-xs shrink-0">
              {#if auth.isAdmin}
                <Shield class="w-3 h-3" />
              {:else}
                <UserIcon class="w-3 h-3" />
              {/if}
            </div>
            <div class="text-left hidden sm:block">
              <div class="text-xs font-semibold text-white leading-none max-w-[120px] truncate" title={auth.currentUser.name}>
                {auth.currentUser.name}
              </div>
              <div class="text-[9px] {auth.isAdmin ? 'text-amber-400 font-bold' : 'text-slate-400'} leading-none mt-0.5">
                {auth.isAdmin ? 'Administrador' : 'Morador'}
              </div>
            </div>
          </div>
        {/if}

        {#if auth.isAdmin}
          <button
            onclick={onresetDatabase}
            class="hidden lg:inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-xl border border-slate-800 hover:border-rose-500/40 bg-slate-950 hover:bg-rose-500/10 text-slate-400 hover:text-rose-300 text-xs font-semibold transition cursor-pointer"
            title="Zerar Banco de Dados (Exclusivo Administrador)"
          >
            <RotateCcw class="w-3.5 h-3.5" />
            <span class="hidden xl:inline">Zerar Banco</span>
          </button>
        {/if}

        <button
          onclick={() => onopenPurchaseModal()}
          class="inline-flex items-center gap-1.5 px-3 py-1.5 sm:py-2 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs shadow-md shadow-emerald-500/20 transition-all cursor-pointer hover:scale-[1.02] active:scale-[0.98]"
        >
          <PlusCircle class="w-3.5 h-3.5 stroke-[2.5]" />
          <span>Nova Compra</span>
        </button>

        <!-- Logout Button -->
        <button
          onclick={() => auth.logout()}
          class="p-2 rounded-xl border border-slate-800 bg-slate-950 hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 hover:border-rose-500/30 transition cursor-pointer"
          title="Encerrar sessão"
        >
          <LogOut class="w-3.5 h-3.5" />
        </button>
      </div>
    </div>

    <!-- Mobile Navigation Bar (< md) -->
    <div class="md:hidden pb-2.5 pt-0.5">
      <nav class="grid grid-cols-5 gap-1 p-1 bg-slate-950/80 border border-slate-800 rounded-xl">
        <button
          onclick={() => onselectTab('dashboard')}
          class="flex flex-col items-center justify-center py-1.5 px-1 rounded-lg text-[10px] font-semibold transition-all cursor-pointer {activeTab === 'dashboard' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200'}"
        >
          <BarChart3 class="w-3.5 h-3.5 mb-0.5" />
          <span>Preços</span>
        </button>

        <button
          onclick={() => onselectTab('purchases')}
          class="flex flex-col items-center justify-center py-1.5 px-1 rounded-lg text-[10px] font-semibold transition-all cursor-pointer {activeTab === 'purchases' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200'}"
        >
          <ReceiptText class="w-3.5 h-3.5 mb-0.5" />
          <span>Compras</span>
        </button>

        <button
          onclick={() => onselectTab('shopping-list')}
          class="flex flex-col items-center justify-center py-1.5 px-1 rounded-lg text-[10px] font-semibold transition-all cursor-pointer {activeTab === 'shopping-list' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200'}"
        >
          <ListChecks class="w-3.5 h-3.5 mb-0.5 text-emerald-400" />
          <span>Lista</span>
        </button>

        <button
          onclick={() => onselectTab('household')}
          class="flex flex-col items-center justify-center py-1.5 px-1 rounded-lg text-[10px] font-semibold transition-all cursor-pointer {activeTab === 'household' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200'}"
        >
          <Users class="w-3.5 h-3.5 mb-0.5" />
          <span>Moradores</span>
        </button>

        <button
          onclick={() => onselectTab('stores')}
          class="flex flex-col items-center justify-center py-1.5 px-1 rounded-lg text-[10px] font-semibold transition-all cursor-pointer {activeTab === 'stores' ? 'bg-slate-800 text-emerald-400 border border-slate-700' : 'text-slate-400 hover:text-slate-200'}"
        >
          <StoreIcon class="w-3.5 h-3.5 mb-0.5" />
          <span>Lojas</span>
        </button>
      </nav>
    </div>
  </div>
</header>
