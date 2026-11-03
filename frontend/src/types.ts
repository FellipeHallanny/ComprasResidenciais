export interface User {
  id: string;
  name: string;
  email?: string;
  role: 'ADMIN' | 'MEMBER';
  is_blocked: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface AuthStatusResponse {
  has_admin: boolean;
}

export interface Store {
  id: string;
  name: string;
  cnpj?: string;
  address?: string;
  created_at: string;
  updated_at: string;
}

export type PurchaseStatus = 'DRAFT' | 'CONFIRMED';
export type PurchaseOrigin = 'MANUAL' | 'LLM_UPLOAD' | 'SEFAZ';

export interface PurchaseItem {
  id?: string;
  purchase_id?: string;
  raw_description: string;
  quantity: number;
  package_size: number;
  package_unit: string;
  total_units?: number;
  total_price: number;
  unit_price?: number;
  normalized_price?: number;
  canonical_product_id?: string;
  created_at?: string;
}

export interface Purchase {
  id: string;
  user_id: string;
  store_id: string;
  total_amount: number;
  purchased_at: string;
  status: PurchaseStatus;
  origin: PurchaseOrigin;
  fiscal_key?: string;
  items?: PurchaseItem[];
  created_at: string;
  updated_at: string;
}

export interface CanonicalProduct {
  id: string;
  name: string;
  category: string;
  base_unit: string;
  created_at: string;
  updated_at: string;
}

export interface RecentItemPurchase {
  purchase_id: string;
  purchased_at: string;
  store_name: string;
  buyer_name: string;
  raw_description: string;
  package_info: string;
  normalized_price: number;
  total_price: number;
}

export interface PriceComparisonView {
  canonical_product_id: string;
  product_name: string;
  base_unit: string;
  lowest_price_per_unit: number;
  highest_price_per_unit: number;
  average_price_per_unit: number;
  cheapest_store_name: string;
  cheapest_purchase_date: string;
  max_savings_percentage: number;
  best_month_period: string;
  recent_purchases: RecentItemPurchase[];
}

export interface PeriodMetric {
  period_label: string;
  average_price: number;
  total_count: number;
}

export interface MonthlyPriceOscillation {
  canonical_product_id: string;
  product_name: string;
  base_unit: string;
  periods: PeriodMetric[];
}

export interface StoreShareMetric {
  store_name: string;
  total_spent: number;
  percentage: number;
}

export interface GlobalSummaryView {
  current_month_total: number;
  total_purchases: number;
  spending_by_user: Record<string, number>;
  top_stores: StoreShareMetric[];
}
