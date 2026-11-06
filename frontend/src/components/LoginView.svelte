<script lang="ts">
  import { auth } from '../lib/auth.svelte';
  import { Lock, Mail, User, ShieldCheck, Sparkles, AlertCircle, ArrowRight, CheckCircle2 } from '@lucide/svelte';

  interface Props {
    onsuccess: () => void;
  }

  let { onsuccess }: Props = $props();

  let isSetupMode = $derived(!auth.hasAdmin);

  // Form states
  let name = $state('');
  let email = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let isSubmitting = $state(false);
  let errorMessage = $state('');
  let successMessage = $state('');

  async function handleLogin(e: Event) {
    e.preventDefault();
    if (!email.trim() || !password) {
      errorMessage = 'Informe e-mail e senha para continuar.';
      return;
    }

    isSubmitting = true;
    errorMessage = '';
    try {
      await auth.login(email.trim(), password);
      onsuccess();
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao autenticar no sistema.';
    } finally {
      isSubmitting = false;
    }
  }

  async function handleSetupAdmin(e: Event) {
    e.preventDefault();
    if (!name.trim()) {
      errorMessage = 'Informe o nome completo do administrador.';
      return;
    }
    if (!email.trim()) {
      errorMessage = 'Informe um e-mail válido.';
      return;
    }
    if (password.length < 6) {
      errorMessage = 'A senha deve conter pelo menos 6 caracteres.';
      return;
    }
    if (password !== confirmPassword) {
      errorMessage = 'A confirmação de senha não confere.';
      return;
    }

    isSubmitting = true;
    errorMessage = '';
    try {
      await auth.setupInitialAdmin(name.trim(), email.trim(), password);
      successMessage = 'Administrador inicial criado com sucesso! Redirecionando…';
      setTimeout(() => {
        onsuccess();
      }, 600);
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao configurar administrador.';
    } finally {
      isSubmitting = false;
    }
  }
</script>

