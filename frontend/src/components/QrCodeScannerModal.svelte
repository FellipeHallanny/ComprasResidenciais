<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte';
  import {
    X,
    Camera,
    CameraOff,
    Zap,
    ZapOff,
    Upload,
    Check,
    AlertCircle,
    QrCode,
    CheckCircle2,
    Building2,
    MapPin,
    Calendar,
    FileCheck2,
    ShieldAlert,
    RefreshCw,
    Lock
  } from '@lucide/svelte';
  import { Html5Qrcode, type CameraDevice } from 'html5-qrcode';
  import { parseNfceFromText, type ParsedNfceInfo } from '../lib/nfce';

  interface Props {
    onclose: () => void;
    onscan: (info: ParsedNfceInfo) => void;
  }

  let { onclose, onscan }: Props = $props();

  let scannerContainerId = 'qr-camera-viewport';
  let html5QrCode: Html5Qrcode | null = null;
  let fileInputRef: HTMLInputElement | null = null;

  // Estados reativos (Svelte 5 Runes)
  let isScanning = $state(false);
  let isInitializing = $state(true);
  let errorMessage = $state('');
  let failureReason = $state<'permission_denied' | 'no_camera' | 'insecure_context' | 'busy' | 'unknown' | null>(null);
  let cameras = $state<CameraDevice[]>([]);
  let selectedCameraId = $state<string>('');
  let isTorchOn = $state(false);
  let hasTorch = $state(false);

  // Resultado do escaneamento
  let scannedInfo = $state<ParsedNfceInfo | null>(null);

  // Feedback sonoro e tátil de sucesso
  function playSuccessFeedback() {
    if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
      try {
        navigator.vibrate([80, 40, 80]);
      } catch {
        // ignore
      }
    }
    try {
      const audioCtx = new (window.AudioContext || (window as any).webkitAudioContext)();
      const osc = audioCtx.createOscillator();
      const gain = audioCtx.createGain();
      osc.type = 'sine';
      osc.frequency.setValueAtTime(880, audioCtx.currentTime);
      osc.frequency.exponentialRampToValueAtTime(1760, audioCtx.currentTime + 0.12);
      gain.gain.setValueAtTime(0.15, audioCtx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.01, audioCtx.currentTime + 0.12);
      osc.connect(gain);
      gain.connect(audioCtx.destination);
      osc.start();
      osc.stop(audioCtx.currentTime + 0.12);
    } catch {
      // AudioContext may be blocked before gesture, harmless to ignore
    }
  }

  /**
   * Inicializa o scanner de câmera com fallback em cascata:
   * 1. Câmera selecionada explicitamente (se houver)
   * 2. Câmera traseira ({ facingMode: "environment" })
   * 3. Câmera frontal / webcam ({ facingMode: "user" })
   * 4. Enumeração direta de dispositivos via ID
   */
  async function startScanner() {
    errorMessage = '';
    failureReason = null;
    isInitializing = true;

    // 1. Validação de Contexto Seguro (HTTPS ou localhost)
    const isLocal = typeof window !== 'undefined' && (
      location.hostname === 'localhost' ||
      location.hostname === '127.0.0.1' ||
      location.hostname === '[::1]'
    );
    const isSecure = typeof window !== 'undefined' && (window.isSecureContext || isLocal);

    if (!isSecure) {
      failureReason = 'insecure_context';
      isInitializing = false;
      errorMessage = 'Vídeo ao vivo exige HTTPS para funcionar pelo celular. Dica: use o botão abaixo para tirar uma foto do cupom diretamente.';
      return;
    }

    if (typeof navigator === 'undefined' || !navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      failureReason = 'no_camera';
      isInitializing = false;
      errorMessage = 'Seu navegador não oferece suporte à captura de vídeo ao vivo. Use a opção de foto do cupom.';
      return;
    }

    await tick();
    const container = document.getElementById(scannerContainerId);
    if (!container) {
      isInitializing = false;
      errorMessage = 'Elemento de exibição da câmera não encontrado na página.';
      return;
    }

    // Limpa scanner prévio de forma segura
    await safeStopAndClear();

    try {
      html5QrCode = new Html5Qrcode(scannerContainerId, { verbose: false });

      // Tenta enumerar câmeras disponíveis
      try {
        const devices = await Html5Qrcode.getCameras();
        cameras = devices || [];
        if (cameras.length > 0 && !selectedCameraId) {
          const backCam = cameras.find(c => /back|rear|traseira|ambiente|environment/i.test(c.label));
          selectedCameraId = backCam ? backCam.id : cameras[0].id;
        }
      } catch (enumErr) {
        console.warn('Enumeração prévia de câmeras:', enumErr);
      }

      await runScannerWithCascade();
    } catch (err: any) {
      console.warn('Falha na inicialização do scanner:', err);
      handleCameraFailure(err);
    } finally {
      isInitializing = false;
    }
  }

  async function runScannerWithCascade() {
    if (!html5QrCode) return;

    const config = {
      fps: 15,
      qrbox: (viewfinderWidth: number, viewfinderHeight: number) => {
        const minDim = Math.min(viewfinderWidth, viewfinderHeight);
        const size = Math.floor(minDim * 0.72);
        return { width: size, height: size };
      },
      aspectRatio: 1.0,
    };

    const successCb = (decodedText: string) => {
      handleDecodedText(decodedText);
    };

    const errorCb = () => {
      // ciclo normal por frame sem QR code
    };

    // Tentativa 1: ID selecionado
    if (selectedCameraId) {
      try {
        await html5QrCode.start(selectedCameraId, config, successCb, errorCb);
        finalizeStartSuccess();
        return;
      } catch (idErr) {
        console.warn('Falha com ID específico, tentando modo traseiro…', idErr);
      }
    }

    // Tentativa 2: Traseira (environment)
    try {
      await html5QrCode.start({ facingMode: 'environment' }, config, successCb, errorCb);
      finalizeStartSuccess();
      return;
    } catch (envErr) {
      console.warn('Falha facingMode environment, tentando webcam frontal…', envErr);
    }

    // Tentativa 3: Frontal / Webcam notebook (user)
    try {
      await html5QrCode.start({ facingMode: 'user' }, config, successCb, errorCb);
      finalizeStartSuccess();
      return;
    } catch (userErr) {
      console.warn('Falha facingMode user, tentando dispositivos enumerados…', userErr);
    }

    // Tentativa 4: Iterar em todas as câmeras listadas pelo sistema
    try {
      const devices = await Html5Qrcode.getCameras();
      if (devices && devices.length > 0) {
        for (const dev of devices) {
          try {
            await html5QrCode.start(dev.id, config, successCb, errorCb);
            selectedCameraId = dev.id;
            finalizeStartSuccess();
            return;
          } catch {
            // Continua para o próximo dispositivo
          }
        }
      }
    } catch {
      // Ignora erro de enumeração e cai no erro final
    }

    throw new Error('Nenhuma câmera física pôde ser iniciada neste dispositivo.');
  }

  function finalizeStartSuccess() {
    isScanning = true;
    failureReason = null;
    errorMessage = '';

    // Atualiza lista de câmeras com rótulos agora liberados
    Html5Qrcode.getCameras().then(devices => {
      cameras = devices || [];
    }).catch(() => {});

    // Verifica lanterna
    try {
      // @ts-ignore
      const capabilities = html5QrCode?.getRunningTrackCapabilities();
      hasTorch = Boolean(capabilities && 'torch' in capabilities);
    } catch {
      hasTorch = false;
    }
  }

  function handleCameraFailure(err: any) {
    isScanning = false;
    const str = String(err?.message || err || '');

    if (
      str.includes('NotAllowedError') ||
      str.includes('PermissionDeniedError') ||
      str.includes('permission') ||
      err?.name === 'NotAllowedError'
    ) {
      failureReason = 'permission_denied';
      errorMessage = 'Permissão de acesso à câmera negada. Desbloqueie o acesso no navegador.';
    } else if (
      str.includes('NotFoundError') ||
      str.includes('DevicesNotFoundError') ||
      str.includes('Requested device not found') ||
      str.includes('Nenhuma câmera física') ||
      err?.name === 'NotFoundError'
    ) {
      failureReason = 'no_camera';
      errorMessage = 'Nenhuma câmera física foi detectada neste computador ou aparelho.';
    } else if (
      str.includes('NotReadableError') ||
      str.includes('TrackStartError') ||
      str.includes('Could not start video source') ||
      err?.name === 'NotReadableError'
    ) {
      failureReason = 'busy';
      errorMessage = 'A câmera está ocupada por outro aplicativo (Teams, Zoom, etc.) ou o driver travou.';
    } else {
      failureReason = 'unknown';
      errorMessage = 'Falha ao acessar a câmera: ' + (err?.message || str);
    }
  }

  async function safeStopAndClear() {
    if (html5QrCode) {
      try {
        if (html5QrCode.isScanning) {
          await html5QrCode.stop();
        }
        html5QrCode.clear();
      } catch (err) {
        console.warn('Erro ao finalizar html5QrCode:', err);
      } finally {
        html5QrCode = null;
        isScanning = false;
        isTorchOn = false;
      }
    }
  }

  async function toggleTorch() {
    if (!html5QrCode || !hasTorch) return;
    try {
      // @ts-ignore
      await html5QrCode.applyVideoConstraints({
        advanced: [{ torch: !isTorchOn } as any]
      });
      isTorchOn = !isTorchOn;
    } catch (err) {
      console.warn('Falha ao alternar lanterna:', err);
    }
  }

  async function switchCamera(deviceId: string) {
    selectedCameraId = deviceId;
    await safeStopAndClear();
    await startScanner();
  }

  function handleDecodedText(text: string) {
    const parsed = parseNfceFromText(text);
    if (parsed) {
      playSuccessFeedback();
      scannedInfo = parsed;
      safeStopAndClear();
    } else {
      errorMessage = 'QR Code lido, mas não contém uma chave de NFC-e válida (44 dígitos). Aponte para o QR Code fiscal.';
      setTimeout(() => {
        if (errorMessage.includes('QR Code lido')) errorMessage = '';
      }, 4500);
    }
  }

  // Fallback 100% garantido: Tirar foto com câmera nativa do celular ou escolher da galeria
  async function handleFileScan(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const file = input.files[0];

    try {
      await tick();
      await safeStopAndClear();

      if (!html5QrCode) {
        html5QrCode = new Html5Qrcode(scannerContainerId, { verbose: false });
      }

      errorMessage = '';
      isInitializing = true;
      const decodedText = await html5QrCode.scanFile(file, true);
      handleDecodedText(decodedText);
    } catch (err: any) {
      console.warn('Falha ao decodificar imagem:', err);
      errorMessage = 'Não foi possível encontrar um QR Code nítido nesta imagem. Tente tirar uma foto mais aproximada e bem iluminada do cupom.';
    } finally {
      isInitializing = false;
      input.value = ''; // Permite selecionar o mesmo arquivo novamente
    }
  }

  function triggerFileInput() {
    fileInputRef?.click();
  }

  function handleConfirmUse() {
    if (scannedInfo) {
      onscan(scannedInfo);
      onclose();
    }
  }

  async function handleRescan() {
    scannedInfo = null;
    await tick();
    await startScanner();
  }

  onMount(() => {
    startScanner();
  });

  onDestroy(async () => {
    await safeStopAndClear();
  });
