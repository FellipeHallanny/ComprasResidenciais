import type {
  User,
  Store,
  Purchase,
  CanonicalProduct,
  PriceComparisonView,
  MonthlyPriceOscillation,
  GlobalSummaryView
} from '../types';

const API_BASE = '/api/v1';

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
    const res = await fetch(`${API_BASE}/health`);
    return handleResponse<{ status: string; timestamp: string }>(res);
  },

  // Users
  getUsers: async (): Promise<User[]> => {
    const res = await fetch(`${API_BASE}/users`);
    return handleResponse<User[]>(res);
  },

  createUser: async (name: string, email?: string): Promise<User> => {
    const res = await fetch(`${API_BASE}/users`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, email: email || '' }),
    });
    return handleResponse<User>(res);
  },

  deleteUser: async (id: string): Promise<{ action: string; message: string; user_id: string }> => {
    const res = await fetch(`${API_BASE}/users/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<{ action: string; message: string; user_id: string }>(res);
  },

  // Stores
  getStores: async (): Promise<Store[]> => {
    const res = await fetch(`${API_BASE}/stores`);
    return handleResponse<Store[]>(res);
  },

  createStore: async (name: string, cnpj?: string): Promise<Store> => {
    const res = await fetch(`${API_BASE}/stores`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, cnpj: cnpj || '' }),
    });
    return handleResponse<Store>(res);
  },

  // Purchases
  getPurchases: async (limit = 50, offset = 0): Promise<Purchase[]> => {
    const res = await fetch(`${API_BASE}/purchases?limit=${limit}&offset=${offset}`);
    return handleResponse<Purchase[]>(res);
  },

  getPurchase: async (id: string): Promise<Purchase> => {
    const res = await fetch(`${API_BASE}/purchases/${id}`);
    return handleResponse<Purchase>(res);
  },

  uploadReceipt: async (file: File): Promise<Purchase> => {
    const formData = new FormData();
    formData.append('file', file);
    const res = await fetch(`${API_BASE}/purchases/upload`, {
      method: 'POST',
      body: formData,
    });
    return handleResponse<Purchase>(res);
  },

  processFiscalKey: async (fiscalKey: string): Promise<Purchase> => {
    const res = await fetch(`${API_BASE}/purchases/sefaz`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ fiscal_key: fiscalKey }),
    });
    return handleResponse<Purchase>(res);
  },

  createManualPurchase: async (purchase: Partial<Purchase>): Promise<Purchase> => {
    const res = await fetch(`${API_BASE}/purchases/manual`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(purchase),
    });
    return handleResponse<Purchase>(res);
  },

  confirmPurchase: async (id: string, purchase: Partial<Purchase>): Promise<Purchase> => {
    const res = await fetch(`${API_BASE}/purchases/${id}/confirm`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(purchase),
    });
    return handleResponse<Purchase>(res);
  },

  // Analytics
  searchProducts: async (query: string): Promise<CanonicalProduct[]> => {
    const res = await fetch(`${API_BASE}/analytics/search-products?q=${encodeURIComponent(query)}`);
    return handleResponse<CanonicalProduct[]>(res);
  },

  getPriceComparison: async (productId: string): Promise<PriceComparisonView> => {
    const res = await fetch(`${API_BASE}/analytics/price-comparison/${productId}`);
    return handleResponse<PriceComparisonView>(res);
  },

  getMonthlyOscillation: async (productId: string): Promise<MonthlyPriceOscillation> => {
    const res = await fetch(`${API_BASE}/analytics/price-oscillation/${productId}`);
    return handleResponse<MonthlyPriceOscillation>(res);
  },

  getSummary: async (): Promise<GlobalSummaryView> => {
    const res = await fetch(`${API_BASE}/analytics/summary`);
    return handleResponse<GlobalSummaryView>(res);
  },
};
