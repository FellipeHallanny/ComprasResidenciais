import type { User, LoginResponse, AuthStatusResponse } from '../types';

const TOKEN_KEY = 'compras_residenciais_jwt';

class AuthState {
  token = $state<string | null>(localStorage.getItem(TOKEN_KEY));
  currentUser = $state<User | null>(null);
  isLoading = $state(true);
  hasAdmin = $state(true);

  isAuthenticated = $derived(!!this.token && !!this.currentUser);
  isAdmin = $derived(this.currentUser?.role === 'ADMIN');

  setSession(token: string, user: User) {
    this.token = token;
    this.currentUser = user;
    localStorage.setItem(TOKEN_KEY, token);
  }

  clearSession() {
    this.token = null;
    this.currentUser = null;
    localStorage.removeItem(TOKEN_KEY);
  }

  async init() {
    this.isLoading = true;
    try {
      // 1. Verifica status geral de setup do sistema
      const statusRes = await fetch('/api/v1/auth/status');
      if (statusRes.ok) {
        const data: AuthStatusResponse = await statusRes.json();
        this.hasAdmin = data.has_admin;
      }

      // 2. Se há token no storage, valida e recupera perfil atual
      if (this.token) {
        const meRes = await fetch('/api/v1/auth/me', {
          headers: {
            Authorization: `Bearer ${this.token}`,
          },
        });
        if (meRes.ok) {
          const user: User = await meRes.json();
          this.currentUser = user;
        } else {
          this.clearSession();
        }
      }
    } catch (err) {
      console.error('Falha ao inicializar autenticação:', err);
    } finally {
      this.isLoading = false;
    }
  }

  async login(email: string, password: string): Promise<User> {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    if (!res.ok) {
      let msg = 'Falha no login';
      try {
        const err = await res.json();
        if (err.error) msg = err.error;
      } catch {
        // fallback
      }
      throw new Error(msg);
    }

    const data: LoginResponse = await res.json();
    this.setSession(data.token, data.user);
    this.hasAdmin = true;
    return data.user;
  }

  async setupInitialAdmin(name: string, email: string, password: string): Promise<User> {
    const res = await fetch('/api/v1/auth/setup-admin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, email, password }),
    });

    if (!res.ok) {
      let msg = 'Falha ao configurar administrador inicial';
      try {
        const err = await res.json();
        if (err.error) msg = err.error;
      } catch {
        // fallback
      }
      throw new Error(msg);
    }

    const data: LoginResponse = await res.json();
    this.setSession(data.token, data.user);
    this.hasAdmin = true;
    return data.user;
  }

  logout() {
    this.clearSession();
  }
}

export const auth = new AuthState();
