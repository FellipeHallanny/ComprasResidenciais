<script lang="ts">
  import type { Store } from '../types';
  import { Store as StoreIcon, Plus, Building2, CheckCircle, AlertCircle } from '@lucide/svelte';
  import { api } from '../lib/api';

  interface Props {
    stores: Store[];
    loading: boolean;
    onrefresh: () => void;
  }

  let { stores, loading, onrefresh }: Props = $props();

  let name = $state('');
  let cnpj = $state('');
  let isSubmitting = $state(false);
  let feedbackMessage = $state('');
  let errorMessage = $state('');

  async function handleCreateStore(e: Event) {
    e.preventDefault();
    if (!name.trim()) {
      errorMessage = 'Informe o nome do estabelecimento.';
      return;
    }
    isSubmitting = true;
    errorMessage = '';
    feedbackMessage = '';
    try {
      await api.createStore(name.trim(), cnpj.trim());
      name = '';
      cnpj = '';
      feedbackMessage = 'Estabelecimento cadastrado com sucesso!';
      onrefresh();
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao cadastrar estabelecimento.';
    } finally {
      isSubmitting = false;
    }
  }
</script>

<div class="space-y-6">
  <div class="bg-slate-900 border border-slate-800 p-5 rounded-2xl">
    <div class="flex items-center gap-3">
      <div class="p-2.5 rounded-xl bg-purple-500/10 text-purple-400">
        <StoreIcon class="w-5 h-5" />
      </div>
      <div>
        <h2 class="text-lg font-bold text-white">Estabelecimentos Comerciais</h2>
        <p class="text-xs text-slate-400">
          Supermercados, atacadistas, feiras e comércios locais monitorados
        </p>
      </div>
    </div>
  </div>

  {#if feedbackMessage}
    <div class="p-3.5 rounded-xl bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2">
      <CheckCircle class="w-4 h-4 shrink-0" />
      <span>{feedbackMessage}</span>
    </div>
  {/if}
  {#if errorMessage}
    <div class="p-3.5 rounded-xl bg-rose-500/15 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2">
      <AlertCircle class="w-4 h-4 shrink-0" />
      <span>{errorMessage}</span>
    </div>
  {/if}

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Form to add store -->
    <div class="bg-slate-900 border border-slate-800 p-5 rounded-2xl h-fit">
      <h3 class="text-sm font-bold text-white mb-3 flex items-center gap-2">
        <Plus class="w-4 h-4 text-emerald-400" />
        Novo Estabelecimento
      </h3>
      <form onsubmit={handleCreateStore} class="space-y-4">
        <div>
          <label for="new-store-name" class="block text-xs font-semibold text-slate-300 mb-1">Nome Fantasia *</label>
          <input
            id="new-store-name"
            type="text"
            bind:value={name}
            placeholder="Ex: Pão de Açúcar / Assaí"
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
          />
        </div>

        <div>
          <label for="new-store-cnpj" class="block text-xs font-semibold text-slate-300 mb-1">CNPJ (opcional)</label>
          <input
            id="new-store-cnpj"
            type="text"
            bind:value={cnpj}
            placeholder="00.000.000/0000-00"
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
          />
        </div>

        <button
          type="submit"
          disabled={isSubmitting || !name.trim()}
          class="w-full py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-bold text-xs shadow-md transition cursor-pointer"
        >
          {isSubmitting ? 'Salvando...' : 'Salvar Estabelecimento'}
        </button>
      </form>
    </div>

    <!-- Stores Table/Grid -->
    <div class="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
      <div class="p-4 border-b border-slate-800 flex justify-between items-center">
        <span class="text-xs font-bold uppercase text-slate-400">Lojas Cadastradas ({stores.length})</span>
      </div>

      {#if loading}
        <div class="p-8 text-center text-slate-400 text-sm">Carregando estabelecimentos...</div>
      {:else if stores.length === 0}
        <div class="p-8 text-center text-slate-500 text-xs">
          Nenhum estabelecimento cadastrado ainda.
        </div>
      {:else}
        <div class="divide-y divide-slate-800/60">
          {#each stores as s}
            <div class="p-4 flex items-center justify-between hover:bg-slate-800/30 transition">
              <div class="flex items-center gap-3">
                <div class="h-10 w-10 rounded-xl bg-purple-500/10 text-purple-400 flex items-center justify-center font-bold border border-purple-500/20">
                  <Building2 class="w-5 h-5" />
                </div>
                <div>
                  <div class="font-semibold text-white text-sm">
                    {s.name}
                  </div>
                  {#if s.cnpj}
                    <div class="text-xs text-slate-400 font-mono mt-0.5">
                      CNPJ: {s.cnpj}
                    </div>
                  {/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
