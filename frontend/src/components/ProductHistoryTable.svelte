<script lang="ts">
  import type { RecentItemPurchase } from '../types';
  import { History, Store, User, Package, Calendar } from '@lucide/svelte';

  interface Props {
    purchases: RecentItemPurchase[] | undefined;
    baseUnit: string;
    productName: string;
  }

  let { purchases, baseUnit, productName }: Props = $props();

  function formatDate(iso: string) {
    try {
      return new Date(iso).toLocaleDateString('pt-BR');
    } catch {
      return iso;
    }
  }

  function formatBRL(val: number) {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(val);
  }
</script>

<div class="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-sm mb-8">
  <div class="p-5 border-b border-slate-800 flex items-center justify-between">
    <div class="flex items-center gap-2">
      <History class="w-4 h-4 text-emerald-400" aria-hidden="true" />
      <h3 class="text-base font-bold text-white text-pretty">Histórico de Compras Registradas</h3>
    </div>
    <span class="text-xs text-slate-400 tabular-nums">
      {purchases?.length ?? 0} {purchases?.length === 1 ? 'registro' : 'registros'} de {productName}
    </span>
  </div>

  {#if !purchases || purchases.length === 0}
    <div class="p-8 text-center text-slate-400 text-sm">
      Nenhuma compra individual encontrada para este produto no histórico recente.
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-left text-sm text-slate-300">
        <thead class="text-xs uppercase bg-slate-950/60 text-slate-400 border-b border-slate-800 font-semibold tracking-wider">
          <tr>
            <th class="px-5 py-3.5"><div class="flex items-center gap-1.5"><Calendar class="w-3.5 h-3.5" aria-hidden="true" /> Data</div></th>
            <th class="px-5 py-3.5"><div class="flex items-center gap-1.5"><Store class="w-3.5 h-3.5" aria-hidden="true" /> Estabelecimento</div></th>
            <th class="px-5 py-3.5"><div class="flex items-center gap-1.5"><User class="w-3.5 h-3.5" aria-hidden="true" /> Morador (Comprador)</div></th>
            <th class="px-5 py-3.5"><div class="flex items-center gap-1.5"><Package class="w-3.5 h-3.5" aria-hidden="true" /> Formato / Embalagem</div></th>
            <th class="px-5 py-3.5 text-right">Total Pago</th>
            <th class="px-5 py-3.5 text-right text-emerald-400">Preço Normalizado</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-800/60">
          {#each purchases as item}
            <tr class="hover:bg-slate-800/40 transition-colors">
              <td class="px-5 py-3.5 font-medium text-slate-300 whitespace-nowrap font-mono tabular-nums">
                {formatDate(item.purchased_at)}
              </td>
              <td class="px-5 py-3.5 font-medium text-white whitespace-nowrap">
                {item.store_name || 'Estabelecimento Local'}
              </td>
              <td class="px-5 py-3.5 whitespace-nowrap">
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-800 text-slate-300 border border-slate-700">
                  {item.buyer_name || 'Residência'}
                </span>
              </td>
              <td class="px-5 py-3.5">
                <div class="text-slate-200 font-medium">{item.raw_description}</div>
                <div class="text-xs text-slate-500 font-mono mt-0.5">{item.package_info}</div>
              </td>
              <td class="px-5 py-3.5 text-right font-medium text-slate-200 whitespace-nowrap tabular-nums">
                {formatBRL(item.total_price)}
              </td>
              <td class="px-5 py-3.5 text-right font-bold text-emerald-400 whitespace-nowrap font-mono tabular-nums">
                {formatBRL(item.normalized_price)}/{baseUnit}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
