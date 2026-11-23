<script lang="ts" module>
  // Stroke icons on a 24×24 grid. Keys match the `icon` field the API sends
  // for nav entries; unknown keys fall back to a dot so a new module never
  // breaks the sidebar.
  const PATHS: Record<string, string> = {
    home: '<path d="M3 11 12 4l9 7"/><path d="M5 10v10h5v-6h4v6h5V10"/>',
    receipt: '<path d="M6 3h12v18l-3-2-3 2-3-2-3 2z"/><path d="M9 8h6M9 12h6M9 16h3"/>',
    history: '<path d="M3 12a9 9 0 1 0 3-6.7"/><path d="M3 4v5h5"/><path d="M12 8v4l3 2"/>',
    undo: '<path d="M9 14 4 9l5-5"/><path d="M4 9h11a5 5 0 0 1 0 10h-3"/>',
    pause: '<circle cx="12" cy="12" r="9"/><path d="M10 9v6M14 9v6"/>',
    box: '<path d="M3 7.5 12 3l9 4.5v9L12 21l-9-4.5z"/><path d="m3 7.5 9 4.5 9-4.5M12 12v9"/>',
    tags: '<path d="M3 12V4h8l9 9-8 8z"/><circle cx="7.5" cy="8.5" r="1.5"/>',
    tag: '<path d="M3 12V4h8l9 9-8 8z"/><circle cx="7.5" cy="8.5" r="1.5"/>',
    layers: '<path d="m12 3 9 5-9 5-9-5z"/><path d="m3 13 9 5 9-5"/>',
    arrows: '<path d="M7 4v16M4 7l3-3 3 3"/><path d="M17 20V4M14 17l3 3 3-3"/>',
    transfer: '<path d="M4 8h14l-4-4"/><path d="M20 16H6l4 4"/>',
    clipboard:
      '<rect x="5" y="4" width="14" height="17" rx="2"/><path d="M9 4V3h6v1"/><path d="M9 10h6M9 14h6M9 18h3"/>',
    percent: '<path d="M19 5 5 19"/><circle cx="7" cy="7" r="2.5"/><circle cx="17" cy="17" r="2.5"/>',
    truck:
      '<path d="M3 6h11v10H3zM14 9h4l3 3v4h-7"/><circle cx="7" cy="18" r="2"/><circle cx="17" cy="18" r="2"/>',
    file: '<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4M9 12h6M9 16h6"/>',
    inbox: '<path d="M3 13h5l2 3h4l2-3h5"/><path d="M5 5h14l2 8v6H3v-6z"/>',
    phone: '<rect x="7" y="2" width="10" height="20" rx="2"/><path d="M11 18h2"/>',
    wallet:
      '<path d="M4 6h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H4z"/><path d="M4 6V5a2 2 0 0 1 2-2h10v3"/><path d="M15 13h2"/>',
    chart: '<path d="M4 4v16h16"/><path d="m7 15 4-4 3 3 5-6"/>',
    'chart-bars': '<path d="M4 4v16h16"/><path d="M8 16v-4M12 16V8M16 16v-6"/>',
    calculator:
      '<rect x="5" y="3" width="14" height="18" rx="2"/><path d="M8 7h8M8 12h.01M12 12h.01M16 12h.01M8 16h.01M12 16h.01M16 16h.01"/>',
    users:
      '<circle cx="9" cy="8" r="3.5"/><path d="M2.5 20a6.5 6.5 0 0 1 13 0"/><path d="M16 4.5a3.5 3.5 0 0 1 0 7M18 14a6 6 0 0 1 3.5 6"/>',
    key: '<circle cx="8" cy="15" r="4"/><path d="m11 12 9-9M17 6l3 3M14 9l2 2"/>',
    building:
      '<path d="M4 21V5l8-2v18M12 8h8v13"/><path d="M8 8h.01M8 12h.01M8 16h.01M16 12h.01M16 16h.01M2 21h20"/>',
    monitor: '<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>',
    shield: '<path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6z"/><path d="m9 12 2 2 4-4"/>',
    store:
      '<path d="M4 9 5.5 4h13L20 9"/><path d="M4 9a2.7 2.7 0 0 0 5.3 0 2.7 2.7 0 0 0 5.4 0A2.7 2.7 0 0 0 20 9"/><path d="M5 11v9h14v-9M10 20v-5h4v5"/>',
    rocket:
      '<path d="M5 15c-1.5 1.5-2 5-2 5s3.5-.5 5-2"/><path d="M9 15 6 12l3-3c3-3 7-5 12-6-1 5-3 9-6 12l-3 3z"/><circle cx="14.5" cy="9.5" r="1.5"/>',
    puzzle:
      '<path d="M10 3h4v3a2 2 0 1 0 4 0V3h3v7h-3a2 2 0 1 0 0 4h3v7h-7v-3a2 2 0 1 0-4 0v3H3v-7h3a2 2 0 1 0 0-4H3V3z"/>',
    plug: '<path d="M9 3v5M15 3v5"/><path d="M6 8h12v3a6 6 0 0 1-12 0z"/><path d="M12 17v4"/>',
    printer:
      '<path d="M7 9V3h10v6"/><rect x="3" y="9" width="18" height="8" rx="2"/><path d="M7 14h10v7H7z"/>',
    cog: '<circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9 7 7M17 17l2.1 2.1M4.9 19.1 7 17M17 7l2.1-2.1"/>',
    user: '<circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/>',
    logout: '<path d="M15 4h4v16h-4"/><path d="M10 8l-4 4 4 4M6 12h10"/>',
    cart: '<path d="M2 3h3l2.5 12h11L21 7H6.2"/><circle cx="9" cy="19.5" r="1.5"/><circle cx="17" cy="19.5" r="1.5"/>',
    calendar: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/>',
    'chevron-down': '<path d="m6 9 6 6 6-6"/>',
    'chevron-right': '<path d="m9 6 6 6-6 6"/>',
    'arrow-right': '<path d="M5 12h14M13 6l6 6-6 6"/>',
    'arrow-up': '<path d="M12 19V5M6 11l6-6 6 6"/>',
    'arrow-down': '<path d="M12 5v14M6 13l6 6 6-6"/>',
    menu: '<path d="M4 6h16M4 12h16M4 18h16"/>',
    x: '<path d="M6 6l12 12M18 6 6 18"/>',
    alert: '<path d="M12 3 2 20h20z"/><path d="M12 9v5M12 17h.01"/>',
    info: '<circle cx="12" cy="12" r="9"/><path d="M12 11v6M12 7h.01"/>',
    check: '<path d="m5 12 5 5 9-10"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    dashboard: '<path d="M4 20V10M10 20V4M16 20v-7M22 20H2"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="m20.5 20.5-4.5-4.5"/>',
    filter: '<path d="M4 5h16l-6.4 7.4V19l-3.2 2v-8.6z"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    minus: '<path d="M5 12h14"/>',
    pencil: '<path d="M17.5 3.5a2.1 2.1 0 0 1 3 3L8 19l-4.5 1.5L5 16z"/><path d="m14.5 6.5 3 3"/>',
    trash:
      '<path d="M4 7h16M9.5 7V5.5A1.5 1.5 0 0 1 11 4h2a1.5 1.5 0 0 1 1.5 1.5V7"/><path d="M6.5 7l1 13h9l1-13"/><path d="M10 11v5M14 11v5"/>',
    download: '<path d="M12 4v10m0 0 4-4m-4 4-4-4"/><path d="M4.5 17v2.5h15V17"/>',
    upload: '<path d="M12 14V4m0 0-4 4m4-4 4 4"/><path d="M4.5 17v2.5h15V17"/>',
    dots: '<circle cx="12" cy="5.5" r="1.5"/><circle cx="12" cy="12" r="1.5"/><circle cx="12" cy="18.5" r="1.5"/>',
    'dots-h':
      '<circle cx="5.5" cy="12" r="1.5"/><circle cx="12" cy="12" r="1.5"/><circle cx="18.5" cy="12" r="1.5"/>',
    eye: '<path d="M2.5 12S6 5.8 12 5.8 21.5 12 21.5 12 18 18.2 12 18.2 2.5 12 2.5 12z"/><circle cx="12" cy="12" r="2.6"/>',
    lock: '<rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>',
    unlock: '<rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 7.8-1.4"/>',
    refresh: '<path d="M20.5 12a8.5 8.5 0 1 1-2.5-6"/><path d="M21 3.5V8h-4.5"/>',
    bell: '<path d="M6.2 9.2a5.8 5.8 0 0 1 11.6 0c0 4.6 2 5.8 2 5.8H4.2s2-1.2 2-5.8"/><path d="M10.3 18.7a1.9 1.9 0 0 0 3.4 0"/>',
    banknote:
      '<rect x="3" y="6" width="18" height="12" rx="2"/><circle cx="12" cy="12" r="2.5"/><path d="M6.5 9.5h.01M17.5 14.5h.01"/>',
    'credit-card': '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3 9.8h18"/><path d="M7 15h4"/>',
    landmark:
      '<path d="m12 3.5 8 4.5H4z"/><path d="M5.5 10v7M9.8 10v7M14.2 10v7M18.5 10v7"/><path d="M3.5 20.5h17"/>',
    book: '<path d="M5 4.5A1.5 1.5 0 0 1 6.5 3H19v15H6.5A1.5 1.5 0 0 0 5 19.5z"/><path d="M5 19.5A1.5 1.5 0 0 1 6.5 18H19v3H6.5A1.5 1.5 0 0 1 5 19.5z"/>',
    'check-circle': '<circle cx="12" cy="12" r="9"/><path d="m8.5 12.4 2.4 2.4 4.8-5.3"/>',
    'x-circle': '<circle cx="12" cy="12" r="9"/><path d="m9 9 6 6m0-6-6 6"/>',
    'chevron-left': '<path d="m15 6-6 6 6 6"/>',
    'chevron-up': '<path d="m6 15 6-6 6 6"/>',
    'arrow-left': '<path d="M19 12H5M11 6l-6 6 6 6"/>',
    star: '<path d="m12 3.5 2.7 5.6 6.1.8-4.5 4.2 1.1 6-5.4-3-5.4 3 1.1-6L3.5 9.9l6.1-.8z"/>',
    image:
      '<rect x="3" y="5" width="18" height="14" rx="2"/><circle cx="8.6" cy="10" r="1.5"/><path d="m5 18 4.8-4.8 3 3 3.4-3.4L21 17"/>',
    grid: '<rect x="4" y="4" width="7" height="7" rx="1.4"/><rect x="13" y="4" width="7" height="7" rx="1.4"/><rect x="4" y="13" width="7" height="7" rx="1.4"/><rect x="13" y="13" width="7" height="7" rx="1.4"/>',
    scale:
      '<path d="M7 21h10"/><rect x="4" y="11" width="16" height="8" rx="2"/><path d="M9.5 11V7.5h5V11"/><circle cx="12" cy="15" r="1.8"/>',
    utensils:
      '<path d="M6 3v5.5a2.5 2.5 0 0 0 5 0V3"/><path d="M8.5 3v18"/><path d="M17.5 3c-1.7.6-3 2.8-3 5.2 0 2.1 1.2 3.8 3 3.8v9"/><path d="M17.5 3v9"/>',
    basket:
      '<path d="m5.5 9.5 2.3 10.5h8.4l2.3-10.5"/><path d="M3.5 9.5h17"/><path d="m9.5 9.5 2.5-5 2.5 5"/><path d="M10 13.5v4M14 13.5v4"/>',
    cup: '<path d="M5.5 4h11v9.5a5 5 0 0 1-5 5h-1a5 5 0 0 1-5-5z"/><path d="M16.5 6.5h1.8a2.7 2.7 0 0 1 0 5.4h-1.8"/>',
    bottle:
      '<path d="M10 3h4v2.6l1.7 2.5V19a2 2 0 0 1-2 2h-3.4a2 2 0 0 1-2-2V8.1L10 5.6z"/><path d="M8.3 12h7.4"/>',
    loaf: '<path d="M5 9.5a7 4.5 0 0 1 14 0V17a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2z"/><path d="M8.5 12.5c.4 1 .4 2.5 0 3.5M12 12.5c.4 1 .4 2.5 0 3.5M15.5 12.5c.4 1 .4 2.5 0 3.5"/>',
    drumstick:
      '<path d="M15.9 8.1a4.4 4.4 0 0 0-6.3 6.3l1.2 1.2-4.4 4.4"/><circle cx="4.9" cy="17.8" r="1.7"/><circle cx="6.8" cy="20.3" r="1.4"/>',
    fish: '<path d="M6.5 12c2.8-3.7 7.8-3.7 11 0-3.2 3.7-8.2 3.7-11 0z"/><path d="m17.5 12 3.5-3v6z"/><path d="M9.3 11h.01"/>',
    pill: '<path d="m10.6 3.6 9.9 9.9a4.95 4.95 0 0 1-7 7l-9.9-9.9a4.95 4.95 0 0 1 7-7z"/><path d="m8.6 8.6 6.8 6.8"/>',
    spray:
      '<path d="M10 8h4v2.5h-4z"/><path d="M8.2 10.5h7.6l1.2 9.5H7z"/><path d="M11 8V5h2v3"/><path d="M16.5 5.5 19 4M17 8h2.5M16.5 3.2 18 1.8"/>',
    shirt: '<path d="M9 3.5 4.5 6.2l1.8 3.4 2-1V20h7.5V8.6l2 1 1.7-3.4L15 3.5a3 3 0 0 1-6 0z"/>',
    paw: '<circle cx="7" cy="8.2" r="1.6"/><circle cx="12" cy="6.2" r="1.6"/><circle cx="17" cy="8.2" r="1.6"/><path d="M12 11.5c-2.9 0-4.8 2.3-4.8 4.3 0 1.3 1 2.2 2.3 2.2 1 0 1.6-.4 2.5-.4s1.5.4 2.5.4c1.3 0 2.3-.9 2.3-2.2 0-2-1.9-4.3-4.8-4.3z"/>',
    baby: '<circle cx="12" cy="12" r="8.5"/><path d="M12 3.5c0 1.8 1 2.6 2.2 2.6"/><path d="M9.2 12.8h.01M14.8 12.8h.01"/><path d="M9.3 15.8c1.7 1 3.7 1 5.4 0"/>',
    carrot: '<path d="M15.5 9.5C11 11.5 8 14.5 5.5 20.5 11 18 14 15 15.5 9.5z"/><path d="M15.5 9.5c1.5-4.5 4.5-5.5 6-5.5-.4 1.7-2.2 4.6-4.2 6.8"/>',
    snowflake: '<path d="M12 3v18M4.3 7.5l15.4 9M19.7 7.5l-15.4 9"/><path d="M12 3 9.8 4.8M12 3l2.2 1.8M12 21l-2.2-1.8M12 21l2.2-1.8"/>',
    wrench:
      '<path d="M17.5 3.2a4.9 4.9 0 0 0-6.3 6.3L4 16.7a2.2 2.2 0 0 0 3.1 3.1l7.2-7.2a4.9 4.9 0 0 0 6.3-6.3l-3 3-2.6-2.6z"/>',
    cookie: '<circle cx="12" cy="12" r="8.5"/><path d="M9 9.7h.01M14.8 9.3h.01M9.7 14.7h.01M14 15.2h.01"/><path d="M16.4 12.7c.2-2-1.2-3.5-2.6-3.6"/>',
    barcode: '<path d="M4.5 6v12M7.5 6v12M10.5 6v7M13.5 6v12M16.5 6v7M19.5 6v12"/>',
    wifi: '<path d="M2.8 9.3a15.5 15.5 0 0 1 18.4 0M5.8 12.8a10.5 10.5 0 0 1 12.4 0M8.8 16.2a5.5 5.5 0 0 1 6.4 0"/><circle cx="12" cy="19.3" r="1"/>',
    'wifi-off':
      '<path d="M4.5 4.5l15 15"/><path d="M8.8 16.2a5.5 5.5 0 0 1 6.4 0"/><path d="M5.8 12.8a10.5 10.5 0 0 1 4.6-2.5M14.4 10.9a10.5 10.5 0 0 1 3.8 1.9M2.8 9.3A15.5 15.5 0 0 1 8.4 6M15.4 5a15.5 15.5 0 0 1 5.8 4.3"/><circle cx="12" cy="19.3" r="1"/>',
    mail: '<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3.5 7.5 8.5 6 8.5-6"/>',
    wine: '<path d="M7 3h10c0 4.8-2.2 7.6-5 7.6S7 7.8 7 3z"/><path d="M12 10.6V20M8.5 21h7"/>',
    glass: '<path d="M6.5 3h11l-1.4 17.5h-8.2z"/><path d="M6.8 8h10.4"/>',
    'git-branch':
      '<circle cx="6.5" cy="6" r="2.4"/><circle cx="6.5" cy="18" r="2.4"/><circle cx="17.5" cy="8" r="2.4"/><path d="M6.5 8.4v7.2M17.5 10.4c0 4-4.5 4.3-8.5 4.6"/>',
    'id-badge': '<rect x="4.5" y="3" width="15" height="18" rx="2"/><circle cx="12" cy="9.8" r="2.4"/><path d="M8.3 16.5a4 4 0 0 1 7.4 0"/>',
    list: '<path d="M8.5 6h12M8.5 12h12M8.5 18h12"/><path d="M3.6 6h.01M3.6 12h.01M3.6 18h.01"/>',
    drawer: '<rect x="3" y="10" width="18" height="9.5" rx="2"/><path d="M8 10V6.5h8V10"/><path d="M10.3 15h3.4"/>',
    gift: '<rect x="4" y="9.5" width="16" height="10.5" rx="1.5"/><path d="M12 9.5V20M4 13.5h16"/><path d="M12 9.5c-1.4 0-3.6-.7-3.6-2.4 0-1.6 2-2.1 3.6-.2 1.6-1.9 3.6-1.4 3.6.2 0 1.7-2.2 2.4-3.6 2.4z"/>',
    help: '<circle cx="12" cy="12" r="9"/><path d="M9.6 9.6a2.5 2.5 0 1 1 3.6 2.2c-.8.4-1.2.9-1.2 1.9"/><path d="M12 17h.01"/>',
    power: '<path d="M12 3.5V11"/><path d="M6.8 6.2a8 8 0 1 0 10.4 0"/>',
    copy: '<rect x="9" y="9" width="11" height="11" rx="2"/><path d="M5.5 15v-9a2 2 0 0 1 2-2h9"/>',
    qr: '<rect x="4" y="4" width="6.5" height="6.5" rx="1"/><rect x="13.5" y="4" width="6.5" height="6.5" rx="1"/><rect x="4" y="13.5" width="6.5" height="6.5" rx="1"/><path d="M14 14h2.5v2.5H14zM20 14h.01M14 20h.01M17.5 20H20"/>',
    globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><ellipse cx="12" cy="12" rx="4.5" ry="9"/>',
    'user-check': '<circle cx="10" cy="8" r="3.5"/><path d="M3.5 20a6.5 6.5 0 0 1 13 0"/><path d="m15.5 12.5 1.8 1.8 3.4-3.8"/>',
    send: '<path d="m3.5 11.2 17-7.2-7.2 17-2.6-7.4z"/><path d="m20.5 4-9.8 7.4"/>',
    save: '<path d="M5 4h11.5L20 7.5V20H5z"/><path d="M8 4v4.5h7.5V4"/><path d="M8 20v-6.5h8V20"/>',
    sparkle:
      '<path d="M12 3v4M12 17v4M3 12h4M17 12h4M6 6l2.5 2.5M15.5 15.5 18 18M6 18l2.5-2.5M15.5 8.5 18 6"/>',
  };
  const FALLBACK = '<circle cx="12" cy="12" r="2.5"/>';
</script>

<script lang="ts">
  interface Props {
    name: string;
    size?: number;
    strokeWidth?: number;
    /** Accessible label; omit for decorative icons. */
    label?: string;
  }
  let { name, size = 20, strokeWidth = 1.8, label }: Props = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width={strokeWidth}
  stroke-linecap="round"
  stroke-linejoin="round"
  role={label ? 'img' : undefined}
  aria-label={label}
  aria-hidden={label ? undefined : 'true'}
  class="icon"
>
  <!-- eslint-disable-next-line svelte/no-at-html-tags -- static, local markup -->
  {@html PATHS[name] ?? FALLBACK}
</svg>

<style>
  .icon {
    flex: none;
    display: block;
  }
</style>
