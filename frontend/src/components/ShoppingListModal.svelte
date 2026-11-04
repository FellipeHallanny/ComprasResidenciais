<script lang="ts">
  import {
    X,
    Send,
    Store,
    Trash2,
    Check,
    AlertCircle,
    Copy,
    Share2,
    Mail,
    Phone,
    ShoppingCart,
    ExternalLink,
    MapPin,
    Navigation
  } from '@lucide/svelte';
  import type { User, SelectedShoppingItem } from '../types';
  import { api } from '../lib/api';
  import { auth } from '../lib/auth.svelte';

  interface Props {
    selectedItems: SelectedShoppingItem[];
    users: User[];
    onclose: () => void;
    onremoveitem: (productId: string) => void;
    onupdatequantity: (productId: string, qty: number) => void;
  }

  let { selectedItems, users, onclose, onremoveitem, onupdatequantity }: Props = $props();

  const userList = $derived(users ?? []);

  let selectedUserId = $state('');
  let dispatchChannel = $state<'whatsapp' | 'email'>('whatsapp');
  let customPhone = $state('');
  let customEmail = $state('');
  let isSending = $state(false);
  let feedbackMessage = $state('');
  let errorMessage = $state('');
  let copied = $state(false);

  // Quando seleciona um morador, sincroniza os campos de telefone e e-mail
  function handleSelectUser(userId: string) {
    selectedUserId = userId;
    const found = userList.find(u => u.id === userId);
    if (found) {
      customPhone = found.phone || '';
      customEmail = found.email || '';
    } else {
      customPhone = '';
      customEmail = '';
    }
  }

  // Pré-seleciona automaticamente o morador logado ou o primeiro morador da casa
  $effect(() => {
    if (!selectedUserId && userList.length > 0) {
      const current = userList.find(u => u.id === auth.currentUser?.id) || userList[0];
      if (current) {
        handleSelectUser(current.id);
      }
    }
  });

  interface StoreGroup {
    storeName: string;
    storeAddress?: string;
    items: SelectedShoppingItem[];
    subtotal: number;
  }

  // Agrupamento automático dos itens por estabelecimento onde estão mais baratos
  const groupedByStore = $derived.by(() => {
    const groups: Record<string, StoreGroup> = {};
    for (const it of selectedItems) {
      const store = it.product.cheapest_store_name || 'Sem Histórico (Qualquer Loja)';
      if (!groups[store]) {
        groups[store] = {
          storeName: store,
          storeAddress: it.product.cheapest_store_address,
          items: [],
          subtotal: 0
        };
      } else if (!groups[store].storeAddress && it.product.cheapest_store_address) {
        groups[store].storeAddress = it.product.cheapest_store_address;
      }
      groups[store].items.push(it);
      const unitPrice = it.product.lowest_price_per_unit || 0;
      groups[store].subtotal += unitPrice * it.desired_quantity;
    }
    return groups;
  });

  const overallTotal = $derived.by(() => {
    return selectedItems.reduce((acc, it) => acc + (it.product.lowest_price_per_unit || 0) * it.desired_quantity, 0);
  });

  // Retorna link que abre o destino no GPS preferido (Google Maps, Waze, Apple Maps)
  function getGpsUrl(storeName: string, storeAddress?: string): string {
    const query = storeAddress && storeAddress.trim()
      ? `${storeName}, ${storeAddress.trim()}`
      : storeName;
    return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(query)}`;
  }

  // Gera texto estruturado da lista para WhatsApp ou E-mail, incluindo endereço e link do GPS
  function generateFormattedText(): string {
    let text = `🛒 LISTA DE COMPRAS INTELIGENTE\n`;
    text += `📅 ${new Date().toLocaleDateString('pt-BR')} • Otimizada por menor preço\n\n`;

    const storeKeys = Object.keys(groupedByStore);
    for (const store of storeKeys) {
      const group = groupedByStore[store];
      text += `🏪 *${group.storeName.toUpperCase()}*\n`;

      const isKnownStore = group.storeName !== 'Sem Histórico (Qualquer Loja)';
      if (isKnownStore) {
        const gpsLink = getGpsUrl(group.storeName, group.storeAddress);
        if (group.storeAddress && group.storeAddress.trim()) {
          text += `📍 Endereço: ${group.storeAddress.trim()}\n`;
        }
        text += `🧭 Abrir no GPS: ${gpsLink}\n`;
      }
      text += `\n`;

      for (const it of group.items) {
        const priceStr = it.product.lowest_price_per_unit > 0
          ? `R$ ${it.product.lowest_price_per_unit.toFixed(2).replace('.', ',')}/${it.product.base_unit}`
          : 'Consultar no local';
        const itemSubtotal = it.product.lowest_price_per_unit > 0
          ? `(~R$ ${(it.product.lowest_price_per_unit * it.desired_quantity).toFixed(2).replace('.', ',')})`
          : '';
        text += `  ▫️ ${it.desired_quantity}x ${it.product.name} — ${priceStr} ${itemSubtotal}\n`;
      }
      if (group.subtotal > 0) {
        text += `  💰 Subtotal estimado: R$ ${group.subtotal.toFixed(2).replace('.', ',')}\n`;
      }
      text += `\n`;
    }

    if (overallTotal > 0) {
      text += `💵 TOTAL GERAL ESTIMADO: R$ ${overallTotal.toFixed(2).replace('.', ',')}\n`;
    }
    text += `\nGerado pelo sistema Compras Residenciais`;
    return text;
  }

  // Copiar para a área de transferência
  async function copyToClipboard() {
    try {
      const text = generateFormattedText();
      await navigator.clipboard.writeText(text);
      copied = true;
      feedbackMessage = 'Lista copiada para a área de transferência com links de GPS!';
      setTimeout(() => { copied = false; feedbackMessage = ''; }, 3500);
    } catch {
      errorMessage = 'Não foi possível copiar automaticamente. Selecione e copie o texto.';
    }
  }

  // Envio da lista
  async function handleSend() {
    errorMessage = '';
    feedbackMessage = '';

    const text = generateFormattedText();

    if (dispatchChannel === 'whatsapp') {
      const raw = customPhone.replace(/\D/g, '');
      if (!raw) {
        errorMessage = 'Informe ou selecione o número de WhatsApp do destinatário.';
        return;
      }
      // Se não tiver DDI (ex: 11987654321), adiciona 55 do Brasil
      const fullNumber = raw.length <= 11 ? `55${raw}` : raw;
      const url = `https://api.whatsapp.com/send?phone=${fullNumber}&text=${encodeURIComponent(text)}`;
      window.open(url, '_blank');
      feedbackMessage = 'WhatsApp aberto com a lista formatada e links do GPS!';
    } else {
      // E-mail
      if (!customEmail.trim()) {
        errorMessage = 'Informe ou selecione o e-mail do morador destinatário.';
        return;
      }
      isSending = true;
      const subject = `🛒 Lista de Compras Otimizada (${selectedItems.length} itens)`;
      try {
        await api.sendShoppingListEmail(customEmail.trim(), subject, text);
        // Também abre o cliente de e-mail como fallback
        const mailtoUrl = `mailto:${encodeURIComponent(customEmail.trim())}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(text)}`;
        window.open(mailtoUrl, '_blank');
        feedbackMessage = `Lista despachada com sucesso para ${customEmail}!`;
      } catch (err: any) {
        // Fallback direto via mailto
        const mailtoUrl = `mailto:${encodeURIComponent(customEmail.trim())}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(text)}`;
        window.open(mailtoUrl, '_blank');
        feedbackMessage = 'Cliente de e-mail aberto para envio!';
      } finally {
        isSending = false;
      }
    }
  }
