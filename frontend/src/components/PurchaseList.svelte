<script lang="ts">
  import type { Purchase, Store, User } from '../types';
  import {
    ReceiptText,
    Calendar,
    Store as StoreIcon,
    User as UserIcon,
    Sparkles,
    CheckCircle,
    Clock,
    X,
    Eye
  } from '@lucide/svelte';
  import { api } from '../lib/api';

  interface Props {
    purchases: Purchase[];
    stores: Store[];
    users: User[];
    loading: boolean;
    onrefresh: () => void;
  }

  let { purchases, stores, users, loading, onrefresh }: Props = $props();

  const purchaseList = $derived(purchases ?? []);
  const storeList = $derived(stores ?? []);
  const userList = $derived(users ?? []);

  let filterStatus = $state<'ALL' | 'CONFIRMED' | 'DRAFT'>('ALL');
  let selectedPurchase = $state<Purchase | null>(null);
  let loadingDetails = $state(false);

  const filteredList = $derived.by(() => {
    if (filterStatus === 'ALL') return purchaseList;
    return purchaseList.filter(p => p.status === filterStatus);
  });

  function getStoreName(storeId: string): string {
    const s = storeList.find(x => x.id === storeId);
    return s?.name || 'Estabelecimento';
  }

  function getUserName(userId: string): string {
    const u = userList.find(x => x.id === userId);
    return u?.name || 'Não atribuído';
  }

  function formatDate(iso: string): string {
    try {
      return new Date(iso).toLocaleDateString('pt-BR');
    } catch {
      return iso;
    }
  }

  function formatBRL(val: number): string {
    return new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(val);
  }

  async function openDetails(p: Purchase) {
    loadingDetails = true;
    try {
      const full = await api.getPurchase(p.id);
      selectedPurchase = full;
    } catch {
      selectedPurchase = p;
    } finally {
      loadingDetails = false;
    }
  }
</script>

