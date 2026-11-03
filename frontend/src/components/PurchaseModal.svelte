<script lang="ts">
  import {
    X,
    Upload,
    FileText,
    Key,
    Edit3,
    Plus,
    Trash2,
    Check,
    AlertCircle,
    Sparkles,
    Calendar,
    Store,
    User as UserIcon,
    ArrowRight
  } from '@lucide/svelte';
  import type { User, Store as StoreType, Purchase, PurchaseItem, CanonicalProduct } from '../types';
  import { api } from '../lib/api';

  interface Props {
    users: User[];
    stores: StoreType[];
    onclose: () => void;
    onsuccess: (purchase: Purchase) => void;
  }

  let { users, stores, onclose, onsuccess }: Props = $props();

  const userList = $derived(users ?? []);
  const storeList = $derived(stores ?? []);

  type TabMode = 'upload' | 'sefaz' | 'manual';
  let activeTab = $state<TabMode>('upload');

  // Step: 'input' or 'review'
  let step = $state<'input' | 'review'>('input');
  let draftPurchase = $state<Purchase | null>(null);
  let errorMessage = $state('');
  let isSubmitting = $state(false);

  // --- Upload State ---
  let selectedFile = $state<File | null>(null);
  let filePreview = $state<string | null>(null);

  // --- Sefaz State ---
  let fiscalKey = $state('');

  // --- Manual / Review State ---
  let selectedUserId = $state('');
  let selectedStoreId = $state('');
  let customStoreName = $state('');
  let purchaseDate = $state(new Date().toISOString().split('T')[0]);

  // Items for manual entry or draft review
  let editableItems = $state<PurchaseItem[]>([
    {
      raw_description: '',
      quantity: 1,
      package_size: 1,
      package_unit: 'kg',
      total_price: 0,
    }
  ]);

  const units = ['kg', 'g', 'L', 'ml', 'un'];

  // --- pgvector Semantic Search for Items ---
  let activeSearchIndex = $state<number | null>(null);
  let searchResults = $state<CanonicalProduct[]>([]);
  let isSearchingProduct = $state(false);
  let searchDebounceTimer: any = null;

  function handleItemDescriptionInput(idx: number, val: string) {
    if (!editableItems[idx]) return;
    editableItems[idx].raw_description = val;
    editableItems[idx].canonical_product_id = undefined;

    clearTimeout(searchDebounceTimer);
    if (!val.trim()) {
      searchResults = [];
      activeSearchIndex = null;
      return;
    }

    activeSearchIndex = idx;
    isSearchingProduct = true;
    searchDebounceTimer = setTimeout(async () => {
      try {
        const results = await api.searchProducts(val.trim());
        if (activeSearchIndex === idx) {
          searchResults = results;
        }
      } catch (err) {
        console.error('Erro na busca sem�ntica pgvector:', err);
      } finally {
        isSearchingProduct = false;
      }
    }, 250);
  }

  async function triggerItemSearch(idx: number) {
    if (activeSearchIndex === idx) {
      activeSearchIndex = null;
      return;
    }
    activeSearchIndex = idx;
    const term = editableItems[idx]?.raw_description?.trim() || '';
    isSearchingProduct = true;
    try {
      const results = await api.searchProducts(term);
      if (activeSearchIndex === idx) {
        searchResults = results;
      }
    } catch (err) {
      console.error('Erro na busca sem�ntica pgvector:', err);
    } finally {
      isSearchingProduct = false;
    }
  }

  function selectProductForIndex(idx: number, product: CanonicalProduct) {
    if (editableItems[idx]) {
      editableItems[idx].raw_description = product.name;
      if (product.base_unit) {
        editableItems[idx].package_unit = product.base_unit.toLowerCase();
      }
      editableItems[idx].canonical_product_id = product.id;
    }
    activeSearchIndex = null;
  }

  function handleWindowClick(e: MouseEvent) {
    const target = e.target as HTMLElement | null;
    if (target && !target.closest('.item-search-container')) {
      activeSearchIndex = null;
    }
  }

  // Calculate live normalized price for an item
  function calcNormalized(item: PurchaseItem): number {
    let baseMultiplier = 1;
    const u = (item.package_unit || 'un').toLowerCase();
    if (u === 'g' || u === 'ml') {
      baseMultiplier = 0.001;
    }
    const totalBaseUnits = (item.quantity || 1) * (item.package_size || 1) * baseMultiplier;
    return totalBaseUnits > 0 ? (item.total_price || 0) / totalBaseUnits : (item.total_price || 0);
  }

  const reviewTotal = $derived.by(() => {
    return editableItems.reduce((acc, it) => acc + (Number(it.total_price) || 0), 0);
  });

  // Handle file select
  function handleFileChange(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files && input.files[0]) {
      const file = input.files[0];
      selectedFile = file;
      errorMessage = '';

      if (file.type.startsWith('image/')) {
        const reader = new FileReader();
        reader.onload = () => {
          filePreview = reader.result as string;
        };
        reader.readAsDataURL(file);
      } else {
        filePreview = null;
      }
    }
  }

  // Submit Upload
  async function submitUpload() {
    if (!selectedFile) {
      errorMessage = 'Selecione uma imagem ou PDF de cupom fiscal.';
      return;
    }
    isSubmitting = true;
    errorMessage = '';
    try {
      const draft = await api.uploadReceipt(selectedFile);
      draftPurchase = draft;
      if (draft.user_id) selectedUserId = draft.user_id;
      if (draft.store_id) selectedStoreId = draft.store_id;
      if (draft.purchased_at) {
        purchaseDate = new Date(draft.purchased_at).toISOString().split('T')[0];
      }
      if (draft.items && draft.items.length > 0) {
        editableItems = draft.items.map(it => ({ ...it }));
      }
      step = 'review';
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao processar nota com IA Gemini.';
    } finally {
      isSubmitting = false;
    }
  }

  // Submit SEFAZ Key
  async function submitSefaz() {
    const cleaned = fiscalKey.replace(/\s+/g, '');
    if (cleaned.length !== 44) {
      errorMessage = 'A chave fiscal deve conter exatamente 44 d�gitos num�ricos.';
      return;
    }
    isSubmitting = true;
    errorMessage = '';
    try {
      const draft = await api.processFiscalKey(cleaned);
      draftPurchase = draft;
      if (draft.user_id) selectedUserId = draft.user_id;
      if (draft.store_id) selectedStoreId = draft.store_id;
      if (draft.items && draft.items.length > 0) {
        editableItems = draft.items.map(it => ({ ...it }));
      }
      step = 'review';
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao consultar chave fiscal no SEFAZ.';
    } finally {
      isSubmitting = false;
    }
  }

  // Prepare manual purchase
  function proceedToManualReview() {
    if (!selectedUserId) {
      errorMessage = 'Selecione o morador respons�vel pela compra.';
      return;
    }
    if (!selectedStoreId && !customStoreName.trim()) {
      errorMessage = 'Selecione ou digite o nome do estabelecimento.';
      return;
    }
    errorMessage = '';
    step = 'review';
  }

  function addItem() {
    editableItems.push({
      raw_description: '',
      quantity: 1,
      package_size: 1,
      package_unit: 'kg',
      total_price: 0,
    });
  }

  function removeItem(index: number) {
    if (editableItems.length > 1) {
      editableItems.splice(index, 1);
      if (activeSearchIndex === index) {
        activeSearchIndex = null;
      }
    }
  }

  // Final Confirmation
  async function handleConfirmAndSave() {
    if (!selectedUserId) {
      errorMessage = 'Selecione o morador comprador (obrigat�rio).';
      return;
    }

    if (editableItems.length === 0) {
      errorMessage = 'A compra precisa conter pelo menos 1 item.';
      return;
    }

    for (const it of editableItems) {
      if (!it.raw_description.trim()) {
        errorMessage = 'Todos os itens precisam de uma descri��o v�lida.';
        return;
      }
      if (it.total_price <= 0) {
        errorMessage = `Pre�o inv�lido para o item "${it.raw_description}".`;
        return;
      }
    }

    isSubmitting = true;
    errorMessage = '';

    try {
      let finalStoreId = selectedStoreId;
      if (!finalStoreId && customStoreName.trim()) {
        const newStore = await api.createStore(customStoreName.trim());
        finalStoreId = newStore.id;
      }

      if (draftPurchase?.id) {
        const payload: Partial<Purchase> = {
          user_id: selectedUserId,
          store_id: finalStoreId || draftPurchase.store_id,
          purchased_at: new Date(purchaseDate).toISOString(),
          total_amount: reviewTotal,
          items: editableItems,
        };
        const confirmed = await api.confirmPurchase(draftPurchase.id, payload);
        onsuccess(confirmed);
      } else {
        const payload: Partial<Purchase> = {
          user_id: selectedUserId,
          store_id: finalStoreId,
          purchased_at: new Date(purchaseDate).toISOString(),
          total_amount: reviewTotal,
          items: editableItems,
        };
        const created = await api.createManualPurchase(payload);
        onsuccess(created);
      }
    } catch (err: any) {
      errorMessage = err.message || 'Falha ao confirmar compra.';
    } finally {
      isSubmitting = false;
    }
  }