</script>

<!-- Input oculto para captura nativa da câmera do celular ou upload de foto -->
<input
  bind:this={fileInputRef}
  type="file"
  accept="image/*"
  capture="environment"
  onchange={handleFileScan}
  class="hidden"
  aria-label="Upload de foto ou cupom fiscal"
/>

<div class="fixed inset-0 z-50 bg-black/85 backdrop-blur-md flex items-center justify-center p-3 sm:p-4 overflow-y-auto overscroll-contain">
  <div
    role="dialog"
    aria-modal="true"
    aria-labelledby="qr-modal-title"
    class="bg-slate-900 border border-slate-700 rounded-3xl w-full max-w-lg shadow-2xl overflow-hidden my-auto flex flex-col"
  >
    <!-- Modal Header -->
    <div class="px-5 py-3.5 border-b border-slate-800 flex items-center justify-between bg-slate-900/80">
      <div class="flex items-center gap-2.5">
        <div class="h-8 w-8 rounded-xl bg-emerald-500/15 text-emerald-400 flex items-center justify-center border border-emerald-500/30">
          <QrCode class="w-4 h-4 stroke-[2.5]" aria-hidden="true" />
        </div>
        <div>
          <h3 id="qr-modal-title" class="text-sm font-bold text-white flex items-center gap-2">
            <span>Leitor de QR Code de Cupom Fiscal</span>
            <span class="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              NFC-e
            </span>
          </h3>
          <p class="text-[11px] text-slate-400">
            Aponte a câmera para o QR Code impresso no cupom da SEFAZ
          </p>
        </div>
      </div>

      <button
        type="button"
        onclick={onclose}
        class="text-slate-400 hover:text-slate-200 p-1.5 rounded-lg hover:bg-slate-800 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
        aria-label="Fechar modal"
        title="Fechar"
      >
        <X class="w-5 h-5" aria-hidden="true" />
      </button>
    </div>

    <!-- Banner Superior de Erro (se houver mensagem e scanner ativo ou foto com falha) -->
    {#if errorMessage && (isScanning || failureReason === null)}
      <div role="status" aria-live="polite" class="mx-4 mt-3 p-3 rounded-xl bg-rose-500/15 border border-rose-500/30 text-rose-300 text-xs flex items-center justify-between gap-2 animate-in fade-in">
        <div class="flex items-center gap-2">
          <AlertCircle class="w-4 h-4 shrink-0" aria-hidden="true" />
          <span>{errorMessage}</span>
        </div>
        <button
          type="button"
          onclick={() => (errorMessage = '')}
          class="text-slate-400 hover:text-white text-xs cursor-pointer p-1 rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rose-500"
          aria-label="Fechar mensagem de erro"
        >&times;</button>
      </div>
    {/if}

    <!-- Camera / Viewport Area -->
    <div class="p-4 flex flex-col items-center">
      {#if scannedInfo}
        <!-- Sucesso: Exibe Dados Decodificados da NFC-e -->
        <div class="w-full bg-slate-950/80 border border-emerald-500/40 rounded-2xl p-4 space-y-3.5 animate-in fade-in zoom-in-95 duration-200">
          <div class="flex items-center gap-2.5 text-emerald-400 font-bold text-xs uppercase tracking-wider pb-2 border-b border-slate-800">
            <div class="h-6 w-6 rounded-full bg-emerald-500/20 flex items-center justify-center">
              <Check class="w-3.5 h-3.5 stroke-[3]" aria-hidden="true" />
            </div>
            <span>QR Code Fiscal Decodificado com Sucesso!</span>
          </div>

          <!-- Chave Formatada -->
          <div>
            <span class="text-[10px] font-semibold text-slate-400 uppercase tracking-wider block mb-1">
              Chave de Acesso (44 dígitos)
            </span>
            <div class="p-2.5 bg-slate-900 border border-slate-700/80 rounded-xl font-mono tabular-nums text-xs text-emerald-300 break-all select-all leading-relaxed">
              {scannedInfo.formattedKey}
            </div>
          </div>

          <!-- Metadados da NFC-e Extraídos do Código Fiscal -->
          <div class="grid grid-cols-2 gap-2 text-xs">
            <div class="p-2.5 bg-slate-900/60 border border-slate-800 rounded-xl">
              <span class="text-[10px] text-slate-400 flex items-center gap-1">
                <MapPin class="w-3 h-3 text-emerald-400" aria-hidden="true" /> Estado / UF:
              </span>
              <strong class="text-white text-xs mt-0.5 block">{scannedInfo.stateUf} (Código {scannedInfo.stateCode})</strong>
            </div>

            <div class="p-2.5 bg-slate-900/60 border border-slate-800 rounded-xl">
              <span class="text-[10px] text-slate-400 flex items-center gap-1">
                <Calendar class="w-3 h-3 text-emerald-400" aria-hidden="true" /> Emissão:
              </span>
              <strong class="text-white text-xs mt-0.5 block tabular-nums">{scannedInfo.issueYearMonth}</strong>
            </div>

            <div class="p-2.5 bg-slate-900/60 border border-slate-800 rounded-xl col-span-2">
              <span class="text-[10px] text-slate-400 flex items-center gap-1">
                <Building2 class="w-3 h-3 text-emerald-400" aria-hidden="true" /> CNPJ do Estabelecimento:
              </span>
              <strong class="text-white font-mono tabular-nums text-xs mt-0.5 block">{scannedInfo.formattedCnpj || scannedInfo.storeCnpj}</strong>
            </div>

            <div class="p-2.5 bg-slate-900/60 border border-slate-800 rounded-xl">
              <span class="text-[10px] text-slate-400 block">Modelo:</span>
              <strong class="text-emerald-400 text-xs mt-0.5 block tabular-nums">{scannedInfo.model}</strong>
            </div>

            <div class="p-2.5 bg-slate-900/60 border border-slate-800 rounded-xl">
              <span class="text-[10px] text-slate-400 block">Número do Cupom:</span>
              <strong class="text-white font-mono tabular-nums text-xs mt-0.5 block">Nº {scannedInfo.number} (Série {scannedInfo.series})</strong>
            </div>
          </div>

          <!-- Ações pós-leitura -->
          <div class="pt-2 flex items-center justify-between gap-3 border-t border-slate-800/80">
            <button
              type="button"
              onclick={handleRescan}
              class="px-3.5 py-2.5 rounded-xl border border-slate-700 text-slate-300 hover:text-white hover:bg-slate-800 text-xs font-semibold transition-colors cursor-pointer flex items-center gap-1.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
            >
              <RefreshCw class="w-3.5 h-3.5" aria-hidden="true" />
              <span>Escanear Outro</span>
            </button>

            <button
              type="button"
              onclick={handleConfirmUse}
              class="flex-1 px-5 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs shadow-lg shadow-emerald-500/20 transition-colors flex items-center justify-center gap-1.5 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
            >
              <FileCheck2 class="w-4 h-4" aria-hidden="true" />
              <span>Usar esta Chave Fiscal</span>
            </button>
          </div>
        </div>
      {:else}
        <!-- Visualizador da Câmera Ativa -->
        <div class="w-full relative rounded-2xl overflow-hidden bg-slate-950 border border-slate-800 aspect-square max-w-sm flex items-center justify-center shadow-inner">
          
          <!-- Elemento HTML onde o Html5Qrcode injeta o elemento <video> -->
          <div id={scannerContainerId} class="w-full h-full overflow-hidden [&_video]:object-cover [&_video]:w-full [&_video]:h-full"></div>

          <!-- Estado 1: Inicializando / Carregando câmera -->
          {#if isInitializing}
            <div class="absolute inset-0 bg-slate-950 flex flex-col items-center justify-center gap-3 text-slate-400 text-xs p-6 text-center animate-in fade-in" aria-live="polite">
              <div class="w-9 h-9 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
              <span class="font-semibold text-slate-200 text-sm">Conectando à câmera…</span>
              <p class="text-[11px] text-slate-400 max-w-xs">
                Se o navegador solicitar permissão, clique em <strong>"Permitir"</strong>.
              </p>
            </div>

          <!-- Estado 2: Falha por Permissão Negada -->
          {:else if failureReason === 'permission_denied'}
            <div role="alert" class="absolute inset-0 bg-slate-950/95 flex flex-col items-center justify-center gap-3 p-6 text-center animate-in fade-in">
              <div class="h-12 w-12 rounded-2xl bg-amber-500/15 text-amber-400 flex items-center justify-center border border-amber-500/30">
                <ShieldAlert class="w-6 h-6" aria-hidden="true" />
              </div>
              <div>
                <h4 class="text-sm font-bold text-amber-300">Permissão de Câmera Bloqueada</h4>
                <p class="text-xs text-slate-300 mt-1 max-w-xs text-pretty">
                  O navegador bloqueou o acesso à câmera neste site.
                </p>
              </div>
              <div class="p-2.5 rounded-xl bg-slate-900 border border-slate-800 text-[11px] text-left text-slate-300 space-y-1 w-full max-w-xs">
                <div class="flex items-center gap-1.5 text-amber-400 font-semibold text-[10px] uppercase tracking-wider">
                  <Lock class="w-3 h-3" aria-hidden="true" /> Como desbloquear:
                </div>
                <p>1. Clique no <strong>ícone de cadeado / configurações</strong> na barra de endereços do navegador.</p>
                <p>2. Mude a permissão de <strong>Câmera</strong> para <strong>Permitir</strong>.</p>
                <p>3. Clique no botão abaixo para tentar novamente.</p>
              </div>
              <div class="flex flex-col gap-2 w-full max-w-xs pt-1">
                <button
                  type="button"
                  onclick={startScanner}
                  class="w-full px-4 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold text-xs shadow-md transition-colors flex items-center justify-center gap-1.5 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-400"
                >
                  <RefreshCw class="w-3.5 h-3.5" aria-hidden="true" />
                  <span>Tentar Novamente</span>
                </button>
                <button
                  type="button"
                  onclick={triggerFileInput}
                  class="w-full px-4 py-2 rounded-xl border border-slate-700 bg-slate-900 hover:bg-slate-800 text-slate-200 text-xs font-semibold transition-colors flex items-center justify-center gap-1.5 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                >
                  <Camera class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
                  <span>Tirar Foto / Carregar da Galeria</span>
                </button>
              </div>
            </div>

          <!-- Estado 3: Falha por Nenhuma Câmera Detectada (Desktop sem webcam) -->
          {:else if failureReason === 'no_camera'}
            <div role="alert" class="absolute inset-0 bg-slate-950/95 flex flex-col items-center justify-center gap-3 p-6 text-center animate-in fade-in">
              <div class="h-12 w-12 rounded-2xl bg-rose-500/15 text-rose-400 flex items-center justify-center border border-rose-500/30">
                <CameraOff class="w-6 h-6" aria-hidden="true" />
              </div>
              <div>
                <h4 class="text-sm font-bold text-white">Nenhuma Câmera Detectada</h4>
                <p class="text-xs text-slate-400 mt-1 max-w-xs text-pretty">
                  Este computador não possui webcam conectada ou ela está desativada no sistema operacional.
                </p>
              </div>
              <div class="w-full max-w-xs space-y-2 pt-2">
                <button
                  type="button"
                  onclick={triggerFileInput}
                  class="w-full px-4 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs shadow-lg shadow-emerald-500/20 transition-colors flex items-center justify-center gap-2 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                >
                  <Upload class="w-4 h-4" aria-hidden="true" />
                  <span>Carregar Imagem / Foto do Cupom</span>
                </button>
                <button
                  type="button"
                  onclick={startScanner}
                  class="w-full px-4 py-2 rounded-xl border border-slate-700 bg-slate-900 hover:bg-slate-800 text-slate-300 text-xs font-semibold transition-colors flex items-center justify-center gap-1.5 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                >
                  <RefreshCw class="w-3.5 h-3.5" aria-hidden="true" />
                  <span>Tentar Detectar Câmera Novamente</span>
                </button>
              </div>
            </div>

          <!-- Estado 4: Contexto Inseguro (HTTP na rede local) -->
          {:else if failureReason === 'insecure_context'}
            <div role="alert" class="absolute inset-0 bg-slate-950/95 flex flex-col items-center justify-center gap-3 p-6 text-center animate-in fade-in">
              <div class="h-12 w-12 rounded-2xl bg-sky-500/15 text-sky-400 flex items-center justify-center border border-sky-500/30">
                <ShieldAlert class="w-6 h-6" aria-hidden="true" />
              </div>
              <div>
                <h4 class="text-sm font-bold text-sky-300">Acesso via Rede Local (HTTP)</h4>
                <p class="text-xs text-slate-300 mt-1 max-w-xs text-pretty">
                  Por restrição de segurança dos navegadores, streaming de câmera ao vivo exige <strong>HTTPS</strong> ou <strong>localhost</strong>.
                </p>
              </div>
              <p class="text-[11px] text-slate-400 max-w-xs">
                No celular, você pode tirar uma foto do cupom fiscal agora com a câmera nativa:
              </p>
              <div class="w-full max-w-xs space-y-2 pt-1">
                <button
                  type="button"
                  onclick={triggerFileInput}
                  class="w-full px-4 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs shadow-lg shadow-emerald-500/20 transition-colors flex items-center justify-center gap-2 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                >
                  <Camera class="w-4 h-4" aria-hidden="true" />
                  <span>Tirar Foto com a Câmera do Celular</span>
                </button>
                <div class="p-2 rounded-xl bg-slate-900 border border-slate-800 text-[10px] text-slate-400 text-left">
                  <span class="text-sky-300 font-semibold block">Dica para desenvolvedor:</span>
                  Execute com HTTPS: <code class="text-emerald-300">npm run dev:https</code> para liberar o vídeo ao vivo na rede Wi-Fi.
                </div>
              </div>
            </div>

          <!-- Estado 5: Câmera Ocupada ou Erro Geral -->
          {:else if failureReason === 'busy' || (!isScanning && failureReason !== null)}
            <div role="alert" class="absolute inset-0 bg-slate-950/95 flex flex-col items-center justify-center gap-3 p-6 text-center animate-in fade-in">
              <div class="h-12 w-12 rounded-2xl bg-amber-500/15 text-amber-400 flex items-center justify-center border border-amber-500/30">
                <AlertCircle class="w-6 h-6" aria-hidden="true" />
              </div>
              <div>
                <h4 class="text-sm font-bold text-amber-300">Não foi possível iniciar o vídeo</h4>
                <p class="text-xs text-slate-300 mt-1 max-w-xs text-pretty">
                  {errorMessage || 'A câmera pode estar em uso por outro aplicativo (Zoom, Teams, etc.) ou o driver falhou.'}
                </p>
              </div>
              <div class="w-full max-w-xs space-y-2 pt-2">
                <button
                  type="button"
                  onclick={startScanner}
                  class="w-full px-4 py-2.5 rounded-xl bg-amber-500 hover:bg-amber-400 text-slate-950 font-bold text-xs shadow-md transition-colors flex items-center justify-center gap-1.5 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-400"
                >
                  <RefreshCw class="w-3.5 h-3.5" aria-hidden="true" />
                  <span>Tentar Ligar Câmera Novamente</span>
                </button>
                <button
                  type="button"
                  onclick={triggerFileInput}
                  class="w-full px-4 py-2 rounded-xl border border-slate-700 bg-slate-900 hover:bg-slate-800 text-slate-200 text-xs font-semibold transition-colors flex items-center justify-center gap-1.5 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                >
                  <Camera class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
                  <span>Tirar Foto / Carregar da Galeria</span>
                </button>
              </div>
            </div>

          <!-- Estado 6: Inicial antes da permissão / Pronto para Iniciar -->
          {:else if !isScanning}
            <div class="absolute inset-0 bg-slate-950/95 flex flex-col items-center justify-center gap-3 text-center p-6 animate-in fade-in">
              <div class="h-12 w-12 rounded-2xl bg-emerald-500/10 text-emerald-400 flex items-center justify-center border border-emerald-500/20">
                <Camera class="w-6 h-6" aria-hidden="true" />
              </div>
              <h4 class="text-sm font-bold text-white">Leitor de QR Code</h4>
              <p class="text-xs text-slate-400 max-w-xs text-pretty">
                Inicie a câmera para fazer a leitura em tempo real ou tire uma foto do cupom fiscal:
              </p>
              <div class="w-full max-w-xs space-y-2 pt-1">
                <button
                  type="button"
                  onclick={startScanner}
                  class="w-full px-5 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-bold text-xs shadow-lg shadow-emerald-500/20 transition-colors flex items-center justify-center gap-2 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                >
                  <Camera class="w-4 h-4" aria-hidden="true" />
                  <span>Iniciar Câmera ao Vivo</span>
                </button>
                <button
                  type="button"
                  onclick={triggerFileInput}
                  class="w-full px-4 py-2 rounded-xl border border-slate-700 bg-slate-900 hover:bg-slate-800 text-slate-200 text-xs font-semibold transition-colors flex items-center justify-center gap-2 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
                >
                  <Upload class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
                  <span>Tirar Foto / Carregar Imagem</span>
                </button>
              </div>
            </div>
          {/if}

          <!-- Overlay Guia com Cantos de Foco & Linha Laser (quando a câmera está ativa) -->
          {#if isScanning}
            <div class="absolute inset-0 pointer-events-none flex items-center justify-center" aria-hidden="true">
              <div class="w-[72%] h-[72%] relative border border-emerald-500/40 rounded-2xl shadow-[0_0_0_9999px_rgba(2,6,23,0.65)]">
                <!-- Cantos Marcadores -->
                <div class="absolute -top-0.5 -left-0.5 w-6 h-6 border-t-3 border-l-3 border-emerald-400 rounded-tl-xl"></div>
                <div class="absolute -top-0.5 -right-0.5 w-6 h-6 border-t-3 border-r-3 border-emerald-400 rounded-tr-xl"></div>
                <div class="absolute -bottom-0.5 -left-0.5 w-6 h-6 border-b-3 border-l-3 border-emerald-400 rounded-bl-xl"></div>
                <div class="absolute -bottom-0.5 -right-0.5 w-6 h-6 border-b-3 border-r-3 border-emerald-400 rounded-br-xl"></div>

                <!-- Linha animada de varredura laser -->
                <div class="absolute left-0 right-0 h-0.5 bg-gradient-to-r from-transparent via-emerald-400 to-transparent shadow-[0_0_8px_#10b981] animate-laser-scan"></div>
              </div>
            </div>
          {/if}
        </div>

        <!-- Barra de Controles Inferior -->
        <div class="w-full max-w-sm flex items-center justify-between gap-2 mt-3 pt-2">
          <!-- Alternar Câmera (se houver mais de uma) -->
          {#if cameras.length > 1}
            <div class="relative flex-1">
              <select
                id="camera-select"
                aria-label="Selecionar câmera"
                value={selectedCameraId}
                onchange={(e) => switchCamera((e.target as HTMLSelectElement).value)}
                class="w-full px-2.5 py-1.5 bg-slate-950 border border-slate-800 rounded-xl text-[11px] text-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 cursor-pointer"
              >
                {#each cameras as cam, idx}
                  <option value={cam.id}>
                    {cam.label || `Câmera ${idx + 1}`}
                  </option>
                {/each}
              </select>
            </div>
          {/if}

          <!-- Botão Lanterna (Torch) -->
          {#if hasTorch}
            <button
              type="button"
              onclick={toggleTorch}
              aria-label={isTorchOn ? "Desligar lanterna" : "Ligar lanterna"}
              class="px-3 py-1.5 rounded-xl border border-slate-800 hover:border-slate-700 bg-slate-950 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 {isTorchOn ? 'text-amber-300 border-amber-500/50 bg-amber-500/10' : 'text-slate-400'}"
              title="Ligar/Desligar Flash"
            >
              {#if isTorchOn}
                <Zap class="w-3.5 h-3.5 fill-amber-400" aria-hidden="true" />
                <span>Flash Ligado</span>
              {:else}
                <ZapOff class="w-3.5 h-3.5" aria-hidden="true" />
                <span>Flash</span>
              {/if}
            </button>
          {/if}

          <!-- Botão Foto / Galeria (Sempre disponível como rota 100% confiável) -->
          <button
            type="button"
            onclick={triggerFileInput}
            aria-label="Tirar foto com o app de câmera do celular ou escolher da galeria"
            class="px-3.5 py-1.5 rounded-xl border border-slate-800 hover:border-slate-700 bg-slate-950 hover:bg-slate-800/60 text-slate-300 hover:text-white text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer ml-auto focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
            title="Tirar foto com o app de câmera do celular ou escolher da galeria"
          >
            <Camera class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
            <span>Tirar Foto / Galeria</span>
          </button>
        </div>
      {/if}
    </div>

    <!-- Footer Informativo -->
    <div class="px-5 py-2.5 bg-slate-950/60 border-t border-slate-800/80 text-[11px] text-slate-400 flex items-center justify-between">
      <span class="flex items-center gap-1 text-slate-400">
        <CheckCircle2 class="w-3 h-3 text-emerald-400" aria-hidden="true" /> Compatível com NFC-e e NF-e de todos os estados (SP, MG, RS, PR, etc.)
      </span>
      <button
        type="button"
        onclick={onclose}
        class="text-slate-400 hover:text-white transition-colors cursor-pointer text-xs focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 rounded px-1.5 py-0.5"
      >
        Fechar
      </button>
    </div>
  </div>
</div>

<style>
  @keyframes laser-scan {
    0% {
      top: 5%;
      opacity: 0.2;
    }
    50% {
      opacity: 1;
    }
    100% {
      top: 95%;
      opacity: 0.2;
    }
  }

  .animate-laser-scan {
    animation: laser-scan 2s ease-in-out infinite;
  }

  @media (prefers-reduced-motion: reduce) {
    .animate-laser-scan {
      animation: none;
      top: 50%;
      opacity: 0.8;
    }
  }
</style>