<div class="min-h-screen bg-slate-950 flex flex-col justify-center items-center p-4 relative overflow-hidden">
  <!-- Glowing background ambient blobs -->
  <div class="absolute -top-32 -left-32 w-96 h-96 bg-purple-600/20 rounded-full blur-3xl pointer-events-none" aria-hidden="true"></div>
  <div class="absolute -bottom-32 -right-32 w-96 h-96 bg-emerald-500/15 rounded-full blur-3xl pointer-events-none" aria-hidden="true"></div>

  <div class="w-full max-w-md bg-slate-900 border border-slate-800 rounded-3xl p-6 sm:p-8 shadow-2xl relative z-10">
    <!-- Header / Brand -->
    <div class="text-center mb-8">
      <div class="inline-flex p-3 rounded-2xl bg-gradient-to-tr from-purple-600 to-indigo-600 text-white shadow-lg shadow-purple-500/25 mb-3">
        {#if isSetupMode}
          <ShieldCheck class="w-8 h-8" aria-hidden="true" />
        {:else}
          <Lock class="w-8 h-8" aria-hidden="true" />
        {/if}
      </div>
      <h1 class="text-2xl font-bold text-white tracking-tight text-pretty">
        Compras Residenciais
      </h1>
      <p class="text-xs text-slate-400 mt-1">
        {#if isSetupMode}
          Configuração Inicial &bull; Criação do Administrador
        {:else}
          Controle de Gastos & Inteligência Semântica
        {/if}
      </p>
    </div>

    {#if errorMessage}
      <div class="mb-5 p-3.5 rounded-xl bg-rose-500/15 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2" aria-live="polite">
        <AlertCircle class="w-4 h-4 shrink-0" aria-hidden="true" />
        <span>{errorMessage}</span>
      </div>
    {/if}

    {#if successMessage}
      <div class="mb-5 p-3.5 rounded-xl bg-emerald-500/15 border border-emerald-500/30 text-emerald-300 text-xs flex items-center gap-2" aria-live="polite">
        <CheckCircle2 class="w-4 h-4 shrink-0" aria-hidden="true" />
        <span>{successMessage}</span>
      </div>
    {/if}

    {#if isSetupMode}
      <!-- Setup Initial Admin Form -->
      <form onsubmit={handleSetupAdmin} class="space-y-4">
        <div class="p-3 bg-purple-500/10 border border-purple-500/20 rounded-xl mb-4 text-xs text-purple-300 flex items-start gap-2">
          <Sparkles class="w-4 h-4 text-purple-400 shrink-0 mt-0.5" aria-hidden="true" />
          <span>
            Nenhum morador cadastrado ainda. Configure o usuário principal que gerenciará moradores e permissões da residência.
          </span>
        </div>

        <div>
          <label for="setup-name" class="block text-xs font-semibold text-slate-300 mb-1">Nome Completo</label>
          <div class="relative">
            <input
              id="setup-name"
              type="text"
              name="name"
              autocomplete="name"
              bind:value={name}
              placeholder="Ex: Carlos Silva"
              required
              class="w-full pl-9 pr-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-500"
            />
            <User class="w-4 h-4 text-slate-500 absolute left-3 top-3 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div>
          <label for="setup-email" class="block text-xs font-semibold text-slate-300 mb-1">E-mail de Acesso</label>
          <div class="relative">
            <input
              id="setup-email"
              type="email"
              name="email"
              autocomplete="email"
              spellcheck="false"
              bind:value={email}
              placeholder="admin@residencia.com"
              required
              class="w-full pl-9 pr-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-500"
            />
            <Mail class="w-4 h-4 text-slate-500 absolute left-3 top-3 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div>
          <label for="setup-password" class="block text-xs font-semibold text-slate-300 mb-1">Senha (mínimo 6 caracteres)</label>
          <div class="relative">
            <input
              id="setup-password"
              type="password"
              name="password"
              autocomplete="new-password"
              bind:value={password}
              placeholder="••••••••"
              required
              minlength="6"
              class="w-full pl-9 pr-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-500"
            />
            <Lock class="w-4 h-4 text-slate-500 absolute left-3 top-3 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div>
          <label for="setup-confirm-password" class="block text-xs font-semibold text-slate-300 mb-1">Confirmar Senha</label>
          <div class="relative">
            <input
              id="setup-confirm-password"
              type="password"
              name="confirm-password"
              autocomplete="new-password"
              bind:value={confirmPassword}
              placeholder="••••••••"
              required
              minlength="6"
              class="w-full pl-9 pr-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-500"
            />
            <Lock class="w-4 h-4 text-slate-500 absolute left-3 top-3 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          class="w-full py-3 mt-2 rounded-xl bg-purple-600 hover:bg-purple-500 disabled:opacity-50 text-white font-bold text-xs shadow-lg shadow-purple-600/30 transition-colors cursor-pointer flex items-center justify-center gap-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-400"
        >
          {#if isSubmitting}
            <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
            <span>Configurando…</span>
          {:else}
            <span>Criar Administrador e Iniciar</span>
            <ArrowRight class="w-4 h-4" aria-hidden="true" />
          {/if}
        </button>
      </form>
    {:else}
      <!-- Regular Login Form -->
      <form onsubmit={handleLogin} class="space-y-4">
        <div>
          <label for="login-email" class="block text-xs font-semibold text-slate-300 mb-1">E-mail</label>
          <div class="relative">
            <input
              id="login-email"
              type="email"
              name="email"
              autocomplete="email"
              spellcheck="false"
              bind:value={email}
              placeholder="seu.email@exemplo.com"
              required
              class="w-full pl-9 pr-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-500"
            />
            <Mail class="w-4 h-4 text-slate-500 absolute left-3 top-3 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div>
          <div class="flex items-center justify-between mb-1">
            <label for="login-password" class="block text-xs font-semibold text-slate-300">Senha</label>
          </div>
          <div class="relative">
            <input
              id="login-password"
              type="password"
              name="password"
              autocomplete="current-password"
              bind:value={password}
              placeholder="••••••••"
              required
              class="w-full pl-9 pr-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-500"
            />
            <Lock class="w-4 h-4 text-slate-500 absolute left-3 top-3 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          class="w-full py-3 mt-2 rounded-xl bg-purple-600 hover:bg-purple-500 disabled:opacity-50 text-white font-bold text-xs shadow-lg shadow-purple-600/30 transition-colors cursor-pointer flex items-center justify-center gap-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-400"
        >
          {#if isSubmitting}
            <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
            <span>Entrando…</span>
          {:else}
            <span>Acessar Painel</span>
            <ArrowRight class="w-4 h-4" aria-hidden="true" />
          {/if}
        </button>
      </form>

      <div class="mt-6 text-center">
        <p class="text-[11px] text-slate-500">
          Novo morador na casa? Solicite seu acesso ao administrador da residência.
        </p>
      </div>
    {/if}
  </div>
</div>
