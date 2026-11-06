<script lang="ts">
  import type { GlobalSummaryView } from '../types';
  import { Wallet, ShoppingBag, Store, UserCheck, TrendingDown } from '@lucide/svelte';

  interface Props {
    summary: GlobalSummaryView | null;
    loading: boolean;
  }

  let { summary, loading }: Props = $props();

  const formattedMonthTotal = $derived(
    summary ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(summary.current_month_total) : 'R$ 0,00'
  );

  const userSpendingList = $derived.by(() => {
    if (!summary?.spending_by_user) return [];
    return Object.entries(summary.spending_by_user).map(([name, amount]) => ({
      name,
      amount,
      pct: summary.current_month_total > 0 ? (amount / summary.current_month_total) * 100 : 0,
    }));
  });
</script>

<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
  <!-- Card 1: Gasto Mensal -->
  <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm hover:border-slate-700 transition-colors">
    <div class="flex items-center justify-between text-slate-400 mb-2">
      <span class="text-xs font-semibold uppercase tracking-wider">Total Gasto no Mês</span>
      <div class="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
        <Wallet class="w-4 h-4" aria-hidden="true" />
      </div>
    </div>
    {#if loading}
      <div class="h-8 bg-slate-800 animate-pulse rounded w-28 my-1" aria-hidden="true"></div>
    {:else}
      <div class="text-2xl font-bold text-white tracking-tight tabular-nums">{formattedMonthTotal}</div>
      <p class="text-xs text-slate-400 mt-1 flex items-center gap-1">
        <TrendingDown class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
        Controle inteligente ativo
      </p>
    {/if}
  </div>

  <!-- Card 2: Total de Compras -->
  <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm hover:border-slate-700 transition-colors">
    <div class="flex items-center justify-between text-slate-400 mb-2">
      <span class="text-xs font-semibold uppercase tracking-wider">Notas & Compras</span>
      <div class="p-2 rounded-lg bg-cyan-500/10 text-cyan-400">
        <ShoppingBag class="w-4 h-4" aria-hidden="true" />
      </div>
    </div>
    {#if loading}
      <div class="h-8 bg-slate-800 animate-pulse rounded w-16 my-1" aria-hidden="true"></div>
    {:else}
      <div class="text-2xl font-bold text-white tracking-tight tabular-nums">{summary?.total_purchases ?? 0}</div>
      <p class="text-xs text-slate-400 mt-1">Sessões registradas</p>
    {/if}
  </div>

  <!-- Card 3: Top Lojas -->
  <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm hover:border-slate-700 transition-colors">
    <div class="flex items-center justify-between text-slate-400 mb-2">
      <span class="text-xs font-semibold uppercase tracking-wider">Top Estabelecimentos</span>
      <div class="p-2 rounded-lg bg-purple-500/10 text-purple-400">
        <Store class="w-4 h-4" aria-hidden="true" />
      </div>
    </div>
    {#if loading}
      <div class="space-y-2 mt-2" aria-hidden="true">
        <div class="h-3 bg-slate-800 animate-pulse rounded w-full"></div>
        <div class="h-3 bg-slate-800 animate-pulse rounded w-3/4"></div>
      </div>
    {:else if summary?.top_stores && summary.top_stores.length > 0}
      <div class="space-y-1.5 mt-1">
        {#each summary.top_stores.slice(0, 2) as store}
          <div class="flex justify-between items-center text-xs min-w-0">
            <span class="text-slate-300 truncate max-w-[120px] font-medium" title={store.store_name}>{store.store_name}</span>
            <span class="text-slate-400 font-mono tabular-nums">{store.percentage.toFixed(0)}%</span>
          </div>
        {/each}
      </div>
    {:else}
      <p class="text-xs text-slate-500 mt-2">Sem histórico suficiente</p>
    {/if}
  </div>

  <!-- Card 4: Gastos por Morador -->
  <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm hover:border-slate-700 transition-colors">
    <div class="flex items-center justify-between text-slate-400 mb-2">
      <span class="text-xs font-semibold uppercase tracking-wider">Divisão Moradores</span>
      <div class="p-2 rounded-lg bg-amber-500/10 text-amber-400">
        <UserCheck class="w-4 h-4" aria-hidden="true" />
      </div>
    </div>
    {#if loading}
      <div class="space-y-2 mt-2" aria-hidden="true">
        <div class="h-3 bg-slate-800 animate-pulse rounded w-full"></div>
        <div class="h-3 bg-slate-800 animate-pulse rounded w-2/3"></div>
      </div>
    {:else if userSpendingList.length > 0}
      <div class="space-y-1.5 mt-1">
        {#each userSpendingList.slice(0, 2) as user}
          <div class="flex justify-between items-center text-xs min-w-0">
            <span class="text-slate-300 truncate max-w-[110px] font-medium" title={user.name}>{user.name}</span>
            <span class="text-emerald-400 font-mono font-medium tabular-nums">
              {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 }).format(user.amount)}
            </span>
          </div>
        {/each}
      </div>
    {:else}
      <p class="text-xs text-slate-500 mt-2">Nenhum morador vinculado</p>
    {/if}
  </div>
</div>