</script>

<div class="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto">
  <div class="bg-slate-900 border border-slate-700 rounded-3xl w-full max-w-4xl shadow-2xl overflow-hidden my-8">
    
    <!-- Modal Header -->
    <div class="px-6 py-4 border-b border-slate-800 flex items-center justify-between bg-slate-900/60">
      <div class="flex items-center gap-3">
        <div class="h-10 w-10 rounded-xl bg-gradient-to-tr from-emerald-500 to-teal-400 text-slate-950 flex items-center justify-center shadow-lg shadow-emerald-500/20">
          <ShoppingCart class="w-5 h-5 stroke-[2.5]" />
        </div>
        <div>
          <h2 class="text-lg font-bold text-white flex items-center gap-2">
            <span>Conferência da Lista de Compras</span>
            <span class="text-xs font-semibold px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              {selectedItems.length} {selectedItems.length === 1 ? 'item' : 'itens'}
            </span>
          </h2>
          <p class="text-xs text-slate-400">
            Itens agrupados pelo estabelecimento mais econômico com links diretos para navegação no GPS
          </p>
        </div>
      </div>

      <button
        onclick={onclose}
        class="text-slate-400 hover:text-slate-200 p-2 rounded-lg hover:bg-slate-800 transition cursor-pointer"
        title="Fechar"
      >
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Feedback Banners -->
    {#if feedbackMessage}
      <div class="mx-6 mt-4 p-3.5 rounded-xl bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2">
        <Check class="w-4 h-4 shrink-0 text-emerald-400" />
        <span>{feedbackMessage}</span>
      </div>
    {/if}
    {#if errorMessage}
      <div class="mx-6 mt-4 p-3.5 rounded-xl bg-rose-500/15 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2">
        <AlertCircle class="w-4 h-4 shrink-0" />
        <span>{errorMessage}</span>
      </div>
    {/if}

    <div class="p-6 space-y-6">
      <!-- Items Grouped by Store -->
      <div class="space-y-4 max-h-96 overflow-y-auto pr-1">
        {#each Object.entries(groupedByStore) as [storeName, group]}
          <div class="bg-slate-950/70 border border-slate-800 rounded-2xl overflow-hidden shadow-sm">
            <!-- Store Header -->
            <div class="p-3.5 bg-slate-900/90 border-b border-slate-800/80 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="flex items-start sm:items-center gap-2.5">
                <div class="p-1.5 rounded-lg bg-emerald-500/10 text-emerald-400 mt-0.5 sm:mt-0">
                  <Store class="w-4 h-4" />
                </div>
                <div>
                  <div class="flex items-center gap-2 flex-wrap">
                    <h4 class="text-sm font-bold text-white">{storeName}</h4>
                    {#if storeName !== 'Sem Histórico (Qualquer Loja)'}
                      <a
                        href={getGpsUrl(group.storeName, group.storeAddress)}
                        target="_blank"
                        rel="noopener noreferrer"
                        class="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-400 hover:text-emerald-300 bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 px-2 py-0.5 rounded-lg transition"
                        title="Abrir no Google Maps / GPS"
                      >
                        <Navigation class="w-3 h-3" />
                        <span>Abrir no GPS</span>
                        <ExternalLink class="w-2.5 h-2.5 opacity-70" />
                      </a>
                    {/if}
                  </div>
                  {#if group.storeAddress}
                    <p class="text-[11px] text-slate-400 flex items-center gap-1 mt-0.5">
                      <MapPin class="w-3 h-3 text-emerald-400/80 shrink-0" />
                      <span class="truncate max-w-md sm:max-w-lg" title={group.storeAddress}>{group.storeAddress}</span>
                    </p>
                  {/if}
                  <span class="text-[10px] text-slate-500">
                    {group.items.length} {group.items.length === 1 ? 'item recomendado' : 'itens recomendados'}\n                  </span>
                </div>
              </div>

              {#if group.subtotal > 0}
                <div class="text-left sm:text-right shrink-0">
                  <span class="text-[10px] text-slate-400 block uppercase">Subtotal Estimado</span>
                  <span class="text-xs sm:text-sm font-black font-mono text-emerald-400">
                    {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(group.subtotal)}
                  </span>
                </div>
              {/if}
            </div>

            <!-- Items in this store -->
            <div class="divide-y divide-slate-800/50">
              {#each group.items as it}
                <div class="p-3 flex items-center justify-between gap-3 text-xs hover:bg-slate-800/20 transition">
                  <div class="min-w-0 flex-1">
                    <div class="font-semibold text-slate-200 text-xs sm:text-sm truncate">
                      {it.product.name}
                    </div>
                    <div class="text-[11px] text-slate-400 flex flex-wrap items-center gap-2 mt-0.5">
                      <span class="px-1.5 py-0.5 rounded bg-slate-900 border border-slate-800 text-slate-400">
                        {it.product.category || 'Geral'}
                      </span>
                      {#if it.product.lowest_price_per_unit > 0}
                        <span class="text-emerald-400 font-mono">
                          Menor: {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(it.product.lowest_price_per_unit)}/{it.product.base_unit}
                        </span>
                        {#if it.product.cheapest_package_info}
                          <span class="text-slate-500 font-mono text-[10px]">({it.product.cheapest_package_info})</span>
                        {/if}
                      {:else}
                        <span class="text-slate-500 italic">Sem registro recente</span>
                      {/if}
                    </div>
                  </div>

                  <!-- Quantity Controls -->
                  <div class="flex items-center gap-2 shrink-0">
                    <div class="flex items-center border border-slate-700 rounded-lg overflow-hidden bg-slate-900">
                      <button
                        type="button"
                        onclick={() => onupdatequantity(it.product.id, Math.max(1, it.desired_quantity - 1))}
                        class="px-2 py-1 text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
                        title="Diminuir"
                      >
                        -
                      </button>
                      <span class="px-2.5 py-1 text-xs font-mono font-bold text-white">
                        {it.desired_quantity}
                      </span>
                      <button
                        type="button"
                        onclick={() => onupdatequantity(it.product.id, it.desired_quantity + 1)}
                        class="px-2 py-1 text-slate-400 hover:text-white hover:bg-slate-800 transition cursor-pointer"
                        title="Aumentar"
                      >
                        +
                      </button>
                    </div>

                    <!-- Item Total -->
                    {#if it.product.lowest_price_per_unit > 0}
                      <span class="font-mono font-bold text-slate-200 min-w-[70px] text-right">
                        {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(it.product.lowest_price_per_unit * it.desired_quantity)}
                      </span>
                    {/if}

                    <!-- Remove Item -->
                    <button
                      type="button"
                      onclick={() => onremoveitem(it.product.id)}
                      class="p-1.5 rounded-lg text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 transition cursor-pointer"
                      title="Remover da lista"
                    >
                      <Trash2 class="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/each}
      </div>

      <!-- Total Summary Bar -->
      <div class="p-4 rounded-2xl bg-slate-950 border border-slate-800 flex flex-col sm:flex-row items-center justify-between gap-3">
        <div>
          <span class="text-xs text-slate-400 uppercase font-semibold">Total Geral Estimado da Lista</span>
          <div class="text-2xl font-black text-white font-mono tracking-tight text-emerald-400">
            {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(overallTotal)}
          </div>
        </div>

        <button
          onclick={copyToClipboard}
          class="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl border border-slate-700 hover:border-slate-600 bg-slate-900 text-slate-300 hover:text-white text-xs font-semibold transition cursor-pointer"
          title="Copiar texto formatado da lista"
        >
          {#if copied}
            <Check class="w-3.5 h-3.5 text-emerald-400" />
            <span class="text-emerald-400">Copiado!</span>
          {:else}
            <Copy class="w-3.5 h-3.5" />
            <span>Copiar Lista (Texto)</span>
          {/if}
        </button>
      </div>

      <!-- Dispatch Section (Morador & Canal WhatsApp / Email) -->
      <div class="p-5 rounded-2xl bg-slate-950/80 border border-slate-800 space-y-4">
        <h4 class="text-xs font-bold uppercase tracking-wider text-slate-300 flex items-center gap-2">
          <Share2 class="w-4 h-4 text-emerald-400" />
          Enviar Lista para um Morador
        </h4>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <!-- Morador Responsável -->
          <div>
            <label for="dispatch-user-select" class="block text-xs font-semibold text-slate-400 mb-1.5">
              Selecionar Morador Cadastrado
            </label>
            <select
              id="dispatch-user-select"
              value={selectedUserId}
              onchange={(e) => handleSelectUser((e.target as HTMLSelectElement).value)}
              class="w-full px-3 py-2.5 bg-slate-900 border border-slate-700 rounded-xl text-slate-200 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
            >
              <option value="">Selecione o destinatário...</option>
              {#each userList as u}
                <option value={u.id}>{u.name} {u.phone ? `(${u.phone})` : (u.email ? `(${u.email})` : '')}</option>
              {/each}
            </select>
          </div>

          <!-- Canal de Envio: WhatsApp vs E-mail -->
          <div>
            <span class="block text-xs font-semibold text-slate-400 mb-1.5">Canal de Envio</span>
            <div class="grid grid-cols-2 gap-2">
              <button
                type="button"
                onclick={() => { dispatchChannel = 'whatsapp'; }}
                class="flex items-center justify-center gap-2 px-3 py-2 rounded-xl text-xs font-bold border transition cursor-pointer {dispatchChannel === 'whatsapp' ? 'bg-emerald-500/15 border-emerald-500/50 text-emerald-400' : 'bg-slate-900 border-slate-700 text-slate-400 hover:text-slate-200'}"
              >
                <Phone class="w-3.5 h-3.5" />
                <span>WhatsApp</span>
              </button>

              <button
                type="button"
                onclick={() => { dispatchChannel = 'email'; }}
                class="flex items-center justify-center gap-2 px-3 py-2 rounded-xl text-xs font-bold border transition cursor-pointer {dispatchChannel === 'email' ? 'bg-sky-500/15 border-sky-500/50 text-sky-400' : 'bg-slate-900 border-slate-700 text-slate-400 hover:text-slate-200'}"
              >
                <Mail class="w-3.5 h-3.5" />
                <span>E-mail</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Inputs específicos de acordo com o canal -->
        {#if dispatchChannel === 'whatsapp'}\n          <div>
            <label for="dispatch-whatsapp-input" class="block text-xs font-semibold text-slate-400 mb-1">
              Número de WhatsApp do Destinatário
            </label>
            <div class="relative">
              <input
                id="dispatch-whatsapp-input"
                type="tel"
                bind:value={customPhone}
                placeholder="Ex: (11) 98765-4321 ou 5511999998888"
                class="w-full pl-9 pr-3 py-2 bg-slate-900 border border-slate-700 rounded-xl text-slate-100 text-xs font-mono focus:ring-1 focus:ring-emerald-500 focus:outline-none"
              />
              <Phone class="w-3.5 h-3.5 text-emerald-400 absolute left-3 top-2.5" />
            </div>
          </div>
        {:else}
          <div>
            <label for="dispatch-email-input" class="block text-xs font-semibold text-slate-400 mb-1">
              Endereço de E-mail do Destinatário
            </label>
            <div class="relative">
              <input
                id="dispatch-email-input"
                type="email"
                bind:value={customEmail}
                placeholder="morador@email.com"
                class="w-full pl-9 pr-3 py-2 bg-slate-900 border border-slate-700 rounded-xl text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
              />
              <Mail class="w-3.5 h-3.5 text-sky-400 absolute left-3 top-2.5" />
            </div>
          </div>
        {/if}

        <div class="pt-2 flex flex-col sm:flex-row items-center justify-between gap-3 border-t border-slate-800/80">
          <p class="text-[11px] text-slate-500">
            A lista é agrupada automaticamente pelos estabelecimentos mais econômicos com link de GPS.
          </p>

          <button
            onclick={handleSend}
            disabled={isSending || selectedItems.length === 0}
            class="w-full sm:w-auto px-6 py-2.5 rounded-xl {dispatchChannel === 'whatsapp' ? 'bg-emerald-500 hover:bg-emerald-400 text-slate-950' : 'bg-sky-500 hover:bg-sky-400 text-slate-950'} font-bold text-xs shadow-lg transition flex items-center justify-center gap-2 cursor-pointer disabled:opacity-50"
          >
            {#if isSending}
              <div class="w-3.5 h-3.5 border-2 border-slate-950 border-t-transparent rounded-full animate-spin"></div>
              <span>Despachando...</span>
            {:else if dispatchChannel === 'whatsapp'}
              <Phone class="w-4 h-4" />
              <span>Abrir no WhatsApp</span>
              <ExternalLink class="w-3.5 h-3.5" />
            {:else}
              <Send class="w-4 h-4" />
              <span>Enviar por E-mail</span>
            {/if}
          </button>
        </div>
      </div>
    </div>
  </div>
</div>
