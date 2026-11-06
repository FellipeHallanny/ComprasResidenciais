<script lang="ts">
  import { auth } from '../lib/auth.svelte';
  import { Lock, Mail, User, ShieldCheck, AlertCircle, ArrowRight, CheckCircle2 } from '@lucide/svelte';

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

<div class="min-h-screen bg-[#0a0d14] flex flex-col justify-center items-center p-4">
  <div class="w-full max-w-sm sm:max-w-md bg-[#0e131f] border border-[#1b2234] rounded-xl p-6 sm:p-8 shadow-xl">
    <!-- Header / Brand -->
    <div class="text-center mb-6">
      <div class="inline-flex p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 mb-3.5 shadow-xs">
        {#if isSetupMode}
          <ShieldCheck class="w-6 h-6 stroke-[2.2]" aria-hidden="true" />
        {:else}
          <Lock class="w-6 h-6 stroke-[2.2]" aria-hidden="true" />
        {/if}
      </div>
      <h1 class="text-xl sm:text-2xl font-bold text-white tracking-tight text-pretty">
        Compras Residenciais
      </h1>
      <p class="text-xs text-slate-400 mt-1">
        {#if isSetupMode}
          Configuração inicial do administrador
        {:else}
          Controle de gastos e inteligência de compras
        {/if}
      </p>
    </div>

    {#if errorMessage}
      <div class="mb-5 p-3 rounded-lg bg-rose-500/10 border border-rose-500/25 text-rose-300 text-xs flex items-center gap-2" aria-live="polite">
        <AlertCircle class="w-4 h-4 shrink-0" aria-hidden="true" />
        <span>{errorMessage}</span>
      </div>
    {/if}

    {#if successMessage}
      <div class="mb-5 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/25 text-emerald-300 text-xs flex items-center gap-2" aria-live="polite">
        <CheckCircle2 class="w-4 h-4 shrink-0" aria-hidden="true" />
        <span>{successMessage}</span>
      </div>
    {/if}

    {#if isSetupMode}
      <!-- Setup Initial Admin Form -->
      <form onsubmit={handleSetupAdmin} class="space-y-4">
        <div class="p-3 bg-[#121927] border border-[#1f2a40] rounded-lg mb-4 text-xs text-slate-300">
          <span>
            Nenhum morador cadastrado ainda. Configure o usuário principal que gerenciará moradores e permissões da residência.
          </span>
        </div>

        <div>
          <label for="setup-name" class="block text-xs font-medium text-slate-300 mb-1.5">Nome Completo</label>
          <div class="relative">
            <input
              id="setup-name"
              type="text"
              name="name"
              autocomplete="name"
              bind:value={name}
              placeholder="Ex: Carlos Silva"
              required
              class="w-full pl-9 pr-3 py-2 bg-[#090d16] border border-[#1b2234] focus:border-emerald-500/60 rounded-lg text-slate-100 text-xs focus:outline-none transition-colors"
            />
            <User class="w-4 h-4 text-slate-500 absolute left-2.5 top-2.5 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div>
          <label for="setup-email" class="block text-xs font-medium text-slate-300 mb-1.5">E-mail de Acesso</label>
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
              class="w-full pl-9 pr-3 py-2 bg-[#090d16] border border-[#1b2234] focus:border-emerald-500/60 rounded-lg text-slate-100 text-xs focus:outline-none transition-colors"
            />
            <Mail class="w-4 h-4 text-slate-500 absolute left-2.5 top-2.5 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div>
          <label for="setup-password" class="block text-xs font-medium text-slate-300 mb-1.5">Senha (mínimo 6 caracteres)</label>
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
              class="w-full pl-9 pr-3 py-2 bg-[#090d16] border border-[#1b2234] focus:border-emerald-500/60 rounded-lg text-slate-100 text-xs focus:outline-none transition-colors"
            />
            <Lock class="w-4 h-4 text-slate-500 absolute left-2.5 top-2.5 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div>
          <label for="setup-confirm-password" class="block text-xs font-medium text-slate-300 mb-1.5">Confirmar Senha</label>
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
              class="w-full pl-9 pr-3 py-2 bg-[#090d16] border border-[#1b2234] focus:border-emerald-500/60 rounded-lg text-slate-100 text-xs focus:outline-none transition-colors"
            />
            <Lock class="w-4 h-4 text-slate-500 absolute left-2.5 top-2.5 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          class="w-full py-2.5 mt-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-semibold text-xs shadow-xs transition-all cursor-pointer flex items-center justify-center gap-2 active:translate-y-px focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
        >
          {#if isSubmitting}
            <div class="w-4 h-4 border-2 border-slate-950 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
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
          <label for="login-email" class="block text-xs font-medium text-slate-300 mb-1.5">E-mail</label>
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
              class="w-full pl-9 pr-3 py-2 bg-[#090d16] border border-[#1b2234] focus:border-emerald-500/60 rounded-lg text-slate-100 text-xs focus:outline-none transition-colors"
            />
            <Mail class="w-4 h-4 text-slate-500 absolute left-2.5 top-2.5 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <div>
          <div class="flex items-center justify-between mb-1.5">
            <label for="login-password" class="block text-xs font-medium text-slate-300">Senha</label>
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
              class="w-full pl-9 pr-3 py-2 bg-[#090d16] border border-[#1b2234] focus:border-emerald-500/60 rounded-lg text-slate-100 text-xs focus:outline-none transition-colors"
            />
            <Lock class="w-4 h-4 text-slate-500 absolute left-2.5 top-2.5 pointer-events-none" aria-hidden="true" />
          </div>
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          class="w-full py-2.5 mt-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-semibold text-xs shadow-xs transition-all cursor-pointer flex items-center justify-center gap-2 active:translate-y-px focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400"
        >
          {#if isSubmitting}
            <div class="w-4 h-4 border-2 border-slate-950 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
            <span>Entrando…</span>
          {:else}
            <span>Acessar Painel</span>
            <ArrowRight class="w-4 h-4" aria-hidden="true" />
          {/if}
        </button>
      </form>

      <div class="mt-6 text-center">
        <p class="text-[11px] text-slate-500">
          Novo morador? Solicite seu acesso ao administrador da casa.
        </p>
      </div>
    {/if}
  </div>
</div>
