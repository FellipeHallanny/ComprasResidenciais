<script lang="ts">
  import type { PriceComparisonView } from '../types';
  import { Trophy, TrendingDown, Calendar, AlertCircle, ShoppingCart } from '@lucide/svelte';

  interface Props {
    comparison: PriceComparisonView | null;
    loading: boolean;
  }

  let { comparison, loading }: Props = $props();

  const formattedLowest = $derived(
    comparison ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(comparison.lowest_price_per_unit) : 'R$ 0,00'
  );

  const formattedHighest = $derived(
    comparison ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(comparison.highest_price_per_unit) : 'R$ 0,00'
  );

  const formattedAverage = $derived(
    comparison ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(comparison.average_price_per_unit) : 'R$ 0,00'
  );

  const formattedDate = $derived.by(() => {
    if (!comparison?.cheapest_purchase_date) return '';
    try {
      const d = new Date(comparison.cheapest_purchase_date);
      return d.toLocaleDateString('pt-BR');
    } catch {
      return '';
    }
  });
</script>

{#if loading}
  <div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 animate-pulse mb-8">
    <div class="h-6 bg-slate-800 rounded w-1/3 mb-4"></div>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="h-20 bg-slate-800 rounded"></div>
      <div class="h-20 bg-slate-800 rounded"></div>
      <div class="h-20 bg-slate-800 rounded"></div>
    </div>
  </div>
{:else if comparison}
  <div class="bg-gradient-to-b from-slate-900 to-slate-950 border border-emerald-500/30 rounded-2xl p-6 shadow-xl shadow-emerald-950/20 mb-8 relative overflow-hidden">
    <!-- Background highlight glow -->
    <div class="absolute -top-24 -right-24 w-60 h-60 bg-emerald-500/10 rounded-full blur-3xl pointer-events-none"></div>

    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6 border-b border-slate-800 pb-5">
      <div>
        <div class="flex items-center gap-2">
          <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
            <Trophy class="w-3.5 h-3.5" /> Melhor Oportunidade Identificada
          </span>
          {#if comparison.max_savings_percentage > 0}
            <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-bold bg-amber-500/15 text-amber-400 border border-amber-500/30">
              <TrendingDown class="w-3.5 h-3.5" /> -{comparison.max_savings_percentage.toFixed(1)}% de economia
            </span>
          {/if}
        </div>
        <h3 class="text-2xl font-bold text-white mt-2">
          {comparison.product_name}
        </h3>
        <p class="text-xs text-slate-400 mt-0.5">
          Normalizado por <strong class="text-slate-300 font-mono">1 {comparison.base_unit}</strong> (independente de tamanho do pacote ou marca)
        </p>
      </div>

      <!-- Recommendation Period -->
      {#if comparison.best_month_period}
        <div class="bg-slate-800/80 border border-slate-700/60 rounded-xl px-4 py-2.5 flex items-center gap-3">
          <div class="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
            <Calendar class="w-4 h-4" />
          </div>
          <div>
            <span class="text-[11px] uppercase font-semibold text-slate-400">Melhor Época do Mês</span>
            <div class="text-sm font-bold text-emerald-400">{comparison.best_month_period}</div>
          </div>
        </div>
      {/if}
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <!-- Menor Preço (Winner) -->
      <div class="bg-slate-800/50 border border-emerald-500/40 rounded-xl p-4 relative">
        <div class="text-xs font-semibold text-emerald-400 uppercase tracking-wider mb-1">
          Menor Preço Normalizado
        </div>
        <div class="text-3xl font-extrabold text-white tracking-tight">
          {formattedLowest}
          <span class="text-sm font-normal text-slate-400">/{comparison.base_unit}</span>
        </div>
        <div class="mt-2 text-xs text-slate-300 flex items-center gap-1.5">
          <ShoppingCart class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
          <span>Vendido no: <strong class="text-white">{comparison.cheapest_store_name || 'Estabelecimento Local'}</strong></span>
        </div>
        {#if formattedDate}
          <div class="text-[11px] text-slate-500 mt-0.5">
            Registrado em {formattedDate}
          </div>
        {/if}
      </div>

      <!-- Preço Médio -->
      <div class="bg-slate-800/30 border border-slate-800 rounded-xl p-4">
        <div class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">
          Média Ponderada
        </div>
        <div class="text-2xl font-bold text-slate-200 tracking-tight">
          {formattedAverage}
          <span class="text-sm font-normal text-slate-500">/{comparison.base_unit}</span>
        </div>
        <p class="text-xs text-slate-400 mt-2">
          Preço praticado em média entre os estabelecimentos cadastrados.
        </p>
      </div>

      <!-- Maior Preço -->
      <div class="bg-slate-800/30 border border-slate-800 rounded-xl p-4">
        <div class="text-xs font-semibold text-rose-400/90 uppercase tracking-wider mb-1">
          Maior Preço Registrado
        </div>
        <div class="text-2xl font-bold text-slate-200 tracking-tight">
          {formattedHighest}
          <span class="text-sm font-normal text-slate-500">/{comparison.base_unit}</span>
        </div>
        <p class="text-xs text-rose-400/80 mt-2 flex items-center gap-1">
          <AlertCircle class="w-3.5 h-3.5 shrink-0" />
          Diferença de até {comparison.max_savings_percentage.toFixed(0)}% de ágio
        </p>
      </div>
    </div>
  </div>
{:else}
  <div class="bg-slate-900 border border-slate-800/80 rounded-2xl p-8 text-center text-slate-400 mb-8">
    <ShoppingCart class="w-10 h-10 text-slate-600 mx-auto mb-3" />
    <h4 class="text-base font-semibold text-slate-300">Nenhum produto selecionado</h4>
    <p class="text-xs text-slate-500 max-w-md mx-auto mt-1">
      Utilize o campo de busca acima para pesquisar qualquer item da lista de compras da sua residência.
    </p>
  </div>
{/if}
