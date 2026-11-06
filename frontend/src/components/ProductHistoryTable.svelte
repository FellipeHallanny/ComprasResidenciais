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

<div class="bg-[#0e131f] border border-[#1b2234] rounded-xl overflow-hidden mb-8 shadow-xs">
  <div class="px-5 py-4 border-b border-[#1b2234] flex items-center justify-between">
    <div class="flex items-center gap-2.5">
      <History class="w-4 h-4 text-emerald-400" aria-hidden="true" />
      <h3 class="text-sm sm:text-base font-semibold text-white">Histórico de Compras Registradas</h3>
    </div>
    <span class="text-xs text-slate-400 tabular-nums font-mono">
      {purchases?.length ?? 0} {purchases?.length === 1 ? 'registro' : 'registros'} de {productName}
    </span>
  </div>

  {#if !purchases || purchases.length === 0}
    <div class="p-8 text-center text-slate-400 text-xs sm:text-sm">
      Nenhuma compra individual encontrada para este produto no histórico recente.
    </div>
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs sm:text-sm text-slate-300">
        <thead class="text-xs font-medium text-slate-400 bg-[#090d16] border-b border-[#1b2234]">
          <tr>
            <th class="px-5 py-3"><div class="flex items-center gap-1.5"><Calendar class="w-3.5 h-3.5" aria-hidden="true" /> Data</div></th>
            <th class="px-5 py-3"><div class="flex items-center gap-1.5"><Store class="w-3.5 h-3.5" aria-hidden="true" /> Estabelecimento</div></th>
            <th class="px-5 py-3"><div class="flex items-center gap-1.5"><User class="w-3.5 h-3.5" aria-hidden="true" /> Comprador</div></th>
            <th class="px-5 py-3"><div class="flex items-center gap-1.5"><Package class="w-3.5 h-3.5" aria-hidden="true" /> Embalagem / Descrição</div></th>
            <th class="px-5 py-3 text-right">Total Pago</th>
            <th class="px-5 py-3 text-right text-emerald-400">Preço Normalizado</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-[#182030]">
          {#each purchases as item}
            <tr class="hover:bg-[#131926] transition-colors">
              <td class="px-5 py-3.5 font-medium text-slate-300 whitespace-nowrap font-mono tabular-nums">
                {formatDate(item.purchased_at)}
              </td>
              <td class="px-5 py-3.5 font-medium text-white whitespace-nowrap">
                {item.store_name || 'Estabelecimento local'}
              </td>
              <td class="px-5 py-3.5 whitespace-nowrap">
                <span class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium bg-[#141b29] text-slate-300 border border-[#212c42]">
                  {item.buyer_name || 'Residência'}
                </span>
              </td>
              <td class="px-5 py-3.5">
                <div class="text-slate-200 font-medium text-xs sm:text-sm">{item.raw_description}</div>
                <div class="text-xs text-slate-500 font-mono mt-0.5">{item.package_info}</div>
              </td>
              <td class="px-5 py-3.5 text-right font-medium text-slate-200 whitespace-nowrap tabular-nums">
                {formatBRL(item.total_price)}
              </td>
              <td class="px-5 py-3.5 text-right font-semibold text-emerald-400 whitespace-nowrap font-mono tabular-nums">
                {formatBRL(item.normalized_price)}/{baseUnit}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
