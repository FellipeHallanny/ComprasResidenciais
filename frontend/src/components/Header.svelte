<script lang="ts">
  import {
    ShoppingCart,
    BarChart3,
    ReceiptText,
    Users,
    Store as StoreIcon,
    Plus,
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

<header class="border-b border-[#1b2234] bg-[#0c101a]/90 backdrop-blur-md sticky top-0 z-40 transition-colors">
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
    <div class="h-16 flex items-center justify-between gap-4">
      <!-- Left: Brand -->
      <div class="flex items-center gap-3 shrink-0">
        <div class="h-9 w-9 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 shrink-0 shadow-sm">
          <ShoppingCart class="w-4 h-4 stroke-[2.2]" aria-hidden="true" />
        </div>
        <div class="flex flex-col">
          <span class="font-semibold text-sm sm:text-base text-white tracking-tight leading-tight">Compras Residenciais</span>
          <span class="text-[11px] text-slate-400 leading-none mt-0.5">Gestão & Preços</span>
        </div>
      </div>

      <!-- Center: Navigation Tabs (Desktop md+) -->
      <nav class="hidden md:flex items-center" aria-label="Navegação principal">
        <div role="tablist" class="flex items-center bg-[#090d16] border border-[#1b2234] p-1 rounded-lg">
          <button
            role="tab"
            aria-selected={activeTab === 'dashboard'}
            onclick={() => onselectTab('dashboard')}
            class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 {activeTab === 'dashboard' ? 'bg-[#182030] text-emerald-400 shadow-sm border border-[#26334d]' : 'text-slate-400 hover:text-slate-200 hover:bg-[#121824]'}"
          >
            <BarChart3 class="w-3.5 h-3.5" aria-hidden="true" />
            <span>Preços</span>
          </button>

          <button
            role="tab"
            aria-selected={activeTab === 'purchases'}
            onclick={() => onselectTab('purchases')}
            class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 {activeTab === 'purchases' ? 'bg-[#182030] text-emerald-400 shadow-sm border border-[#26334d]' : 'text-slate-400 hover:text-slate-200 hover:bg-[#121824]'}"
          >
            <ReceiptText class="w-3.5 h-3.5" aria-hidden="true" />
            <span>Compras</span>
          </button>

          <button
            role="tab"
            aria-selected={activeTab === 'shopping-list'}
            onclick={() => onselectTab('shopping-list')}
            class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 {activeTab === 'shopping-list' ? 'bg-[#182030] text-emerald-400 shadow-sm border border-[#26334d]' : 'text-slate-400 hover:text-slate-200 hover:bg-[#121824]'}"
          >
            <ListChecks class="w-3.5 h-3.5" aria-hidden="true" />
            <span>Lista de Compras</span>
          </button>

          <button
            role="tab"
            aria-selected={activeTab === 'household'}
            onclick={() => onselectTab('household')}
            class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 {activeTab === 'household' ? 'bg-[#182030] text-emerald-400 shadow-sm border border-[#26334d]' : 'text-slate-400 hover:text-slate-200 hover:bg-[#121824]'}"
          >
            <Users class="w-3.5 h-3.5" aria-hidden="true" />
            <span>Moradores</span>
          </button>

          <button
            role="tab"
            aria-selected={activeTab === 'stores'}
            onclick={() => onselectTab('stores')}
            class="flex items-center gap-2 px-3 py-1.5 rounded-md text-xs font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 {activeTab === 'stores' ? 'bg-[#182030] text-emerald-400 shadow-sm border border-[#26334d]' : 'text-slate-400 hover:text-slate-200 hover:bg-[#121824]'}"
          >
            <StoreIcon class="w-3.5 h-3.5" aria-hidden="true" />
            <span>Lojas</span>
          </button>
        </div>
      </nav>

      <!-- Right: User Info & Actions -->
      <div class="flex items-center gap-2 sm:gap-3 shrink-0">
        {#if auth.currentUser}
          <!-- User Profile Badge -->
          <div class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg bg-[#0e131f] border border-[#1b2234] min-w-0">
            <div class="w-5 h-5 rounded {auth.isAdmin ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20' : 'bg-[#1b2234] text-slate-300'} flex items-center justify-center text-xs shrink-0">
              {#if auth.isAdmin}
                <Shield class="w-3 h-3" aria-hidden="true" />
              {:else}
                <UserIcon class="w-3 h-3" aria-hidden="true" />
              {/if}
            </div>
            <div class="text-left hidden sm:block min-w-0">
              <div class="text-xs font-medium text-slate-200 leading-none max-w-[120px] truncate" title={auth.currentUser.name}>
                {auth.currentUser.name}
              </div>
              <div class="text-[10px] {auth.isAdmin ? 'text-amber-400' : 'text-slate-500'} leading-none mt-1">
                {auth.isAdmin ? 'Administrador' : 'Morador'}
              </div>
            </div>
          </div>
        {/if}

        {#if auth.isAdmin}
          <button
            onclick={onresetDatabase}
            class="hidden lg:inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-[#1b2234] hover:border-rose-500/40 bg-[#0e131f] hover:bg-rose-500/10 text-slate-400 hover:text-rose-300 text-xs font-medium transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400"
            title="Reiniciar banco de dados (exclusivo para administradores)"
            aria-label="Reiniciar banco de dados"
          >
            <RotateCcw class="w-3.5 h-3.5" aria-hidden="true" />
            <span class="hidden xl:inline">Zerar Dados</span>
          </button>
        {/if}

        <button
          onclick={() => onopenPurchaseModal()}
          class="inline-flex items-center gap-1.5 px-3 py-1.5 sm:py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 active:translate-y-px text-slate-950 font-semibold text-xs transition-all cursor-pointer shadow-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
        >
          <Plus class="w-3.5 h-3.5 stroke-[2.5]" aria-hidden="true" />
          <span>Nova Compra</span>
        </button>

        <!-- Logout Button -->
        <button
          onclick={() => auth.logout()}
          class="p-2 rounded-lg border border-[#1b2234] bg-[#0e131f] hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 hover:border-rose-500/30 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400"
          title="Encerrar sessão"
          aria-label="Encerrar sessão"
        >
          <LogOut class="w-3.5 h-3.5" aria-hidden="true" />
        </button>
      </div>
    </div>

    <!-- Mobile Navigation Bar (< md) -->
    <div class="md:hidden pb-3 pt-1">
      <nav aria-label="Navegação móvel">
        <div role="tablist" class="grid grid-cols-5 gap-1 p-1 bg-[#090d16] border border-[#1b2234] rounded-lg">
          <button
            role="tab"
            aria-selected={activeTab === 'dashboard'}
            onclick={() => onselectTab('dashboard')}
            class="flex flex-col items-center justify-center py-1.5 px-1 rounded-md text-[10px] font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {activeTab === 'dashboard' ? 'bg-[#182030] text-emerald-400 border border-[#26334d]' : 'text-slate-400 hover:text-slate-200'}"
          >
            <BarChart3 class="w-3.5 h-3.5 mb-1" aria-hidden="true" />
            <span>Preços</span>
          </button>

          <button
            role="tab"
            aria-selected={activeTab === 'purchases'}
            onclick={() => onselectTab('purchases')}
            class="flex flex-col items-center justify-center py-1.5 px-1 rounded-md text-[10px] font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {activeTab === 'purchases' ? 'bg-[#182030] text-emerald-400 border border-[#26334d]' : 'text-slate-400 hover:text-slate-200'}"
          >
            <ReceiptText class="w-3.5 h-3.5 mb-1" aria-hidden="true" />
            <span>Compras</span>
          </button>

          <button
            role="tab"
            aria-selected={activeTab === 'shopping-list'}
            onclick={() => onselectTab('shopping-list')}
            class="flex flex-col items-center justify-center py-1.5 px-1 rounded-md text-[10px] font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {activeTab === 'shopping-list' ? 'bg-[#182030] text-emerald-400 border border-[#26334d]' : 'text-slate-400 hover:text-slate-200'}"
          >
            <ListChecks class="w-3.5 h-3.5 mb-1" aria-hidden="true" />
            <span>Lista</span>
          </button>

          <button
            role="tab"
            aria-selected={activeTab === 'household'}
            onclick={() => onselectTab('household')}
            class="flex flex-col items-center justify-center py-1.5 px-1 rounded-md text-[10px] font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {activeTab === 'household' ? 'bg-[#182030] text-emerald-400 border border-[#26334d]' : 'text-slate-400 hover:text-slate-200'}"
          >
            <Users class="w-3.5 h-3.5 mb-1" aria-hidden="true" />
            <span>Moradores</span>
          </button>

          <button
            role="tab"
            aria-selected={activeTab === 'stores'}
            onclick={() => onselectTab('stores')}
            class="flex flex-col items-center justify-center py-1.5 px-1 rounded-md text-[10px] font-medium transition-all cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {activeTab === 'stores' ? 'bg-[#182030] text-emerald-400 border border-[#26334d]' : 'text-slate-400 hover:text-slate-200'}"
          >
            <StoreIcon class="w-3.5 h-3.5 mb-1" aria-hidden="true" />
            <span>Lojas</span>
          </button>
        </div>
      </nav>
    </div>
  </div>
</header>