<div class="space-y-6">
  <!-- Controls bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-900 border border-slate-800 p-4 rounded-2xl">
    <div>
      <h2 class="text-lg font-bold text-white flex items-center gap-2 text-pretty">
        <ReceiptText class="w-5 h-5 text-emerald-400" aria-hidden="true" />
        Histórico Geral de Compras
      </h2>
      <p class="text-xs text-slate-400 mt-0.5">
        Todas as notas fiscais, cupons e compras manuais registradas
      </p>
    </div>

    <!-- Filter Buttons -->
    <div class="flex items-center gap-1.5 bg-slate-950 p-1 rounded-xl border border-slate-800 text-xs font-medium" role="group" aria-label="Filtrar por status da compra">
      <button
        onclick={() => filterStatus = 'ALL'}
        class="px-3 py-1.5 rounded-lg transition-colors cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {filterStatus === 'ALL' ? 'bg-slate-800 text-emerald-400 shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
      >
        Todas (<span class="tabular-nums">{purchaseList.length}</span>)
      </button>
      <button
        onclick={() => filterStatus = 'CONFIRMED'}
        class="px-3 py-1.5 rounded-lg transition-colors cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {filterStatus === 'CONFIRMED' ? 'bg-slate-800 text-emerald-400 shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
      >
        Confirmadas
      </button>
      <button
        onclick={() => filterStatus = 'DRAFT'}
        class="px-3 py-1.5 rounded-lg transition-colors cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 {filterStatus === 'DRAFT' ? 'bg-slate-800 text-amber-400 shadow-sm' : 'text-slate-400 hover:text-slate-200'}"
      >
        Rascunhos
      </button>
    </div>
  </div>

  <!-- Purchases List / Table -->
  <div class="bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
    {#if loading}
      <div class="p-8 text-center text-slate-400 text-sm">
        <div class="w-6 h-6 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin mx-auto mb-3" aria-hidden="true"></div>
        Carregando compras…
      </div>
    {:else if filteredList.length === 0}
      <div class="p-12 text-center text-slate-400 text-sm">
        <ReceiptText class="w-10 h-10 text-slate-600 mx-auto mb-3" aria-hidden="true" />
        <p class="font-semibold text-slate-300">Nenhuma compra encontrada</p>
        <p class="text-xs text-slate-500 mt-1">
          {purchaseList.length === 0 ? 'Nenhuma compra foi registrada no sistema ainda. Clique em "Registrar Compra" no topo para começar!' : 'Nenhuma compra corresponde ao filtro selecionado.'}
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm text-slate-300">
          <thead class="text-xs uppercase bg-slate-950/70 text-slate-400 border-b border-slate-800 font-semibold tracking-wider">
            <tr>
              <th class="px-5 py-3.5"><div class="flex items-center gap-1.5"><Calendar class="w-3.5 h-3.5" aria-hidden="true" /> Data</div></th>
              <th class="px-5 py-3.5"><div class="flex items-center gap-1.5"><StoreIcon class="w-3.5 h-3.5" aria-hidden="true" /> Estabelecimento</div></th>
              <th class="px-5 py-3.5"><div class="flex items-center gap-1.5"><UserIcon class="w-3.5 h-3.5" aria-hidden="true" /> Comprador</div></th>
              <th class="px-5 py-3.5">Origem</th>
              <th class="px-5 py-3.5">Status</th>
              <th class="px-5 py-3.5 text-right">Valor Total</th>
              <th class="px-5 py-3.5 text-center">Ações</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60">
            {#each filteredList as p}
              <tr class="hover:bg-slate-800/40 transition-colors">
                <td class="px-5 py-3.5 font-medium text-slate-300 whitespace-nowrap font-mono tabular-nums">
                  {formatDate(p.purchased_at)}
                </td>
                <td class="px-5 py-3.5 font-semibold text-white whitespace-nowrap">
                  {getStoreName(p.store_id)}
                </td>
                <td class="px-5 py-3.5 whitespace-nowrap">
                  <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-800 text-slate-300 border border-slate-700">
                    {getUserName(p.user_id)}
                  </span>
                </td>
                <td class="px-5 py-3.5 whitespace-nowrap">
                  {#if p.origin === 'LLM_UPLOAD'}
                    <span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-500/10 text-purple-400 border border-purple-500/20">
                      <Sparkles class="w-3 h-3" aria-hidden="true" /> IA Cupom
                    </span>
                  {:else if p.origin === 'SEFAZ'}
                    <span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20">
                      SEFAZ
                    </span>
                  {:else}
                    <span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-800 text-slate-300 border border-slate-700">
                      Manual
                    </span>
                  {/if}
                </td>
                <td class="px-5 py-3.5 whitespace-nowrap">
                  {#if p.status === 'CONFIRMED'}
                    <span class="inline-flex items-center gap-1 text-xs text-emerald-400 font-semibold">
                      <CheckCircle class="w-3.5 h-3.5" aria-hidden="true" /> Confirmada
                    </span>
                  {:else}
                    <span class="inline-flex items-center gap-1 text-xs text-amber-400 font-semibold">
                      <Clock class="w-3.5 h-3.5" aria-hidden="true" /> Rascunho
                    </span>
                  {/if}
                </td>
                <td class="px-5 py-3.5 text-right font-extrabold text-white font-mono whitespace-nowrap tabular-nums">
                  {formatBRL(p.total_amount)}
                </td>
                <td class="px-5 py-3.5 text-center whitespace-nowrap">
                  <button
                    onclick={() => openDetails(p)}
                    class="p-1.5 rounded-lg bg-slate-800 hover:bg-emerald-500/10 text-slate-300 hover:text-emerald-400 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
                    title="Ver itens da compra"
                    aria-label="Ver itens da compra de {formatDate(p.purchased_at)}"
                  >
                    <Eye class="w-4 h-4" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>

<!-- Purchase Detail Modal -->
{#if selectedPurchase}
  <div class="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 overscroll-contain">
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="purchase-detail-title"
      class="bg-slate-900 border border-slate-700 rounded-3xl w-full max-w-2xl shadow-2xl overflow-hidden"
    >
      <div class="px-6 py-4 border-b border-slate-800 flex items-center justify-between">
        <div>
          <h3 id="purchase-detail-title" class="text-base font-bold text-white flex items-center gap-2 text-pretty">
            <ReceiptText class="w-4 h-4 text-emerald-400" aria-hidden="true" />
            Detalhes da Compra
          </h3>
          <p class="text-xs text-slate-400">
            {getStoreName(selectedPurchase.store_id)} • {formatDate(selectedPurchase.purchased_at)} • Comprador: {getUserName(selectedPurchase.user_id)}
          </p>
        </div>
        <button
          onclick={() => selectedPurchase = null}
          class="text-slate-400 hover:text-slate-200 p-2 rounded-lg hover:bg-slate-800 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
          title="Fechar"
          aria-label="Fechar detalhes da compra"
        >
          <X class="w-5 h-5" aria-hidden="true" />
        </button>
      </div>

      <div class="p-6">
        {#if selectedPurchase.items && selectedPurchase.items.length > 0}
          <div class="border border-slate-800 rounded-xl overflow-hidden max-h-80 overflow-y-auto">
            <table class="w-full text-left text-xs text-slate-300">
              <thead class="bg-slate-950 text-slate-400 uppercase font-semibold border-b border-slate-800">
                <tr>
                  <th class="px-4 py-2.5">Item</th>
                  <th class="px-4 py-2.5">Embalagem</th>
                  <th class="px-4 py-2.5 text-right">Preço Total</th>
                  <th class="px-4 py-2.5 text-right text-emerald-400">Normalizado</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                {#each selectedPurchase.items as item}
                  <tr class="hover:bg-slate-800/30">
                    <td class="px-4 py-2.5 font-medium text-white">{item.raw_description}</td>
                    <td class="px-4 py-2.5 text-slate-400 font-mono">
                      {item.quantity}x {item.package_size} {item.package_unit}
                    </td>
                    <td class="px-4 py-2.5 text-right font-medium tabular-nums">{formatBRL(item.total_price)}</td>
                    <td class="px-4 py-2.5 text-right font-bold text-emerald-400 font-mono tabular-nums">
                      {item.normalized_price ? `${formatBRL(item.normalized_price)}/${item.package_unit === 'g' ? 'kg' : (item.package_unit === 'ml' ? 'L' : item.package_unit)}` : '-'}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {:else}
          <p class="text-xs text-slate-400 text-center py-6">Nenhum item discriminado nesta compra.</p>
        {/if}

        <div class="mt-4 pt-4 border-t border-slate-800 flex justify-between items-center">
          <span class="text-xs text-slate-400 uppercase font-semibold">Valor Total da Nota:</span>
          <span class="text-xl font-bold text-white font-mono tabular-nums">{formatBRL(selectedPurchase.total_amount)}</span>
        </div>
      </div>
    </div>
  </div>
{/if}