</script>

<svelte:window onclick={handleWindowClick} />

<div class="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto">
  <div class="bg-slate-900 border border-slate-700 rounded-3xl w-full max-w-4xl shadow-2xl overflow-hidden my-8">
    
    <!-- Modal Header -->
    <div class="px-6 py-4 border-b border-slate-800 flex items-center justify-between bg-slate-900/50">
      <div class="flex items-center gap-3">
        <div class="h-9 w-9 rounded-xl bg-emerald-500/10 text-emerald-400 flex items-center justify-center border border-emerald-500/20">
          {#if step === 'review'}
            <Check class="w-5 h-5 stroke-[2.5]" />
          {:else}
            <FileText class="w-5 h-5" />
          {/if}
        </div>
        <div>
          <h2 class="text-lg font-bold text-white">
            {step === 'review' ? 'Confer�ncia do Rascunho de Compra' : 'Registrar Nova Compra'}
          </h2>
          <p class="text-xs text-slate-400">
            {step === 'review' ? 'Revise os itens extra�dos e vincule o morador antes de salvar' : 'Selecione o m�todo de captura da nota fiscal'}
          </p>
        </div>
      </div>

      <button
        onclick={onclose}
        class="text-slate-400 hover:text-slate-200 p-2 rounded-lg hover:bg-slate-800 transition cursor-pointer"
        title="Fechar"
      >
        <X class="w-5 h-5" />
      </button>
    </div>

    <!-- Empty Users Notice -->
    {#if userList.length === 0}
      <div class="mx-6 mt-4 p-3.5 rounded-xl bg-amber-500/15 border border-amber-500/30 text-amber-300 text-xs flex items-center gap-2">
        <AlertCircle class="w-4 h-4 shrink-0" />
        <span>Nenhum morador cadastrado ainda. Recomendamos cadastrar pelo menos um morador na aba <strong>Moradores</strong> antes de registrar compras.</span>
      </div>
    {/if}

    <!-- Error Banner -->
    {#if errorMessage}
      <div class="mx-6 mt-4 p-3.5 rounded-xl bg-rose-500/15 border border-rose-500/30 text-rose-300 text-xs flex items-center gap-2">
        <AlertCircle class="w-4 h-4 shrink-0" />
        <span>{errorMessage}</span>
      </div>
    {/if}

    <div class="p-6">
      {#if step === 'input'}
        <!-- Input Tabs -->
        <div class="flex border-b border-slate-800 mb-6">
          <button
            onclick={() => { activeTab = 'upload'; errorMessage = ''; }}
            class="flex items-center gap-2 px-5 py-3 font-semibold text-sm border-b-2 transition cursor-pointer {activeTab === 'upload' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-slate-400 hover:text-slate-200'}"
          >
            <Upload class="w-4 h-4" />
            <span>Upload de Cupom (IA Gemini)</span>
          </button>

          <button
            onclick={() => { activeTab = 'sefaz'; errorMessage = ''; }}
            class="flex items-center gap-2 px-5 py-3 font-semibold text-sm border-b-2 transition cursor-pointer {activeTab === 'sefaz' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-slate-400 hover:text-slate-200'}"
          >
            <Key class="w-4 h-4" />
            <span>Chave SEFAZ / NFC-e</span>
          </button>

          <button
            onclick={() => { activeTab = 'manual'; errorMessage = ''; }}
            class="flex items-center gap-2 px-5 py-3 font-semibold text-sm border-b-2 transition cursor-pointer {activeTab === 'manual' ? 'border-emerald-500 text-emerald-400' : 'border-transparent text-slate-400 hover:text-slate-200'}"
          >
            <Edit3 class="w-4 h-4" />
            <span>Cadastro Manual (Feiras / Sem Nota)</span>
          </button>
        </div>

        <!-- Tab 1: Upload de Cupom -->
        {#if activeTab === 'upload'}
          <div class="space-y-4">
            <div class="border-2 border-dashed border-slate-700 hover:border-emerald-500/60 rounded-2xl p-8 text-center transition bg-slate-950/40 relative">
              <input
                type="file"
                accept="image/*,application/pdf"
                onchange={handleFileChange}
                class="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
              />

              {#if filePreview}
                <div class="max-w-xs mx-auto mb-4 rounded-xl overflow-hidden border border-slate-700 shadow-md">
                  <img src={filePreview} alt="Preview cupom" class="w-full h-44 object-cover" />
                </div>
              {:else}
                <div class="w-14 h-14 rounded-2xl bg-emerald-500/10 text-emerald-400 flex items-center justify-center mx-auto mb-3 border border-emerald-500/20">
                  <Upload class="w-7 h-7" />
                </div>
              {/if}

              <h4 class="text-sm font-semibold text-white">
                {selectedFile ? selectedFile.name : 'Arraste ou clique para selecionar a foto ou PDF do cupom'}
              </h4>
              <p class="text-xs text-slate-400 mt-1">
                Suporta fotos tiradas pelo celular (JPG, PNG) ou notas fiscais eletr�nicas em PDF.
              </p>
            </div>

            <div class="flex items-center justify-between pt-4">
              <div class="flex items-center gap-2 text-xs text-slate-400">
                <Sparkles class="w-4 h-4 text-emerald-400" />
                <span>Extra��o multimodal autom�tica de itens, quantidades e valores via Gemini</span>
              </div>

              <button
                onclick={submitUpload}
                disabled={!selectedFile || isSubmitting}
                class="px-6 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-bold text-sm shadow-md transition flex items-center gap-2 cursor-pointer"
              >
                {#if isSubmitting}
                  <div class="w-4 h-4 border-2 border-slate-950 border-t-transparent rounded-full animate-spin"></div>
                  <span>Processando Nota com IA...</span>
                {:else}
                  <span>Extrair Itens & Revisar</span>
                  <ArrowRight class="w-4 h-4" />
                {/if}
              </button>
            </div>
          </div>
        {/if}

        <!-- Tab 2: Chave SEFAZ -->
        {#if activeTab === 'sefaz'}
          <div class="space-y-4 max-w-xl mx-auto py-4">
            <div>
              <label for="sefaz-key-input" class="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-2">
                Chave de Acesso da NFC-e / NF-e (44 d�gitos)
              </label>
              <input
                id="sefaz-key-input"
                type="text"
                maxlength="54"
                bind:value={fiscalKey}
                placeholder="Ex: 3524 0100 0000 0000 0000 6500 1000 0000 0010 0000 0000"
                class="w-full px-4 py-3 bg-slate-950 border border-slate-700 rounded-xl text-slate-100 font-mono text-sm tracking-widest placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
              />
              <p class="text-xs text-slate-500 mt-1.5">
                Localizada no topo ou rodap� do cupom fiscal pr�ximo ao QR-Code.
              </p>
            </div>

            <div class="pt-2 flex justify-end">
              <button
                onclick={submitSefaz}
                disabled={fiscalKey.replace(/\s+/g, '').length !== 44 || isSubmitting}
                class="px-6 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-bold text-sm shadow-md transition flex items-center gap-2 cursor-pointer"
              >
                {#if isSubmitting}
                  <div class="w-4 h-4 border-2 border-slate-950 border-t-transparent rounded-full animate-spin"></div>
                  <span>Consultando SEFAZ...</span>
                {:else}
                  <span>Consultar & Revisar</span>
                  <ArrowRight class="w-4 h-4" />
                {/if}
              </button>
            </div>
          </div>
        {/if}

        <!-- Tab 3: Cadastro Manual -->
        {#if activeTab === 'manual'}
          <div class="space-y-5">
            <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <!-- Comprador -->
              <div>
                <label for="manual-buyer-select" class="block text-xs font-semibold text-slate-300 mb-1.5">Morador Comprador *</label>
                <select
                  id="manual-buyer-select"
                  bind:value={selectedUserId}
                  class="w-full px-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
                >
                  <option value="">Selecione o morador...</option>
                  {#each userList as u}
                    <option value={u.id}>{u.name}</option>
                  {/each}
                </select>
              </div>

              <!-- Estabelecimento -->
              <div>
                <label for="manual-store-select" class="block text-xs font-semibold text-slate-300 mb-1.5">Estabelecimento *</label>
                <select
                  id="manual-store-select"
                  bind:value={selectedStoreId}
                  class="w-full px-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
                >
                  <option value="">Selecione da lista...</option>
                  {#each storeList as s}
                    <option value={s.id}>{s.name}</option>
                  {/each}
                </select>
                {#if !selectedStoreId}
                  <input
                    type="text"
                    bind:value={customStoreName}
                    placeholder="Ou digite o nome da loja/feira..."
                    class="mt-2 w-full px-3 py-2 bg-slate-950 border border-slate-700 rounded-xl text-slate-200 text-xs focus:outline-none focus:ring-1 focus:ring-emerald-500"
                  />
                {/if}
              </div>

              <!-- Data -->
              <div>
                <label for="manual-date-input" class="block text-xs font-semibold text-slate-300 mb-1.5">Data da Compra</label>
                <input
                  id="manual-date-input"
                  type="date"
                  bind:value={purchaseDate}
                  class="w-full px-3 py-2.5 bg-slate-950 border border-slate-700 rounded-xl text-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
                />
              </div>
            </div>

            <div class="pt-2 flex justify-end">
              <button
                onclick={proceedToManualReview}
                class="px-6 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-sm shadow-md transition flex items-center gap-2 cursor-pointer"
              >
                <span>Avan�ar para Lista de Itens</span>
                <ArrowRight class="w-4 h-4" />
              </button>
            </div>
          </div>
        {/if}

      {:else}
        <!-- Step 2: Tela de Confer�ncia do Rascunho (Draft Review) -->
        <div class="space-y-5">
          <!-- Metadata Bar -->
          <div class="p-4 rounded-2xl bg-slate-950 border border-slate-800 grid grid-cols-1 sm:grid-cols-3 gap-4">
            <!-- Morador (Mandatory) -->
            <div>
              <label for="review-buyer-select" class="block text-xs font-semibold text-emerald-400 mb-1 flex items-center gap-1.5">
                <UserIcon class="w-3.5 h-3.5" /> Morador Comprador *
              </label>
              <select
                id="review-buyer-select"
                bind:value={selectedUserId}
                class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-xl text-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
              >
                <option value="">Selecione o morador da casa...</option>
                {#each userList as u}
                  <option value={u.id}>{u.name}</option>
                {/each}
              </select>
            </div>

            <!-- Estabelecimento -->
            <div>
              <label for="review-store-select" class="block text-xs font-semibold text-slate-400 mb-1 flex items-center gap-1.5">
                <Store class="w-3.5 h-3.5" /> Estabelecimento
              </label>
              <select
                id="review-store-select"
                bind:value={selectedStoreId}
                class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-xl text-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
              >
                <option value="">Selecione ou mantenha detectado...</option>
                {#each storeList as s}
                  <option value={s.id}>{s.name}</option>
                {/each}
              </select>
            </div>

            <!-- Data -->
            <div>
              <label for="review-date-input" class="block text-xs font-semibold text-slate-400 mb-1 flex items-center gap-1.5">
                <Calendar class="w-3.5 h-3.5" /> Data da Compra
              </label>
              <input
                id="review-date-input"
                type="date"
                bind:value={purchaseDate}
                class="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-xl text-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500/50"
              />
            </div>
          </div>

          <!-- Items Table -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <span class="text-xs font-bold text-slate-300 uppercase tracking-wider">
                  Itens da Compra ({editableItems.length})
                </span>
                <span class="text-[10px] px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-medium flex items-center gap-1">
                  <Sparkles class="w-3 h-3" /> pgvector ativo
                </span>
              </div>
              <button
                onclick={addItem}
                class="text-xs font-semibold text-emerald-400 hover:text-emerald-300 flex items-center gap-1 cursor-pointer"
              >
                <Plus class="w-3.5 h-3.5" /> Adicionar Item
              </button>
            </div>

            <div class="border border-slate-800 rounded-2xl min-h-[300px] max-h-96 overflow-y-auto">
              <table class="w-full text-left text-xs">
                <thead class="bg-slate-950 text-slate-400 uppercase font-semibold border-b border-slate-800 sticky top-0 z-20">
                  <tr>
                    <th class="px-3 py-2.5">Descri��o do Item</th>
                    <th class="px-3 py-2.5 w-16">Qtd</th>
                    <th class="px-3 py-2.5 w-24">Emb. (Tam)</th>
                    <th class="px-3 py-2.5 w-20">Unidade</th>
                    <th class="px-3 py-2.5 w-28 text-right">Pre�o Total</th>
                    <th class="px-3 py-2.5 w-28 text-right text-emerald-400">Normalizado</th>
                    <th class="px-2 py-2.5 w-8"></th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-800/60 bg-slate-900/50">
                  {#each editableItems as item, idx}
                    {@const norm = calcNormalized(item)}
                    {@const openUpwards = idx > 1 && idx >= editableItems.length - 2}
                    <tr class="hover:bg-slate-800/30">
                      <td class="px-3 py-2 item-search-container relative">
                        <div class="relative flex items-center">
                          <input
                            type="text"
                            bind:value={item.raw_description}
                            oninput={(e) => handleItemDescriptionInput(idx, (e.target as HTMLInputElement).value)}
                            onfocus={() => {
                              if (item.raw_description.trim().length >= 2 || searchResults.length > 0) {
                                activeSearchIndex = idx;
                              }
                            }}
                            placeholder="Nome / Descri��o da nota"
                            class="w-full pl-2 pr-7 py-1.5 bg-slate-950 border {item.canonical_product_id ? 'border-emerald-500/60 ring-1 ring-emerald-500/30' : 'border-slate-700'} rounded text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
                          />
                          <button
                            type="button"
                            onclick={() => triggerItemSearch(idx)}
                            class="absolute right-1 text-slate-400 hover:text-emerald-400 p-1 rounded transition cursor-pointer"
                            title="Buscar produtos j� cadastrados com pgvector"
                          >
                            <Sparkles class="w-3.5 h-3.5 {isSearchingProduct && activeSearchIndex === idx ? 'animate-spin text-emerald-400' : (item.canonical_product_id ? 'text-emerald-400' : '')}" />
                          </button>
                        </div>

                        {#if item.canonical_product_id}
                          <div class="flex items-center gap-1 mt-1 text-[10px] text-emerald-400">
                            <Sparkles class="w-2.5 h-2.5 shrink-0" />
                            <span class="truncate">Vinculado (pgvector)</span>
                            <button
                              type="button"
                              onclick={() => { item.canonical_product_id = undefined; }}
                              class="text-slate-500 hover:text-rose-400 ml-0.5 cursor-pointer"
                              title="Desvincular produto can�nico"
                            >
                              �
                            </button>
                          </div>
                        {/if}

                        <!-- Autocomplete Dropdown -->
                        {#if activeSearchIndex === idx}
                          <div class="absolute z-50 left-3 {openUpwards ? 'bottom-full mb-1' : 'top-full mt-1'} w-72 sm:w-80 bg-slate-900 border border-emerald-500/50 rounded-xl shadow-2xl overflow-hidden">
                            <div class="p-2 bg-slate-950 border-b border-slate-800 flex items-center justify-between text-[11px] text-slate-400">
                              <span class="flex items-center gap-1.5 text-emerald-400 font-semibold">
                                <Sparkles class="w-3 h-3" /> Cat�logo Inteligente (pgvector)
                              </span>
                              <button
                                type="button"
                                onclick={() => { activeSearchIndex = null; }}
                                class="text-slate-500 hover:text-slate-300 cursor-pointer"
                                title="Fechar sugest�es"
                              >
                                <X class="w-3.5 h-3.5" />
                              </button>
                            </div>

                            <div class="max-h-52 overflow-y-auto divide-y divide-slate-800/60">
                              {#if isSearchingProduct}
                                <div class="p-4 text-center text-xs text-slate-400 flex items-center justify-center gap-2">
                                  <div class="w-3.5 h-3.5 border-2 border-emerald-500/30 border-t-emerald-400 rounded-full animate-spin"></div>
                                  <span>Buscando produtos com IA & pgvector...</span>
                                </div>
                              {:else if searchResults.length === 0}
                                <div class="p-3 text-center text-xs text-slate-400">
                                  <span>Nenhum produto similar encontrado.</span>
                                  <p class="text-[10px] text-slate-500 mt-0.5">Um novo produto can�nico e vetor ser�o gerados ao salvar.</p>
                                </div>
                              {:else}
                                {#each searchResults as product}
                                  <button
                                    type="button"
                                    onclick={() => selectProductForIndex(idx, product)}
                                    class="w-full text-left px-3 py-2 hover:bg-slate-800/80 transition flex items-center justify-between group cursor-pointer"
                                  >
                                    <div class="min-w-0 pr-2">
                                      <div class="font-medium text-slate-200 group-hover:text-emerald-400 transition-colors truncate">
                                        {product.name}
                                      </div>
                                      <div class="text-[10px] text-slate-500 flex items-center gap-1.5 mt-0.5">
                                        <span class="truncate">{product.category || 'Geral'}</span>
                                        <span>�</span>
                                        <span class="text-slate-400">Base: <strong class="text-slate-300 font-mono">{product.base_unit}</strong></span>
                                      </div>
                                    </div>
                                    <span class="text-[10px] font-semibold px-2 py-0.5 rounded bg-slate-800 group-hover:bg-emerald-500/20 group-hover:text-emerald-300 text-slate-400 shrink-0">
                                      Usar
                                    </span>
                                  </button>
                                {/each}
                              {/if}
                            </div>
                          </div>
                        {/if}
                      </td>
                      <td class="px-3 py-2">
                        <input
                          type="number"
                          step="any"
                          min="0.01"
                          bind:value={item.quantity}
                          class="w-full px-2 py-1 bg-slate-950 border border-slate-700 rounded text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
                        />
                      </td>
                      <td class="px-3 py-2">
                        <input
                          type="number"
                          step="any"
                          min="0.01"
                          bind:value={item.package_size}
                          class="w-full px-2 py-1 bg-slate-950 border border-slate-700 rounded text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
                        />
                      </td>
                      <td class="px-3 py-2">
                        <select
                          bind:value={item.package_unit}
                          class="w-full px-2 py-1 bg-slate-950 border border-slate-700 rounded text-slate-100 text-xs focus:ring-1 focus:ring-emerald-500 focus:outline-none"
                        >
                          {#each units as u}
                            <option value={u}>{u}</option>
                          {/each}
                        </select>
                      </td>
                      <td class="px-3 py-2 text-right">
                        <input
                          type="number"
                          step="0.01"
                          min="0"
                          bind:value={item.total_price}
                          class="w-full px-2 py-1 bg-slate-950 border border-slate-700 rounded text-slate-100 text-xs text-right font-mono focus:ring-1 focus:ring-emerald-500 focus:outline-none"
                        />
                      </td>
                      <td class="px-3 py-2 text-right font-mono font-bold text-emerald-400">
                        {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(norm)}
                        <span class="text-[10px] text-slate-400 font-normal">/{item.package_unit === 'g' ? 'kg' : (item.package_unit === 'ml' ? 'L' : item.package_unit)}</span>
                      </td>
                      <td class="px-2 py-2 text-center">
                        <button
                          onclick={() => removeItem(idx)}
                          disabled={editableItems.length === 1}
                          class="text-slate-500 hover:text-rose-400 disabled:opacity-20 transition cursor-pointer"
                          title="Remover item"
                        >
                          <Trash2 class="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>

          <!-- Total Footer & Save -->
          <div class="flex flex-col sm:flex-row items-center justify-between gap-4 pt-3 border-t border-slate-800">
            <div>
              <span class="text-xs text-slate-400 uppercase font-semibold">Valor Total Calculado:</span>
              <div class="text-2xl font-black text-white font-mono tracking-tight">
                {new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' }).format(reviewTotal)}
              </div>
            </div>

            <div class="flex items-center gap-3">
              <button
                onclick={() => { step = 'input'; }}
                class="px-4 py-2.5 rounded-xl border border-slate-700 text-slate-300 hover:bg-slate-800 text-xs font-semibold transition cursor-pointer"
              >
                Voltar
              </button>

              <button
                onclick={handleConfirmAndSave}
                disabled={isSubmitting || !selectedUserId}
                class="px-6 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-bold text-sm shadow-lg shadow-emerald-500/20 transition flex items-center gap-2 cursor-pointer"
              >
                {#if isSubmitting}
                  <div class="w-4 h-4 border-2 border-slate-950 border-t-transparent rounded-full animate-spin"></div>
                  <span>Salvando no Banco & pgvector...</span>
                {:else}
                  <Check class="w-4 h-4 stroke-[3]" />
                  <span>Confirmar e Salvar Compra</span>
                {/if}
              </button>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
