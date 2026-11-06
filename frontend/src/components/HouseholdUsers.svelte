<script lang="ts">
  import type { User } from '../types';
  import {
    Users,
    UserPlus,
    Pencil,
    Trash2,
    ShieldAlert,
    CheckCircle,
    Mail,
    Phone,
    AlertCircle,
    X,
    Check,
    Shield,
    Lock
  } from '@lucide/svelte';
  import { api } from '../lib/api';
  import { auth } from '../lib/auth.svelte';

  interface Props {
    users: User[];
    loading: boolean;
    onrefresh: () => void;
  }

  let { users, loading, onrefresh }: Props = $props();

  const userList = $derived(users ?? []);

  let editingUser = $state<User | null>(null);
  let name = $state('');
  let email = $state('');
  let phone = $state('');
  let role = $state<'ADMIN' | 'MEMBER'>('MEMBER');
  let password = $state('');
  let isBlocked = $state(false);
  let isSubmitting = $state(false);
  let feedbackMessage = $state('');
  let errorMessage = $state('');

  function startEdit(user: User) {
    editingUser = user;
    name = user.name;
    email = user.email || '';
    phone = user.phone || '';
    role = user.role || 'MEMBER';
    isBlocked = !!user.is_blocked;
    password = '';
    errorMessage = '';
    feedbackMessage = '';
  }

  function cancelEdit() {
    editingUser = null;
    name = '';
    email = '';
    phone = '';
    role = 'MEMBER';
    password = '';
    isBlocked = false;
    errorMessage = '';
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!name.trim()) {
      errorMessage = 'Informe o nome do morador.';
      return;
    }
    isSubmitting = true;
    errorMessage = '';
    feedbackMessage = '';

    try {
      if (editingUser) {
        // Modo Edição
        await api.updateUser(editingUser.id, {
          name: name.trim(),
          email: email.trim(),
          phone: phone.trim(),
          role: role,
          is_blocked: isBlocked,
          new_password: password.trim() ? password.trim() : undefined,
        });
        feedbackMessage = `Morador "${name.trim()}" atualizado com sucesso!`;
        cancelEdit();
      } else {
        // Modo Criação
        await api.createUser(name.trim(), email.trim(), phone.trim(), password.trim(), role);
        name = '';
        email = '';
        phone = '';
        password = '';
        role = 'MEMBER';
        feedbackMessage = 'Morador cadastrado com sucesso!';
      }
      onrefresh();
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao salvar dados do morador.';
    } finally {
      isSubmitting = false;
    }
  }

  async function handleDeleteUser(user: User) {
    if (!auth.isAdmin) {
      alert('Apenas administradores podem remover ou bloquear moradores.');
      return;
    }

    if (user.id === auth.currentUser?.id) {
      alert('Você não pode excluir sua própria conta de administrador.');
      return;
    }

    if (!confirm(`Deseja remover ou desativar o morador "${user.name}"?`)) return;

    try {
      const res = await api.deleteUser(user.id);
      feedbackMessage = res.message;
      if (editingUser?.id === user.id) {
        cancelEdit();
      }
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
        <Users class="w-5 h-5" aria-hidden="true" />
      </div>
      <div>
        <h2 class="text-lg font-bold text-white text-pretty">Moradores da Residência</h2>
        <p class="text-xs text-slate-400">
          Gerencie os membros da casa responsáveis por realizar compras, receber listas pelo WhatsApp/e-mail e dividir as despesas
        </p>
      </div>
    </div>
  </div>

  <!-- Messages -->
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
    <!-- Form to add/edit user -->
    <div class="bg-slate-900 border border-slate-800 p-5 rounded-2xl h-fit">
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-sm font-bold text-white flex items-center gap-2">
          {#if editingUser}
            <Pencil class="w-4 h-4 text-amber-400" aria-hidden="true" />
            <span>Editar Morador</span>
          {:else}
            <UserPlus class="w-4 h-4 text-emerald-400" aria-hidden="true" />
            <span>Adicionar Novo Morador</span>
          {/if}
        </h3>
        {#if editingUser}
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
          <label for="user-name" class="block text-xs font-semibold text-slate-300 mb-1">Nome Completo *</label>
          <input
            id="user-name"
            name="name"
            type="text"
            autocomplete="name"
            bind:value={name}
            placeholder="Ex: Carlos Silva"
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500"
          />
        </div>

        <div>
          <label for="user-email" class="block text-xs font-semibold text-slate-300 mb-1">E-mail</label>
          <input
            id="user-email"
            name="email"
            type="email"
            autocomplete="email"
            spellcheck="false"
            bind:value={email}
            placeholder="carlos@email.com"
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500"
          />
        </div>

        <div>
          <label for="user-phone" class="block text-xs font-semibold text-slate-300 mb-1">WhatsApp / Telefone</label>
          <div class="relative">
            <input
              id="user-phone"
              name="tel"
              type="tel"
              autocomplete="tel"
              bind:value={phone}
              placeholder="Ex: (11) 98765-4321 ou 5511999998888"
              class="w-full pl-8 pr-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500"
            />
            <Phone class="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-2.5 pointer-events-none" aria-hidden="true" />
          </div>
          <p class="text-[10px] text-slate-500 mt-1">Usado para receber listas de compras com menor preço.</p>
        </div>

        <div>
          <label for="user-role" class="block text-xs font-semibold text-slate-300 mb-1">Papel na Residência</label>
          <select
            id="user-role"
            bind:value={role}
            disabled={!auth.isAdmin}
            class="w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500 disabled:opacity-50"
          >
            <option value="MEMBER">Morador (Registra compras e visualiza preços)</option>
            <option value="ADMIN">Administrador (Permissão total e bloqueios)</option>
          </select>
          {#if !auth.isAdmin}
            <p class="text-[10px] text-slate-500 mt-1">Apenas administradores podem alterar o papel dos usuários.</p>
          {/if}
        </div>

        <div>
          <label for="user-password" class="block text-xs font-semibold text-slate-300 mb-1">
            {editingUser ? 'Alterar Senha (opcional)' : 'Senha de Acesso'}
          </label>
          <div class="relative">
            <input
              id="user-password"
              name="password"
              type="password"
              autocomplete="new-password"
              bind:value={password}
              placeholder={editingUser ? 'Deixe em branco para não alterar' : 'Mínimo 6 caracteres'}
              minlength="6"
              class="w-full pl-8 pr-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500"
            />
            <Lock class="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-2.5 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        {#if editingUser}
          <div class="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
            <label class="flex items-center gap-2 text-xs text-slate-300 {(!auth.isAdmin || editingUser.id === auth.currentUser?.id) ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}">
              <input
                type="checkbox"
                bind:checked={isBlocked}
                disabled={!auth.isAdmin || editingUser.id === auth.currentUser?.id}
                class="rounded border-slate-700 text-rose-500 focus-visible:ring-2 focus-visible:ring-rose-500 bg-slate-900"
              />
              <span class="font-semibold text-rose-300">Bloquear / Desativar este morador</span>
            </label>
            {#if !auth.isAdmin}
              <p class="text-[11px] text-amber-400/90 mt-1.5 flex items-center gap-1">
                <ShieldAlert class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
                Apenas administradores podem bloquear usuários.
              </p>
            {:else if editingUser.id === auth.currentUser?.id}
              <p class="text-[11px] text-amber-400/90 mt-1.5 flex items-center gap-1">
                <ShieldAlert class="w-3.5 h-3.5 shrink-0" aria-hidden="true" />
                Você não pode bloquear seu próprio usuário administrador.
              </p>
            {:else}
              <p class="text-[11px] text-slate-500 mt-1">
                Moradores bloqueados não podem acessar o sistema nem registrar compras.
              </p>
            {/if}
          </div>
        {/if}

        <div class="flex gap-2">
          {#if editingUser}
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
            class="flex-1 py-2.5 rounded-xl {editingUser ? 'bg-amber-500 hover:bg-amber-400 text-slate-950' : 'bg-emerald-500 hover:bg-emerald-400 text-slate-950'} disabled:opacity-50 font-bold text-xs shadow-md transition-colors cursor-pointer flex items-center justify-center gap-1.5 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
          >
            {#if isSubmitting}
              <div class="w-3.5 h-3.5 border-2 border-slate-950 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
              <span>Salvando…</span>
            {:else if editingUser}
              <Check class="w-3.5 h-3.5 stroke-[3]" aria-hidden="true" />
              <span>Salvar Alterações</span>
            {:else}
              <span>Cadastrar Morador</span>
            {/if}
          </button>
        </div>
      </form>
    </div>

    <!-- Users Table/List -->
    <div class="lg:col-span-2 bg-slate-900 border border-slate-800 rounded-2xl overflow-hidden">
      <div class="p-4 border-b border-slate-800 flex justify-between items-center">
        <span class="text-xs font-bold uppercase text-slate-400">Moradores Registrados ({userList.length})</span>
      </div>

      {#if loading}
        <div class="p-8 text-center text-slate-400 text-sm">Carregando moradores…</div>
      {:else if userList.length === 0}
        <div class="p-10 text-center text-slate-400 space-y-2">
          <Users class="w-8 h-8 text-slate-600 mx-auto" aria-hidden="true" />
          <p class="text-sm font-semibold text-slate-300">Nenhum morador cadastrado ainda</p>
          <p class="text-xs text-slate-500 max-w-sm mx-auto">
            Utilize o formulário ao lado para cadastrar as pessoas da casa que realizam compras.
          </p>
        </div>
      {:else}
        <div class="divide-y divide-slate-800/60">
          {#each userList as u}
            <div class="p-4 flex items-center justify-between hover:bg-slate-800/30 transition-colors {editingUser?.id === u.id ? 'bg-slate-800/50 border-l-2 border-amber-400' : ''}">
              <div class="flex items-center gap-3">
                <div class="h-10 w-10 rounded-full {u.role === 'ADMIN' ? 'bg-gradient-to-tr from-amber-500 to-orange-400' : 'bg-gradient-to-tr from-emerald-500 to-teal-400'} flex items-center justify-center font-bold text-slate-950 text-sm shrink-0">
                  {u.name.substring(0, 2).toUpperCase()}
                </div>
                <div>
                  <div class="font-semibold text-white text-sm flex flex-wrap items-center gap-2">
                    <span>{u.name}</span>

                    {#if u.role === 'ADMIN'}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold bg-amber-500/15 text-amber-300 border border-amber-500/30">
                        <Shield class="w-3 h-3" aria-hidden="true" /> Administrador
                      </span>
                    {:else}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-slate-800 text-slate-400 border border-slate-700">
                        Morador
                      </span>
                    {/if}

                    {#if u.is_blocked}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-rose-500/15 text-rose-400 border border-rose-500/30">
                        <ShieldAlert class="w-3 h-3" aria-hidden="true" /> Bloqueado
                      </span>
                    {/if}
                  </div>
                  
                  <div class="flex flex-wrap items-center gap-3 mt-1 text-xs text-slate-400">
                    {#if u.email}
                      <div class="flex items-center gap-1">
                        <Mail class="w-3 h-3 text-slate-500" aria-hidden="true" />
                        <span>{u.email}</span>
                      </div>
                    {/if}
                    {#if u.phone}
                      <div class="flex items-center gap-1 text-emerald-400/90 font-mono tabular-nums">
                        <Phone class="w-3 h-3 text-emerald-500" aria-hidden="true" />
                        <span>{u.phone}</span>
                      </div>
                    {/if}
                  </div>
                </div>
              </div>

              <div class="flex items-center gap-1.5 shrink-0 ml-2">
                <button
                  onclick={() => startEdit(u)}
                  class="p-2 rounded-lg bg-slate-800 hover:bg-amber-500/10 text-slate-400 hover:text-amber-400 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-amber-400"
                  title="Editar Morador"
                  aria-label="Editar {u.name}"
                >
                  <Pencil class="w-4 h-4" aria-hidden="true" />
                </button>

                {#if auth.isAdmin && u.id !== auth.currentUser?.id}
                  <button
                    onclick={() => handleDeleteUser(u)}
                    class="p-2 rounded-lg bg-slate-800 hover:bg-rose-500/10 text-slate-400 hover:text-rose-400 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400"
                    title="Remover ou Bloquear"
                    aria-label="Remover ou Bloquear {u.name}"
                  >
                    <Trash2 class="w-4 h-4" aria-hidden="true" />
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
