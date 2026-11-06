<script lang="ts">
  import type { MonthlyPriceOscillation } from '../types';
  import { Clock, CheckCircle } from '@lucide/svelte';

  interface Props {
    oscillation: MonthlyPriceOscillation | null;
    loading: boolean;
  }

  let { oscillation, loading }: Props = $props();

  const maxPrice = $derived.by(() => {
    if (!oscillation?.periods || oscillation.periods.length === 0) return 10;
    const max = Math.max(...oscillation.periods.map(p => p.average_price));
    return max > 0 ? max * 1.15 : 10;
  });

  const bestPeriod = $derived.by(() => {
    if (!oscillation?.periods || oscillation.periods.length === 0) return null;
    const valid = oscillation.periods.filter(p => p.average_price > 0);
    if (valid.length === 0) return null;
    return valid.reduce((prev, curr) => (curr.average_price < prev.average_price ? curr : prev));
  });
</script>

{#if loading}
  <div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 animate-pulse mb-8" aria-hidden="true">
    <div class="h-6 bg-slate-800 rounded w-1/4 mb-4"></div>
    <div class="h-44 bg-slate-800 rounded"></div>
  </div>
{:else if oscillation && oscillation.periods && oscillation.periods.length > 0}
  <div class="bg-slate-900 border border-slate-800 rounded-2xl p-6 shadow-sm mb-8">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 mb-6">
      <div>
        <div class="flex items-center gap-2">
          <Clock class="w-4 h-4 text-emerald-400" aria-hidden="true" />
          <h3 class="text-base font-bold text-white text-pretty">Oscilação Temporal de Preço no Mês</h3>
        </div>
        <p class="text-xs text-slate-400 mt-1">
          Identifique o melhor período do mês para comprar <strong class="text-slate-300">{oscillation.product_name}</strong> com menor preço por {oscillation.base_unit}.
        </p>
      </div>

      {#if bestPeriod}
        <div class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-xs font-semibold tabular-nums">
          <CheckCircle class="w-3.5 h-3.5" aria-hidden="true" />
          Melhor momento: {bestPeriod.period_label} ({new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(bestPeriod.average_price)}/{oscillation.base_unit})
        </div>
      {/if}
    </div>

    <!-- Chart visualization -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 pt-4 pb-2">
      {#each oscillation.periods as period}
        {@const isBest = bestPeriod?.period_label === period.period_label && period.average_price > 0}
        {@const heightPct = period.average_price > 0 ? (period.average_price / maxPrice) * 100 : 8}
        
        <div class="flex flex-col items-center group">
          <!-- Value on top of bar -->
          <div class="text-xs font-mono font-semibold mb-2 transition-transform motion-reduce:transition-none group-hover:scale-105 tabular-nums {isBest ? 'text-emerald-400 font-bold' : 'text-slate-300'}">
            {period.average_price > 0
              ? new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(period.average_price)
              : 'Sem dados'}
          </div>

          <!-- Bar Container -->
          <div class="w-full bg-slate-800/60 rounded-xl h-36 flex items-end p-2 relative overflow-hidden border {isBest ? 'border-emerald-500/50 bg-emerald-950/20 shadow-lg shadow-emerald-950/50' : 'border-slate-800'}">
            <div
              class="w-full rounded-lg transition-[height] duration-500 motion-reduce:transition-none {isBest ? 'bg-gradient-to-t from-emerald-600 to-teal-400' : 'bg-slate-700/80 group-hover:bg-slate-600'}"
              style="height: {heightPct}%"
            ></div>
          </div>

          <!-- Label below bar -->
          <div class="mt-3 text-center">
            <div class="text-xs font-semibold {isBest ? 'text-emerald-400' : 'text-slate-300'}">
              {period.period_label}
            </div>
            <div class="text-[11px] text-slate-500 tabular-nums">
              {period.total_count} {period.total_count === 1 ? 'amostra' : 'amostras'}
            </div>
          </div>
        </div>
      {/each}
    </div>
  </div>
{/if}
