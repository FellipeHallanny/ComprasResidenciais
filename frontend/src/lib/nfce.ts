// Utilitários para decodificação e enriquecimento de dados da NFC-e / NF-e brasileira

export interface ParsedNfceInfo {
  rawText: string;
  fiscalKey: string;
  formattedKey: string;
  stateCode?: string;
  stateUf?: string;
  issueYearMonth?: string;
  storeCnpj?: string;
  formattedCnpj?: string;
  model?: 'NFC-e' | 'NF-e' | 'Outro';
  series?: string;
  number?: string;
  rawUrl?: string;
}

export const IBGE_UF_MAP: Record<string, string> = {
  '11': 'RO', '12': 'AC', '13': 'AM', '14': 'RR', '15': 'PA', '16': 'AP', '17': 'TO',
  '21': 'MA', '22': 'PI', '23': 'CE', '24': 'RN', '25': 'PB', '26': 'PE', '27': 'AL', '28': 'SE', '29': 'BA',
  '31': 'MG', '32': 'ES', '33': 'RJ', '35': 'SP',
  '41': 'PR', '42': 'SC', '43': 'RS',
  '50': 'MS', '51': 'MT', '52': 'GO', '53': 'DF'
};

/**
 * Formata CNPJ numérico de 14 dígitos para XX.XXX.XXX/XXXX-XX
 */
export function formatCnpj(cnpjDigits: string): string {
  const clean = cnpjDigits.replace(/\D/g, '');
  if (clean.length !== 14) return cnpjDigits;
  return clean.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, '$1.$2.$3/$4-$5');
}

/**
 * Formata chave de 44 dígitos em grupos de 4 (ex: 3524 0100 0000 ...)
 */
export function formatFiscalKey(key: string): string {
  const clean = key.replace(/\D/g, '');
  if (clean.length !== 44) return key;
  return clean.replace(/(\d{4})/g, '$1 ').trim();
}

/**
 * Validação estrutural e matemática da Chave de Acesso Fiscal (44 dígitos):
 * 1. Deve conter exatamente 44 dígitos numéricos
 * 2. Não pode ser sequência repetida de dígitos triviais (ex: todos zeros)
 * 3. Código de UF deve existir no mapeamento do IBGE
 * 4. Mês de emissão deve estar entre 01 e 12
 * 5. Dígito verificador final (posição 44) deve conferir com cálculo módulo 11
 */
export function validateFiscalKeyCheckDigit(key: string): boolean {
  const clean = key.replace(/\D/g, '');
  if (clean.length !== 44) return false;

  // Rejeita sequências triviais inválidas
  if (/^(\d)\1{43}$/.test(clean)) return false;

  // Valida código de UF inicial
  const ufCode = clean.substring(0, 2);
  if (!IBGE_UF_MAP[ufCode]) return false;

  // Valida mês de emissão (01 a 12)
  const month = parseInt(clean.substring(4, 6), 10);
  if (month < 1 || month > 12) return false;

  const base = clean.slice(0, 43);
  const expectedDigit = parseInt(clean.charAt(43), 10);

  let sum = 0;
  let weight = 2;
  for (let i = base.length - 1; i >= 0; i--) {
    sum += parseInt(base[i], 10) * weight;
    weight++;
    if (weight > 9) weight = 2;
  }

  const mod = sum % 11;
  const calculatedDigit = mod === 0 || mod === 1 ? 0 : 11 - mod;
  return calculatedDigit === expectedDigit;
}

/**
 * Extrai a chave fiscal de 44 dígitos de qualquer texto ou URL de QR Code brasileiro,
 * garantindo que apenas sequências que passam na validação módulo 11 sejam aceitas.
 */
export function extractFiscalKey(text: string): string | null {
  if (!text) return null;
  const trimmed = text.trim();

  // 1. Sequência direta de 44 dígitos com DV válido
  const directMatch = trimmed.match(/\b\d{44}\b/);
  if (directMatch && validateFiscalKeyCheckDigit(directMatch[0])) {
    return directMatch[0];
  }

  // 2. Tentar analisar como URL com parâmetro p= ou chNFe=
  try {
    const url = new URL(trimmed);
    const pParam = url.searchParams.get('p') || url.searchParams.get('P');
    if (pParam) {
      const parts = pParam.split('|');
      for (const part of parts) {
        const cleanPart = part.replace(/\D/g, '');
        if (cleanPart.length === 44 && validateFiscalKeyCheckDigit(cleanPart)) {
          return cleanPart;
        }
      }
    }

    const chParam = url.searchParams.get('chNFe') || url.searchParams.get('chave') || url.searchParams.get('ch');
    if (chParam) {
      const cleanCh = chParam.replace(/\D/g, '');
      if (cleanCh.length === 44 && validateFiscalKeyCheckDigit(cleanCh)) {
        return cleanCh;
      }
    }
  } catch {
    // Não é URL, segue para o extrator geral
  }

  // 3. Fallback: Limpar caracteres não numéricos e procurar 44 dígitos válidos
  const digits = trimmed.replace(/\D/g, '');
  if (digits.length === 44 && validateFiscalKeyCheckDigit(digits)) {
    return digits;
  }
  if (digits.length > 44) {
    // Procura qualquer substring contígua de 44 dígitos que possua dígito verificador módulo 11 válido
    for (let i = 0; i <= digits.length - 44; i++) {
      const candidate = digits.substring(i, i + 44);
      if (validateFiscalKeyCheckDigit(candidate)) {
        return candidate;
      }
    }
  }

  return null;
}

/**
 * Decodifica metadados detalhados embutidos na Chave de Acesso Fiscal de 44 dígitos:
 * - cUF (2 dígitos): Código do estado IBGE
 * - AAMM (4 dígitos): Ano e mês de emissão
 * - CNPJ (14 dígitos): CNPJ do emitente/supermercado
 * - mod (2 dígitos): 65 para NFC-e, 55 para NF-e
 * - serie (3 dígitos)
 * - nNF (9 dígitos): Número da nota fiscal
 * - tpEmis (1 dígito)
 * - cNF (8 dígitos): Código numérico
 * - cDV (1 dígito): Dígito verificador
 *
 * Retorna null se o texto não contiver uma chave válida com módulo 11 correto.
 */
export function parseNfceFromText(rawText: string): ParsedNfceInfo | null {
  const key = extractFiscalKey(rawText);
  if (!key || key.length !== 44 || !validateFiscalKeyCheckDigit(key)) {
    return null;
  }

  const stateCode = key.substring(0, 2);
  const stateUf = IBGE_UF_MAP[stateCode] || 'BR';
  const aa = key.substring(2, 4);
  const mm = key.substring(4, 6);
  const issueYearMonth = `${mm}/20${aa}`;

  const storeCnpj = key.substring(6, 20);
  const formattedCnpj = formatCnpj(storeCnpj);

  const modCode = key.substring(20, 22);
  let model: 'NFC-e' | 'NF-e' | 'Outro' = 'Outro';
  if (modCode === '65') model = 'NFC-e';
  else if (modCode === '55') model = 'NF-e';

  const series = key.substring(22, 25).replace(/^0+/, '') || '1';
  const number = key.substring(25, 34).replace(/^0+/, '') || '0';

  let rawUrl: string | undefined;
  if (rawText.startsWith('http://') || rawText.startsWith('https://')) {
    rawUrl = rawText.trim();
  }

  return {
    rawText,
    fiscalKey: key,
    formattedKey: formatFiscalKey(key),
    stateCode,
    stateUf,
    issueYearMonth,
    storeCnpj,
    formattedCnpj,
    model,
    series,
    number,
    rawUrl,
  };
}
