<script lang="ts">
  import type { User } from '../types';
  import { Users, UserPlus, Trash2, ShieldAlert, CheckCircle, Mail, AlertCircle } from '@lucide/svelte';
  import { api } from '../lib/api';

  interface Props {
    users: User[];
    loading: boolean;
    onrefresh: () => void;
  }

  let { users, loading, onrefresh }: Props = $props();

  let name = $state('');
  let email = $state('');
  let isSubmitting = $state(false);
  let feedbackMessage = $state('');
  let errorMessage = $state('');

  async function handleCreateUser(e: Event) {
    e.preventDefault();
    if (!name.trim()) {
      errorMessage = 'Informe o nome do morador.';
      return;
    }
    isSubmitting = true;
    errorMessage = '';
    feedbackMessage = '';
    try {
      await api.createUser(name.trim(), email.trim());
      name = '';
      email = '';
      feedbackMessage = 'Morador cadastrado com sucesso!';
      onrefresh();
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao cadastrar morador.';
    } finally {
      isSubmitting = false;
    }
  }

  async function handleDeleteUser(user: User) {
    if (!confirm(`Deseja remover ou bloquear o morador "${user.name}"?`)) return;

    try {
      const res = await api.deleteUser(user.id);
      feedbackMessage = res.message;
      onrefresh();
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao processar morador.';
    }
  }
</script>

<div class="space-y-6">
  <!-- Header bar -->
  <div class="bg-slate-900 border border-slate-800 p-5 rounded-2xl">
    <div class="flex items-center gap-3">
      <div class="p-2.5 rounded-xl bg-amber-500/10 text-amber-400">
        <Users class="w-5 h-5" />
      </div>
      <div>
        <h2 class="text-lg font-bold text-white">Moradores da Residência</h2>
        <p class="text-xs text-slate-400">
          Gerencie os membros da casa responsáveis por realizar compras e dividir as despesas
        </p>
      </div>
    </div>
  </div>

  <!-- Messages -->
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
    <!-- Form to add user -->
    <div class="bg-slate-900 border border-slate-800 p-5 rounded-2xl h-fit">
      <h3 class="text-sm font-bold text-white mb-3 flex items-center gap-2">
        <UserPlus class="w-4 h-4 text-emerald-400" />
        Adicionar Novo Morador
      </h3>
      <form onsubmit={handleCreateUser} class="space-y-4">
        <div>
          <label for="new-user-name" class="block text-xs font-semibold text-slate-300 mb-1">Nome Completo *</label>
          <input
            id="new-user-name"
            type="text"
            bind:value={name}
            placeholder="Ex: Carlos Silva"
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
          />
        </div>

        <div>
          <label for="new-user-email" class="block text-xs font-semibold text-slate-300 mb-1">E-mail (opcional)</label>
          <input
            id="new-user-email"
            type="email"
            bind:value={email}
            placeholder="carlos@email.com"
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
          />
        </div>

        <button
          type="submit"
          disabled={isSubmitting || !name.trim()}
          class="w-full py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-bold text-xs shadow-md transition cursor-pointer"
        >
          {isSubmitting ? 'Cadastrando...' : 'Cadastrar Morador'}
        </button>
      </form>
    </div>

    <!-- Users Table/List -->
    <div class="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
      <div class="p-4 border-b border-slate-800 flex justify-between items-center">
        <span class="text-xs font-bold uppercase text-slate-400">Moradores Registrados ({users.length})</span>
      </div>

      {#if loading}
        <div class="p-8 text-center text-slate-400 text-sm">Carregando moradores...</div>
      {:else if users.length === 0}
        <div class="p-8 text-center text-slate-500 text-xs">
          Nenhum morador cadastrado ainda. Adicione o primeiro no formulário ao lado.
        </div>
      {:else}
        <div class="divide-y divide-slate-800/60">
          {#each users as u}
            <div class="p-4 flex items-center justify-between hover:bg-slate-800/30 transition">
              <div class="flex items-center gap-3">
                <div class="h-10 w-10 rounded-full bg-gradient-to-tr from-emerald-500 to-teal-400 flex items-center justify-center font-bold text-slate-950 text-sm">
                  {u.name.substring(0, 2).toUpperCase()}
                </div>
                <div>
                  <div class="font-semibold text-white text-sm flex items-center gap-2">
                    <span>{u.name}</span>
                    {#if u.is_blocked}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20">
                        <ShieldAlert class="w-3 h-3" /> Bloqueado
                      </span>
                    {:else}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                        Ativo
                      </span>
                    {/if}
                  </div>
                  {#if u.email}
                    <div class="text-xs text-slate-400 flex items-center gap-1 mt-0.5">
                      <Mail class="w-3 h-3 text-slate-500" />
                      <span>{u.email}</span>
                    </div>
                  {/if}
                </div>
              </div>

              <div>
                <button
                  onclick={() => handleDeleteUser(u)}
                  class="p-2 rounded-lg bg-slate-800 hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 transition cursor-pointer"
                  title="Remover ou Bloquear"
                >
                  <Trash2 class="w-4 h-4" />
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
