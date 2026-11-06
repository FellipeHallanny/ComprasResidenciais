<script lang="ts">
  import { onMount } from 'svelte';
  import { X, MapPin, Search, Navigation, Check, ExternalLink } from '@lucide/svelte';
  import L from 'leaflet';
  import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
  import markerIcon from 'leaflet/dist/images/marker-icon.png';
  import markerShadow from 'leaflet/dist/images/marker-shadow.png';

  // Fix default leaflet icons in Vite
  delete (L.Icon.Default.prototype as any)._getIconUrl;
  L.Icon.Default.mergeOptions({
    iconUrl: markerIcon,
    iconRetinaUrl: markerIcon2x,
    shadowUrl: markerShadow,
  });

  interface Props {
    initialAddress?: string;
    onselect: (address: string) => void;
    onclose: () => void;
  }

  let { initialAddress = '', onselect, onclose }: Props = $props();

  let mapContainer: HTMLDivElement;
  let map: L.Map | null = null;
  let marker: L.Marker | null = null;

  let searchQuery = $state('');
  let isSearching = $state(false);
  let isReverseGeocoding = $state(false);
  let selectedAddress = $state('');
  let currentCoords = $state<{ lat: number; lng: number } | null>(null);

  $effect(() => {
    selectedAddress = initialAddress;
  });

  async function reverseGeocode(lat: number, lng: number) {
    currentCoords = { lat, lng };
    isReverseGeocoding = true;
    try {
      const res = await fetch(
        `https://nominatim.openstreetmap.org/reverse?format=json&lat=${lat}&lon=${lng}&addressdetails=1`,
        {
          headers: {
            'Accept-Language': 'pt-BR,pt;q=0.9',
          },
        }
      );
      if (res.ok) {
        const data = await res.json();
        const addr = data.address || {};
        const road = addr.road || addr.street || addr.pedestrian || addr.suburb || '';
        const houseNumber = addr.house_number ? `, ${addr.house_number}` : '';
        const neighbourhood = addr.neighbourhood || addr.suburb || addr.quarter || '';
        const city = addr.city || addr.town || addr.municipality || addr.village || '';
        const state = addr.state || '';
        const postcode = addr.postcode ? ` - CEP ${addr.postcode}` : '';

        let parts: string[] = [];
        if (road) parts.push(`${road}${houseNumber}`);
        if (neighbourhood) parts.push(neighbourhood);
        if (city && state) parts.push(`${city} - ${state}`);
        else if (city) parts.push(city);
        if (postcode) parts.push(postcode.trim());

        if (parts.length > 0) {
          selectedAddress = parts.join(', ');
        } else if (data.display_name) {
          selectedAddress = data.display_name;
        }
      }
    } catch (err) {
      console.error('Erro ao resolver endereço:', err);
    } finally {
      isReverseGeocoding = false;
    }
  }

  function setMarkerLocation(lat: number, lng: number, zoom?: number) {
    if (!map) return;
    if (marker) {
      marker.setLatLng([lat, lng]);
    } else {
      marker = L.marker([lat, lng], { draggable: true }).addTo(map);
      marker.on('dragend', () => {
        if (!marker) return;
        const pos = marker.getLatLng();
        reverseGeocode(pos.lat, pos.lng);
      });
    }

    if (zoom) {
      map.setView([lat, lng], zoom);
    } else {
      map.panTo([lat, lng]);
    }

    reverseGeocode(lat, lng);
  }

  async function handleSearch(e: Event) {
    e.preventDefault();
    if (!searchQuery.trim()) return;

    isSearching = true;
    try {
      const res = await fetch(
        `https://nominatim.openstreetmap.org/search?format=json&q=${encodeURIComponent(
          searchQuery.trim()
        )}&countrycodes=br&limit=1`,
        {
          headers: {
            'Accept-Language': 'pt-BR,pt;q=0.9',
          },
        }
      );
      if (res.ok) {
        const data = await res.json();
        if (data && data.length > 0) {
          const item = data[0];
          const lat = parseFloat(item.lat);
          const lon = parseFloat(item.lon);
          setMarkerLocation(lat, lon, 16);
        } else {
          alert('Local não encontrado. Tente buscar pelo nome da rua, bairro ou cidade.');
        }
      }
    } catch (err) {
      console.error('Erro ao buscar localização:', err);
    } finally {
      isSearching = false;
    }
  }

  function handleCurrentLocation() {
    if (!navigator.geolocation) {
      alert('Geolocalização não é suportada pelo seu navegador.');
      return;
    }
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setMarkerLocation(pos.coords.latitude, pos.coords.longitude, 16);
      },
      (err) => {
        alert('Não foi possível obter sua localização atual: ' + err.message);
      }
    );
  }

  function openGoogleMaps() {
    if (currentCoords) {
      window.open(
        `https://www.google.com/maps/search/?api=1&query=${currentCoords.lat},${currentCoords.lng}`,
        '_blank'
      );
    } else if (selectedAddress) {
      window.open(
        `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(selectedAddress)}`,
        '_blank'
      );
    }
  }

  function confirmSelection() {
    if (!selectedAddress.trim()) {
      alert('Selecione um ponto no mapa para capturar o endereço.');
      return;
    }
    onselect(selectedAddress.trim());
  }

  onMount(() => {
    // Default fallback: Brasil / São Paulo
    const defaultLat = -23.5505;
    const defaultLng = -46.6333;

    map = L.map(mapContainer, {
      center: [defaultLat, defaultLng],
      zoom: 13,
    });

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution:
        '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
      maxZoom: 19,
    }).addTo(map);

    map.on('click', (e: L.LeafletMouseEvent) => {
      setMarkerLocation(e.latlng.lat, e.latlng.lng);
    });

    // If initial address exists, search for it
    if (initialAddress) {
      searchQuery = initialAddress;
      handleSearch(new Event('submit'));
    } else {
      // Try centering on user's current location
      if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(
          (pos) => {
            setMarkerLocation(pos.coords.latitude, pos.coords.longitude, 15);
          },
          () => {
            setMarkerLocation(defaultLat, defaultLng, 13);
          }
        );
      } else {
        setMarkerLocation(defaultLat, defaultLng, 13);
      }
    }

    return () => {
      if (map) {
        map.remove();
      }
    };
  });
