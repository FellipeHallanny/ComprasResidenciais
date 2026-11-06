<script lang="ts">
  import type { Store } from '../types';
  import {
    Store as StoreIcon,
    Plus,
    Building2,
    Pencil,
    Trash2,
    CheckCircle,
    AlertCircle,
    X,
    Check,
    MapPin,
    Map,
    Search,
    ExternalLink
  } from '@lucide/svelte';
  import { api } from '../lib/api';
  import LocationPickerModal from './LocationPickerModal.svelte';

  interface Props {
    stores: Store[];
    loading: boolean;
    onrefresh: () => void;
  }

  let { stores, loading, onrefresh }: Props = $props();

  const storeList = $derived(stores ?? []);

  let editingStore = $state<Store | null>(null);
  let name = $state('');
  let cnpj = $state('');
  let address = $state('');
  let cep = $state('');
  let streetNumber = $state('');
  let isSearchingCep = $state(false);
  let cepFeedback = $state('');
  let isSubmitting = $state(false);
  let feedbackMessage = $state('');
  let errorMessage = $state('');

  // Cached components from CEP lookup to allow dynamic number updates
  let baseStreet = $state('');
  let baseNeighborhood = $state('');
  let baseCityState = $state('');
  let baseCep = $state('');

  // Map modal state
  let isMapModalOpen = $state(false);

  function startEdit(store: Store) {
    editingStore = store;
    name = store.name;
    cnpj = store.cnpj || '';
    address = store.address || '';
    cep = '';
    streetNumber = '';
    cepFeedback = '';
    baseStreet = '';
    baseNeighborhood = '';
    baseCityState = '';
    baseCep = '';
    errorMessage = '';
    feedbackMessage = '';
  }

  function cancelEdit() {
    editingStore = null;
    name = '';
    cnpj = '';
    address = '';
    cep = '';
    streetNumber = '';
    cepFeedback = '';
    baseStreet = '';
    baseNeighborhood = '';
    baseCityState = '';
    baseCep = '';
    errorMessage = '';
  }

  function rebuildAddress() {
    if (!baseStreet && !baseCityState) return;
    let parts: string[] = [];
    if (baseStreet) {
      parts.push(streetNumber.trim() ? `${baseStreet}, ${streetNumber.trim()}` : baseStreet);
    }
    if (baseNeighborhood) parts.push(baseNeighborhood);
    if (baseCityState) parts.push(baseCityState);
    if (baseCep) parts.push(`CEP ${baseCep}`);
    address = parts.join(', ');
  }

  function handleNumberInput() {
    if (baseStreet || baseCityState) {
      rebuildAddress();
    }
  }

  // Format and search Brazilian postal code (ViaCEP)
  async function searchCep() {
    const cleanCep = cep.replace(/\D/g, '');
    if (cleanCep.length !== 8) {
      errorMessage = 'Informe um CEP válido com 8 dígitos numéricos.';
      return;
    }

    isSearchingCep = true;
    errorMessage = '';
    cepFeedback = '';

    try {
      const res = await fetch(`https://viacep.com.br/ws/${cleanCep}/json/`);
      if (!res.ok) throw new Error('Não foi possível consultar o ViaCEP.');
      const data = await res.json();

      if (data.erro) {
        errorMessage = `CEP "${cep}" não encontrado na base dos Correios.`;
        return;
      }

      baseStreet = data.logradouro || '';
      baseNeighborhood = data.bairro || '';
      baseCityState = data.localidade && data.uf ? `${data.localidade} - ${data.uf}` : (data.localidade || '');
      baseCep = data.cep || cleanCep;

      rebuildAddress();
      cepFeedback = `Endereço localizado: ${baseStreet ? baseStreet + ', ' : ''}${baseNeighborhood} (${baseCityState})`;
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao buscar CEP.';
    } finally {
      isSearchingCep = false;
    }
  }

  function handleMapSelect(selectedAddr: string) {
    address = selectedAddr;
    baseStreet = '';
    baseNeighborhood = '';
    baseCityState = '';
    baseCep = '';
    isMapModalOpen = false;
    feedbackMessage = 'Endereço capturado a partir do mapa com sucesso!';
  }

  function openGoogleMapsSearch(query: string) {
    if (!query.trim()) return;
    window.open(`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(query)}`, '_blank');
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!name.trim()) {
      errorMessage = 'Informe o nome do estabelecimento.';
      return;
    }
    isSubmitting = true;
    errorMessage = '';
    feedbackMessage = '';

    try {
      if (editingStore) {
        await api.updateStore(editingStore.id, {
          name: name.trim(),
          cnpj: cnpj.trim(),
          address: address.trim(),
        });
        feedbackMessage = `Estabelecimento "${name.trim()}" atualizado com sucesso!`;
        cancelEdit();
      } else {
        await api.createStore(name.trim(), cnpj.trim(), address.trim());
        name = '';
        cnpj = '';
        address = '';
        cep = '';
        streetNumber = '';
        cepFeedback = '';
        baseStreet = '';
        baseNeighborhood = '';
        baseCityState = '';
        baseCep = '';
        feedbackMessage = 'Estabelecimento cadastrado com sucesso!';
      }
      onrefresh();
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao salvar estabelecimento.';
    } finally {
      isSubmitting = false;
    }
  }

  async function handleDeleteStore(store: Store) {
    if (!confirm(`Deseja remover o estabelecimento "${store.name}"?`)) return;

    try {
      const res = await api.deleteStore(store.id);
      feedbackMessage = res.message || 'Estabelecimento removido com sucesso.';
      if (editingStore?.id === store.id) {
        cancelEdit();
      }
      onrefresh();
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao remover estabelecimento.';
    }
  }
