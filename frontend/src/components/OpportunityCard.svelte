<script lang="ts">
  import type { PriceComparisonView } from '../types';
  import { CheckCircle2, TrendingDown, Calendar, AlertCircle, ShoppingCart } from '@lucide/svelte';

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
  <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl p-6 animate-pulse mb-8" aria-hidden="true">
    <div class="h-6 bg-[#182030] rounded w-1/3 mb-4"></div>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="h-24 bg-[#182030] rounded-lg"></div>
      <div class="h-24 bg-[#182030] rounded-lg"></div>
      <div class="h-24 bg-[#182030] rounded-lg"></div>
    </div>
  </div>
{:else if comparison}
  <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl p-6 mb-8 transition-all">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6 border-b border-[#1b2234] pb-5">
      <div>
        <div class="flex items-center gap-2 flex-wrap">
          <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <CheckCircle2 class="w-3.5 h-3.5" aria-hidden="true" />
            Melhor oportunidade registrada
          </span>
          {#if comparison.max_savings_percentage > 0}
            <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium bg-amber-500/10 text-amber-400 border border-amber-500/20 tabular-nums">
              <TrendingDown class="w-3.5 h-3.5" aria-hidden="true" />
              Economia de até {comparison.max_savings_percentage.toFixed(1)}%
            </span>
          {/if}
        </div>
        <h3 class="text-xl sm:text-2xl font-bold text-white mt-2.5 text-pretty">
          {comparison.product_name}
        </h3>
        <p class="text-xs text-slate-400 mt-1">
          Base de comparação por <span class="text-slate-200 font-medium">1 {comparison.base_unit}</span>, independente de marca ou embalagem.
        </p>
      </div>

      <!-- Recommendation Period -->
      {#if comparison.best_month_period}
        <div class="bg-[#121826] border border-[#1f293d] rounded-lg px-3.5 py-2.5 flex items-center gap-3 self-start md:self-auto">
          <div class="w-8 h-8 rounded-md bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 flex items-center justify-center shrink-0">
            <Calendar class="w-4 h-4" aria-hidden="true" />
          </div>
          <div>
            <span class="text-[11px] font-medium text-slate-400 block">Melhor época de compra</span>
            <span class="text-xs font-semibold text-emerald-400">{comparison.best_month_period}</span>
          </div>
        </div>
      {/if}
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <!-- Menor Preço (Winner) -->
      <div class="bg-[#121927] border border-emerald-500/30 rounded-lg p-4 transition-all">
        <div class="text-xs font-medium text-emerald-400 mb-1">
          Menor preço normalizado
        </div>
        <div class="text-2xl sm:text-3xl font-bold text-white tracking-tight tabular-nums">
          {formattedLowest}
          <span class="text-xs font-normal text-slate-400">/{comparison.base_unit}</span>
        </div>
        <div class="mt-3 text-xs text-slate-300 flex items-center gap-1.5">
          <ShoppingCart class="w-3.5 h-3.5 text-emerald-400 shrink-0" aria-hidden="true" />
          <span class="truncate">Em: <strong class="text-white font-medium">{comparison.cheapest_store_name || 'Estabelecimento local'}</strong></span>
        </div>
        {#if formattedDate}
          <div class="text-[11px] text-slate-500 mt-0.5 font-mono">
            Registrado em {formattedDate}
          </div>
        {/if}
      </div>

      <!-- Preço Médio -->
      <div class="bg-[#101522] border border-[#1b2234] rounded-lg p-4">
        <div class="text-xs font-medium text-slate-400 mb-1">
          Média das compras
        </div>
        <div class="text-2xl sm:text-3xl font-bold text-slate-200 tracking-tight tabular-nums">
          {formattedAverage}
          <span class="text-xs font-normal text-slate-500">/{comparison.base_unit}</span>
        </div>
        <p class="text-xs text-slate-400 mt-3">
          Valor médio apurado nos estabelecimentos cadastrados.
        </p>
      </div>

      <!-- Maior Preço -->
      <div class="bg-[#101522] border border-[#1b2234] rounded-lg p-4">
        <div class="text-xs font-medium text-slate-400 mb-1">
          Maior preço registrado
        </div>
        <div class="text-2xl sm:text-3xl font-bold text-slate-300 tracking-tight tabular-nums">
          {formattedHighest}
          <span class="text-xs font-normal text-slate-500">/{comparison.base_unit}</span>
        </div>
        <p class="text-xs text-rose-400/90 mt-3 flex items-center gap-1.5">
          <AlertCircle class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
          <span>Variação de até <strong class="tabular-nums font-semibold">{comparison.max_savings_percentage.toFixed(0)}%</strong></span>
        </p>
      </div>
    </div>
  </div>
{:else}
  <div class="bg-[#0e131f] border border-[#1b2234] rounded-xl p-8 text-center text-slate-400 mb-8">
    <ShoppingCart class="w-8 h-8 text-slate-600 mx-auto mb-2.5" aria-hidden="true" />
    <h4 class="text-sm font-semibold text-slate-300">Nenhum produto selecionado</h4>
    <p class="text-xs text-slate-500 max-w-sm mx-auto mt-1">
      Utilize o campo de busca acima para pesquisar qualquer item da lista de compras da sua casa.
    </p>
  </div>
{/if}