</script>

<div class="fixed inset-0 z-50 bg-black/80 backdrop-blur-xs flex items-center justify-center p-3 sm:p-6 overflow-hidden overscroll-contain">
  <div
    role="dialog"
    aria-modal="true"
    aria-labelledby="location-picker-title"
    class="bg-[#0e131f] border border-[#232d44] rounded-xl w-full max-w-4xl h-[90vh] max-h-[780px] shadow-2xl flex flex-col overflow-hidden relative"
  >
    <!-- Modal Header -->
    <div class="px-5 py-4 border-b border-[#1b2234] flex items-center justify-between bg-[#0e131f] z-10 shrink-0">
      <div class="flex items-center gap-2.5">
        <div class="w-8 h-8 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 flex items-center justify-center">
          <MapPin class="w-4 h-4" aria-hidden="true" />
        </div>
        <div>
          <h3 id="location-picker-title" class="text-sm sm:text-base font-semibold text-white">Selecionar Localização da Loja</h3>
          <p class="text-xs text-slate-400">
            Clique no mapa ou busque pelo nome da rua para definir o endereço automaticamente
          </p>
        </div>
      </div>
      <button
        onclick={onclose}
        class="text-slate-400 hover:text-slate-200 p-1.5 rounded-md hover:bg-[#182030] transition-colors cursor-pointer focus:outline-none"
        title="Fechar"
        aria-label="Fechar"
      >
        <X class="w-4 h-4" aria-hidden="true" />
      </button>
    </div>

    <!-- Search & Control Toolbar -->
    <div class="p-3.5 bg-[#090d16] border-b border-[#1b2234] flex flex-col sm:flex-row gap-2 shrink-0 z-10">
      <form onsubmit={handleSearch} class="flex-1 flex gap-2">
        <div class="relative flex-1">
          <label for="location-search-input" class="sr-only">Buscar endereço</label>
          <input
            id="location-search-input"
            type="text"
            autocomplete="off"
            bind:value={searchQuery}
            placeholder="Buscar por endereço, avenida, bairro ou ponto de referência…"
            class="w-full pl-9 pr-4 py-2 bg-[#0e131f] border border-[#1b2234] focus:border-emerald-500/60 rounded-lg text-slate-100 text-xs focus:outline-none transition-colors"
          />
          <Search class="w-3.5 h-3.5 text-slate-400 absolute left-3 top-2.5 pointer-events-none" aria-hidden="true" />
        </div>
        <button
          type="submit"
          disabled={isSearching || !searchQuery.trim()}
          class="px-3.5 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-semibold text-xs shadow-xs transition-all cursor-pointer flex items-center gap-1.5 shrink-0 focus:outline-none"
        >
          {#if isSearching}
            <div class="w-3.5 h-3.5 border-2 border-slate-950 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
          {:else}
            <span>Buscar</span>
          {/if}
        </button>
      </form>

      <button
        type="button"
        onclick={handleCurrentLocation}
        class="px-3 py-2 rounded-lg border border-[#1b2234] bg-[#0e131f] hover:bg-[#182030] text-slate-300 text-xs font-medium flex items-center gap-1.5 transition-colors cursor-pointer shrink-0 focus:outline-none"
        title="Usar minha localização atual"
        aria-label="Usar minha localização atual"
      >
        <Navigation class="w-3.5 h-3.5 text-emerald-400" aria-hidden="true" />
        <span>Meu Local</span>
      </button>
    </div>

    <!-- Leaflet Map Container -->
    <div class="flex-1 w-full h-full relative">
      <div bind:this={mapContainer} class="w-full h-full z-0"></div>

      <!-- Hint floating on top of map -->
      <div class="absolute top-3 left-1/2 -translate-x-1/2 bg-[#0e131f]/95 backdrop-blur-md border border-[#232d44] text-slate-300 text-xs px-3.5 py-1.5 rounded-lg shadow-lg pointer-events-none z-[400] flex items-center gap-1.5">
        <MapPin class="w-3.5 h-3.5 text-emerald-400 shrink-0" aria-hidden="true" />
        <span>Clique no mapa ou arraste o marcador para capturar o endereço</span>
      </div>
    </div>

    <!-- Bottom Selected Address Bar & Confirmation -->
    <div class="p-4 bg-[#0e131f] border-t border-[#1b2234] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 shrink-0 z-10">
      <div class="flex-1 min-w-0" aria-live="polite">
        <div class="text-[11px] text-slate-400 font-medium mb-0.5 flex items-center gap-2">
          <span>Endereço Identificado:</span>
          {#if isReverseGeocoding}
            <span class="text-emerald-400 flex items-center gap-1">
              <div class="w-3 h-3 border border-emerald-400 border-t-transparent rounded-full animate-spin" aria-hidden="true"></div>
              <span>Obtendo endereço…</span>
            </span>
          {/if}
        </div>
        <div class="text-xs sm:text-sm font-semibold text-white truncate" title={selectedAddress}>
          {selectedAddress || 'Nenhum local selecionado no mapa'}
        </div>
        {#if currentCoords}
          <div class="text-[11px] text-slate-500 font-mono mt-0.5 tabular-nums">
            Coord: {currentCoords.lat.toFixed(5)}, {currentCoords.lng.toFixed(5)}
          </div>
        {/if}
      </div>

      <div class="flex items-center gap-2 w-full sm:w-auto shrink-0 justify-end">
        <button
          type="button"
          onclick={openGoogleMaps}
          class="px-3 py-2 rounded-lg border border-[#1b2234] hover:border-[#27324b] bg-[#141b29] text-slate-300 text-xs font-medium flex items-center gap-1.5 transition-colors cursor-pointer focus:outline-none"
          title="Ver no Google Maps"
        >
          <ExternalLink class="w-3.5 h-3.5 text-slate-400" aria-hidden="true" />
          <span>Google Maps</span>
        </button>

        <button
          type="button"
          onclick={confirmSelection}
          disabled={!selectedAddress}
          class="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-400 disabled:opacity-50 text-slate-950 font-semibold text-xs shadow-xs transition-all flex items-center gap-1.5 cursor-pointer focus:outline-none"
        >
          <Check class="w-3.5 h-3.5 stroke-[3]" aria-hidden="true" />
          <span>Usar este Endereço</span>
        </button>
      </div>
    </div>

  </div>
</div>