</script>

<div class="space-y-6">
  <div class="bg-slate-900 border border-slate-800 p-5 rounded-2xl">
    <div class="flex items-center gap-3">
      <div class="p-2.5 rounded-xl bg-purple-500/10 text-purple-400">
        <StoreIcon class="w-5 h-5" aria-hidden="true" />
      </div>
      <div>
        <h2 class="text-lg font-bold text-white text-pretty">Estabelecimentos Comerciais</h2>
        <p class="text-xs text-slate-400">
          Supermercados, atacadistas, feiras e comércios locais monitorados
        </p>
      </div>
    </div>
  </div>

  {#if feedbackMessage}
    <div class="p-3.5 rounded-xl bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2" aria-live="polite">
      <CheckCircle class="w-4 h-4 shrink-0" aria-hidden="true" />
      <span>{feedbackMessage}</span>
    </div>
  {/if}
  {#if errorMessage}
    <div class="p-3.5 rounded-xl bg-rose-500/15 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2" aria-live="polite">
      <AlertCircle class="w-4 h-4 shrink-0" aria-hidden="true" />
      <span>{errorMessage}</span>
    </div>
  {/if}

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Form to add/edit store -->
    <div class="bg-slate-900 border border-slate-800 p-5 rounded-2xl h-fit">
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-sm font-bold text-white flex items-center gap-2">
          {#if editingStore}
            <Pencil class="w-4 h-4 text-amber-400" aria-hidden="true" />
            <span>Editar Estabelecimento</span>
          {:else}
            <Plus class="w-4 h-4 text-emerald-400" aria-hidden="true" />
            <span>Novo Estabelecimento</span>
          {/if}
        </h3>
        {#if editingStore}
          <button
            onclick={cancelEdit}
            class="text-xs text-slate-400 hover:text-slate-200 flex items-center gap-1 cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-slate-400 rounded px-1"
            aria-label="Cancelar edição"
          >
            <X class="w-3.5 h-3.5" aria-hidden="true" /> <span>Cancelar</span>
          </button>
        {/if}
      </div>

      <form onsubmit={handleSubmit} class="space-y-4">
        <div>
          <label for="store-name" class="block text-xs font-semibold text-slate-300 mb-1">Nome Fantasia *</label>
          <input
            id="store-name"
            name="store-name"
            type="text"
            autocomplete="organization"
            bind:value={name}
            placeholder="Ex: Pão de Açúcar / Assaí / Feira Livre"
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500"
          />
        </div>

        <div>
          <label for="store-cnpj" class="block text-xs font-semibold text-slate-300 mb-1">CNPJ (opcional)</label>
          <input
            id="store-cnpj"
            name="store-cnpj"
            type="text"
            autocomplete="off"
            bind:value={cnpj}
            placeholder="00.000.000/0000-00"
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500 font-mono"
          />
        </div>

        <!-- Localização: Busca por CEP e Seleção no Mapa -->
        <div class="p-3.5 bg-slate-950/70 border border-slate-800 rounded-xl space-y-3">
          <div class="flex items-center justify-between gap-2">
            <span class="text-xs font-semibold text-purple-400 flex items-center gap-1.5 shrink-0">
              <MapPin class="w-3.5 h-3.5" aria-hidden="true" /> Localização & Endereço
            </span>

            <button
              type="button"
              onclick={() => (isMapModalOpen = true)}
              class="text-[11px] font-semibold text-purple-400 hover:text-purple-300 bg-purple-500/10 hover:bg-purple-500/20 px-2.5 py-1 rounded-lg border border-purple-500/30 flex items-center gap-1 transition-colors cursor-pointer shrink-0 focus:outline-none focus-visible:ring-1 focus-visible:ring-purple-400"
              title="Abrir mapa interativo para clicar no local"
            >
              <Map class="w-3 h-3" aria-hidden="true" />
              <span>Selecionar no Mapa</span>
            </button>
          </div>

          <!-- CEP Lookup Row -->
          <div class="space-y-3">
            <div>
              <label for="store-cep" class="block text-[11px] text-slate-400 mb-1">Buscar por CEP</label>
              <div class="flex items-center gap-2">
                <div class="relative flex-1 min-w-0">
                  <input
                    id="store-cep"
                    name="postal-code"
                    type="text"
                    autocomplete="postal-code"
                    maxlength="9"
                    bind:value={cep}
                    onkeydown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        searchCep();
                      }
                    }}
                    placeholder="00000-000"
                    class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-purple-500 font-mono"
                  />
                </div>
                <button
                  type="button"
                  onclick={searchCep}
                  disabled={isSearchingCep || cep.replace(/\D/g, '').length !== 8}
                  class="shrink-0 px-3.5 py-2 bg-purple-600 hover:bg-purple-500 disabled:opacity-40 text-white rounded-xl text-xs font-semibold transition-colors cursor-pointer flex items-center gap-1.5 shadow-sm focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-400"
                  title="Consultar CEP nos Correios"
                  aria-label="Buscar CEP nos Correios"
                >
                  {#if isSearchingCep}
                    <div class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
                  {:else}
                    <Search class="w-3.5 h-3.5" aria-hidden="true" />
                  {/if}
                  <span>Buscar CEP</span>
                </button>
              </div>
            </div>

            <div>
              <label for="store-number" class="block text-[11px] text-slate-400 mb-1">Número / Complemento (opcional)</label>
              <input
                id="store-number"
                name="address-line2"
                type="text"
                autocomplete="address-line2"
                bind:value={streetNumber}
                oninput={handleNumberInput}
                placeholder="Ex: 1500, Bloco B ou S/N"
                class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-purple-500"
              />
            </div>
          </div>

          {#if cepFeedback}
            <div class="text-[11px] text-emerald-400 flex items-center gap-1" aria-live="polite">
              <CheckCircle class="w-3 h-3 shrink-0" aria-hidden="true" />
              <span>{cepFeedback}</span>
            </div>
          {/if}

          <!-- Address textarea / final input -->
          <div>
            <label for="store-address" class="block text-[11px] text-slate-400 mb-1">Endereço Completo</label>
            <textarea
              id="store-address"
              name="street-address"
              autocomplete="street-address"
              rows="2"
              bind:value={address}
              placeholder="Ex: Av. Brigadeiro Faria Lima, 1200 - Pinheiros, São Paulo - SP"
              class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500"
            ></textarea>
          </div>

          {#if address}
            <div class="flex justify-end">
              <button
                type="button"
                onclick={() => openGoogleMapsSearch(name ? `${name} ${address}` : address)}
                class="text-[11px] text-blue-400 hover:text-blue-300 hover:underline flex items-center gap-1 cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-blue-400 rounded"
              >
                <ExternalLink class="w-3 h-3" aria-hidden="true" />
                <span>Conferir no Google Maps</span>
              </button>
            </div>
          {/if}
        </div>

        <div class="flex gap-2">
          {#if editingStore}
            <button
              type="button"
              onclick={cancelEdit}
              class="px-3 py-2.5 rounded-xl border border-slate-700 text-slate-300 hover:bg-slate-800 font-semibold text-xs transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
            >
              Cancelar
            </button>
          {/if}
          <button
            type="submit"
            disabled={isSubmitting || !name.trim()}
            class="flex-1 py-2.5 rounded-xl {editingStore ? 'bg-amber-500 hover:bg-amber-400 text-slate-950' : 'bg-emerald-500 hover:bg-emerald-400 text-slate-950'} disabled:opacity-50 font-bold text-xs shadow-md transition-colors cursor-pointer flex items-center justify-center gap-1.5 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
          >
            {#if isSubmitting}
              <div class="w-3.5 h-3.5 border-2 border-slate-950 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
              <span>Salvando…</span>
            {:else if editingStore}
              <Check class="w-3.5 h-3.5 stroke-[3]" aria-hidden="true" />
              <span>Salvar Alterações</span>
            {:else}
              <span>Salvar Estabelecimento</span>
            {/if}
          </button>
        </div>
      </form>
    </div>

    <!-- Stores Table/Grid -->
    <div class="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
      <div class="p-4 border-b border-slate-800 flex justify-between items-center">
        <span class="text-xs font-bold uppercase text-slate-400">Lojas Cadastradas ({storeList.length})</span>
      </div>

      {#if loading}
        <div class="p-8 text-center text-slate-400 text-sm">Carregando estabelecimentos…</div>
      {:else if storeList.length === 0}
        <div class="p-10 text-center text-slate-400 space-y-2">
          <Building2 class="w-8 h-8 text-slate-600 mx-auto" aria-hidden="true" />
          <p class="text-sm font-semibold text-slate-300">Nenhum estabelecimento cadastrado ainda</p>
          <p class="text-xs text-slate-500 max-w-sm mx-auto">
            Cadastre os supermercados, feiras ou mercearias onde você costuma realizar compras.
          </p>
        </div>
      {:else}
        <div class="divide-y divide-slate-800/60">
          {#each storeList as s}
            <div class="p-4 flex items-center justify-between hover:bg-slate-800/30 transition-colors {editingStore?.id === s.id ? 'bg-slate-800/50 border-l-2 border-amber-400' : ''}">
              <div class="flex items-center gap-3">
                <div class="h-10 w-10 rounded-xl bg-purple-500/10 text-purple-400 flex items-center justify-center font-bold border border-purple-500/20 shrink-0">
                  <Building2 class="w-5 h-5" aria-hidden="true" />
                </div>
                <div>
                  <div class="font-semibold text-white text-sm">
                    {s.name}
                  </div>
                  <div class="text-xs text-slate-400 flex flex-wrap items-center gap-2 mt-0.5">
                    {#if s.cnpj}
                      <span class="font-mono tabular-nums">CNPJ: {s.cnpj}</span>
                    {/if}
                    {#if s.address}
                      <span class="flex items-center gap-1 text-slate-400">
                        <MapPin class="w-3 h-3 text-purple-400 shrink-0" aria-hidden="true" />
                        <span>{s.address}</span>
                      </span>
                      <button
                        onclick={() => openGoogleMapsSearch(`${s.name} ${s.address}`)}
                        class="text-[11px] text-blue-400 hover:text-blue-300 hover:underline flex items-center gap-0.5 cursor-pointer ml-1 focus:outline-none focus-visible:ring-1 focus-visible:ring-blue-400 rounded"
                        title="Abrir no Google Maps"
                      >
                        <ExternalLink class="w-3 h-3" aria-hidden="true" />
                        <span>Google Maps</span>
                      </button>
                    {/if}
                  </div>
                </div>
              </div>

              <div class="flex items-center gap-1.5 shrink-0 ml-2">
                <button
                  onclick={() => startEdit(s)}
                  class="p-2 rounded-lg bg-slate-800 hover:bg-amber-500/10 text-slate-400 hover:text-amber-400 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-amber-400"
                  title="Editar Estabelecimento"
                  aria-label="Editar {s.name}"
                >
                  <Pencil class="w-4 h-4" aria-hidden="true" />
                </button>
                <button
                  onclick={() => handleDeleteStore(s)}
                  class="p-2 rounded-lg bg-slate-800 hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400"
                  title="Excluir Estabelecimento"
                  aria-label="Excluir {s.name}"
                >
                  <Trash2 class="w-4 h-4" aria-hidden="true" />
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>

{#if isMapModalOpen}
  <LocationPickerModal
    initialAddress={address}
    onselect={handleMapSelect}
    onclose={() => (isMapModalOpen = false)}
  />
{/if}
