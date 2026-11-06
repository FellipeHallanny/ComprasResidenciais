<script lang="ts">
  import type { GlobalSummaryView } from '../types';
  import { Wallet, ShoppingBag, Store, Users, ArrowUpRight } from '@lucide/svelte';

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

<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
  <!-- Card 1: Gasto Mensal -->
  <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl p-5 hover:border-[#27324b] transition-all">
    <div class="flex items-center justify-between text-slate-400 mb-3">
      <span class="text-xs font-medium text-slate-400">Total no mês</span>
      <div class="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 flex items-center justify-center">
        <Wallet class="w-4 h-4" aria-hidden="true" />
      </div>
    </div>
    {#if loading}
      <div class="h-8 bg-[#182030] animate-pulse rounded w-28 my-1" aria-hidden="true"></div>
    {:else}
      <div class="text-2xl font-bold text-white tracking-tight tabular-nums">{formattedMonthTotal}</div>
      <p class="text-xs text-slate-500 mt-1.5 flex items-center gap-1">
        <span>Despesas residenciais consolidadas</span>
      </p>
    {/if}
  </div>

  <!-- Card 2: Total de Compras -->
  <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl p-5 hover:border-[#27324b] transition-all">
    <div class="flex items-center justify-between text-slate-400 mb-3">
      <span class="text-xs font-medium text-slate-400">Compras registradas</span>
      <div class="w-8 h-8 rounded-lg bg-slate-800/60 border border-slate-700/50 text-slate-300 flex items-center justify-center">
        <ShoppingBag class="w-4 h-4" aria-hidden="true" />
      </div>
    </div>
    {#if loading}
      <div class="h-8 bg-[#182030] animate-pulse rounded w-16 my-1" aria-hidden="true"></div>
    {:else}
      <div class="text-2xl font-bold text-white tracking-tight tabular-nums">{summary?.total_purchases ?? 0}</div>
      <p class="text-xs text-slate-500 mt-1.5">Notas fiscais e registros manuais</p>
    {/if}
  </div>

  <!-- Card 3: Top Lojas -->
  <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl p-5 hover:border-[#27324b] transition-all">
    <div class="flex items-center justify-between text-slate-400 mb-3">
      <span class="text-xs font-medium text-slate-400">Principais lojas</span>
      <div class="w-8 h-8 rounded-lg bg-slate-800/60 border border-slate-700/50 text-slate-300 flex items-center justify-center">
        <Store class="w-4 h-4" aria-hidden="true" />
      </div>
    </div>
    {#if loading}
      <div class="space-y-2 mt-2" aria-hidden="true">
        <div class="h-3 bg-[#182030] animate-pulse rounded w-full"></div>
        <div class="h-3 bg-[#182030] animate-pulse rounded w-3/4"></div>
      </div>
    {:else if summary?.top_stores && summary.top_stores.length > 0}
      <div class="space-y-2 mt-1">
        {#each summary.top_stores.slice(0, 2) as store}
          <div>
            <div class="flex justify-between items-center text-xs min-w-0 mb-1">
              <span class="text-slate-300 truncate max-w-[140px] font-medium" title={store.store_name}>{store.store_name}</span>
              <span class="text-slate-400 font-mono tabular-nums">{store.percentage.toFixed(0)}%</span>
            </div>
            <div class="w-full bg-[#182030] h-1.5 rounded-full overflow-hidden">
              <div class="bg-emerald-500/70 h-full rounded-full" style="width: {Math.min(100, store.percentage)}%"></div>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <p class="text-xs text-slate-500 mt-2">Sem histórico suficiente</p>
    {/if}
  </div>

  <!-- Card 4: Gastos por Morador -->
  <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl p-5 hover:border-[#27324b] transition-all">
    <div class="flex items-center justify-between text-slate-400 mb-3">
      <span class="text-xs font-medium text-slate-400">Divisão de despesas</span>
      <div class="w-8 h-8 rounded-lg bg-slate-800/60 border border-slate-700/50 text-slate-300 flex items-center justify-center">
        <Users class="w-4 h-4" aria-hidden="true" />
      </div>
    </div>
    {#if loading}
      <div class="space-y-2 mt-2" aria-hidden="true">
        <div class="h-3 bg-[#182030] animate-pulse rounded w-full"></div>
        <div class="h-3 bg-[#182030] animate-pulse rounded w-2/3"></div>
      </div>
    {:else if userSpendingList.length > 0}
      <div class="space-y-2 mt-1">
        {#each userSpendingList.slice(0, 2) as user}
          <div class="flex justify-between items-center text-xs min-w-0">
            <span class="text-slate-300 truncate max-w-[120px] font-medium" title={user.name}>{user.name}</span>
            <span class="text-slate-200 font-mono font-medium tabular-nums">
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
