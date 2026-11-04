import type {
  User,
  Store,
  Purchase,
  CanonicalProduct,
  CatalogItemView,
  PriceComparisonView,
  MonthlyPriceOscillation,
  GlobalSummaryView
} from '../types';
import { auth } from './auth.svelte';

const API_BASE = '/api/v1';

async function authFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const headers = new Headers(init?.headers || {});
  if (auth.token) {
    headers.set('Authorization', `Bearer ${auth.token}`);
  }

  const res = await fetch(input, {
    ...init,
    headers,
  });

  if (res.status === 401) {
    // Se o token expirou ou usuário foi bloqueado, desconecta
    auth.logout();
  }

  return res;
}

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let errorMsg = `Erro ${res.status}: ${res.statusText}`;
    try {
      const errJson = await res.json();
      if (errJson.error) {
        errorMsg = errJson.error;
      }
    } catch {
      // ignore
    }
    throw new Error(errorMsg);
  }
  return res.json();
}

export const api = {
  // Health
  checkHealth: async () => {
    const res = await authFetch(`${API_BASE}/health`);
    return handleResponse<{ status: string; timestamp: string }>(res);
  },

  // Auth
  getMe: async (): Promise<User> => {
    const res = await authFetch(`${API_BASE}/auth/me`);
    return handleResponse<User>(res);
  },

  // Users
  getUsers: async (): Promise<User[]> => {
    const res = await authFetch(`${API_BASE}/users`);
    const data = await handleResponse<User[]>(res);
    return Array.isArray(data) ? data : [];
  },

  createUser: async (name: string, email?: string, phone?: string, password?: string, role?: string): Promise<User> => {
    const res = await authFetch(`${API_BASE}/users`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name,
        email: email || '',
        phone: phone || '',
        password: password || '',
        role: role || 'MEMBER',
      }),
    });
    return handleResponse<User>(res);
  },

  updateUser: async (
    id: string,
    data: {
      name: string;
      email?: string;
      phone?: string;
      role?: string;
      is_blocked?: boolean;
      new_password?: string;
    }
  ): Promise<User> => {
    const res = await authFetch(`${API_BASE}/users/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: data.name,
        email: data.email || '',
        phone: data.phone || '',
        role: data.role,
        is_blocked: !!data.is_blocked,
        new_password: data.new_password,
      }),
    });
    return handleResponse<User>(res);
  },

  deleteUser: async (id: string): Promise<{ action: string; message: string; user_id: string }> => {
    const res = await authFetch(`${API_BASE}/users/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<{ action: string; message: string; user_id: string }>(res);
  },

  // Stores
  getStores: async (): Promise<Store[]> => {
    const res = await authFetch(`${API_BASE}/stores`);
    const data = await handleResponse<Store[]>(res);
    return Array.isArray(data) ? data : [];
  },

  createStore: async (name: string, cnpj?: string, address?: string): Promise<Store> => {
    const res = await authFetch(`${API_BASE}/stores`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, cnpj: cnpj || '', address: address || '' }),
    });
    return handleResponse<Store>(res);
  },

  updateStore: async (id: string, data: { name: string; cnpj?: string; address?: string }): Promise<Store> => {
    const res = await authFetch(`${API_BASE}/stores/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: data.name,
        cnpj: data.cnpj || '',
        address: data.address || '',
      }),
    });
    return handleResponse<Store>(res);
  },

  deleteStore: async (id: string): Promise<{ message: string; id: string }> => {
    const res = await authFetch(`${API_BASE}/stores/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<{ message: string; id: string }>(res);
  },

  // Purchases
  getPurchases: async (limit = 50, offset = 0): Promise<Purchase[]> => {
    const res = await authFetch(`${API_BASE}/purchases?limit=${limit}&offset=${offset}`);
    const data = await handleResponse<Purchase[]>(res);
    return Array.isArray(data) ? data : [];
  },

  getPurchase: async (id: string): Promise<Purchase> => {
    const res = await authFetch(`${API_BASE}/purchases/${id}`);
    return handleResponse<Purchase>(res);
  },

  uploadReceipt: async (file: File): Promise<Purchase> => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await authFetch(`${API_BASE}/purchases/upload`, {
      method: 'POST',
      body: formData,
    });
    return handleResponse<Purchase>(res);
  },

  processFiscalKey: async (fiscalKey: string): Promise<Purchase> => {
    const res = await authFetch(`${API_BASE}/purchases/sefaz`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ fiscal_key: fiscalKey }),
    });
    return handleResponse<Purchase>(res);
  },

  createManualPurchase: async (purchase: Partial<Purchase>): Promise<Purchase> => {
    const res = await authFetch(`${API_BASE}/purchases/manual`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(purchase),
    });
    return handleResponse<Purchase>(res);
  },

  confirmPurchase: async (id: string, purchase: Partial<Purchase>): Promise<Purchase> => {
    const res = await authFetch(`${API_BASE}/purchases/${id}/confirm`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(purchase),
    });
    return handleResponse<Purchase>(res);
  },

  // Analytics & Catalog
  searchProducts: async (query: string): Promise<CanonicalProduct[]> => {
    const res = await authFetch(`${API_BASE}/analytics/search-products?q=${encodeURIComponent(query)}`);
    const data = await handleResponse<CanonicalProduct[]>(res);
    return Array.isArray(data) ? data : [];
  },

  getCatalogWithPrices: async (): Promise<CatalogItemView[]> => {
    const res = await authFetch(`${API_BASE}/analytics/catalog-prices`);
    const data = await handleResponse<CatalogItemView[]>(res);
    return Array.isArray(data) ? data : [];
  },

  getPriceComparison: async (productId: string): Promise<PriceComparisonView> => {
    const res = await authFetch(`${API_BASE}/analytics/price-comparison/${productId}`);
    return handleResponse<PriceComparisonView>(res);
  },

  getMonthlyOscillation: async (productId: string): Promise<MonthlyPriceOscillation> => {
    const res = await authFetch(`${API_BASE}/analytics/price-oscillation/${productId}`);
    return handleResponse<MonthlyPriceOscillation>(res);
  },

  getSummary: async (): Promise<GlobalSummaryView> => {
    const res = await authFetch(`${API_BASE}/analytics/summary`);
    return handleResponse<GlobalSummaryView>(res);
  },

  // Shopping List
  sendShoppingListEmail: async (toEmail: string, subject: string, content: string): Promise<{ status: string; message: string }> => {
    const res = await authFetch(`${API_BASE}/shopping-list/send-email`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ to_email: toEmail, subject, content }),
    });
    return handleResponse<{ status: string; message: string }>(res);
  },

  // Admin / Reset
  resetDatabase: async (): Promise<{ status: string; message: string }> => {
    const res = await authFetch(`${API_BASE}/admin/reset-database`, {
      method: 'POST',
    });
    return handleResponse<{ status: string; message: string }>(res);
  },
};
