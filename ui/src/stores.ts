import { derived, writable } from 'svelte/store';
import Cookies from 'js-cookie';
import { AppConfig, rtlLanguages } from './config';
import { locale } from './i18n/i18n-svelte';
import type { GlobalAlert } from './types/global-alerts';
import type { SessionUser } from './types/user';

const { PathPrefix, CookieName } = AppConfig;
const cookiePath = `${PathPrefix}/`;

declare global {
  let ActiveAlerts: any;
}

function initWarrior() {
  const { subscribe, set, update } = writable(JSON.parse(Cookies.get(CookieName) || '{}'));

  return {
    subscribe,
    create: (warrior: SessionUser) => {
      Cookies.set(CookieName, JSON.stringify(warrior), {
        expires: 365,
        SameSite: 'strict',
        path: cookiePath,
      });
      set(warrior);
    },
    update: (warrior: SessionUser) => {
      Cookies.set(CookieName, JSON.stringify(warrior), {
        expires: 365,
        SameSite: 'strict',
        path: cookiePath,
      });
      update(w => (w = warrior));
    },
    delete: () => {
      Cookies.remove(CookieName, { path: cookiePath });
      set({});
    },
  };
}

export const user = initWarrior();

function initActiveAlerts() {
  const activeAlerts = typeof ActiveAlerts != 'undefined' ? ActiveAlerts : [];
  const { subscribe, update } = writable(activeAlerts);

  return {
    subscribe,
    update: (alerts: GlobalAlert[]) => {
      update(a => (a = alerts));
    },
  };
}

export const activeAlerts = initActiveAlerts();

function initDismissedAlerts() {
  const dismissKey = 'dismissed_alerts';
  const dismissedAlerts = JSON.parse(localStorage.getItem(dismissKey) || '[]') as string[];
  const { subscribe, update } = writable(dismissedAlerts);

  return {
    subscribe,
    dismiss: (actives: string[], dismisses: string[]) => {
      // Only store valid alert IDs
      const validAlerts = actives;
      let alertsToDismiss = [...dismisses.filter(id => validAlerts.includes(id))];
      localStorage.setItem(dismissKey, JSON.stringify(alertsToDismiss));
      update((a: any) => (a = alertsToDismiss));
    },
  };
}

export const dir = derived(locale, $locale => (rtlLanguages.includes($locale) ? 'rtl' : 'ltr'));

export const dismissedAlerts = initDismissedAlerts();

export interface Branding {
  brand_name: string;
  primary_color: string;
  accent_color: string;
  dark_color: string;
  has_logo_main: boolean;
  has_logo_dark: boolean;
  has_favicon: boolean;
  has_email_logo: boolean;
}

const emptyBranding: Branding = {
  brand_name: '',
  primary_color: '',
  accent_color: '',
  dark_color: '',
  has_logo_main: false,
  has_logo_dark: false,
  has_favicon: false,
  has_email_logo: false,
};

// hexToRgbTriplet returns "R G B" (space-separated) suitable for CSS custom
// properties consumed by Tailwind's rgb(var(--x) / <alpha>) form. Returns ""
// when the input isn't a valid hex color so the caller can keep the default.
function hexToRgbTriplet(hex: string): string {
  if (!hex || !hex.startsWith('#')) return '';
  let body = hex.slice(1);
  if (body.length === 3) {
    body = body
      .split('')
      .map(c => c + c)
      .join('');
  }
  if (body.length !== 6 && body.length !== 8) return '';
  const r = parseInt(body.slice(0, 2), 16);
  const g = parseInt(body.slice(2, 4), 16);
  const b = parseInt(body.slice(4, 6), 16);
  if (Number.isNaN(r) || Number.isNaN(g) || Number.isNaN(b)) return '';
  return `${r} ${g} ${b}`;
}

function applyBrandingToDocument(b: Branding) {
  if (typeof document === 'undefined') return;
  const root = document.documentElement;
  const map: Array<[string, string]> = [
    ['--brand-primary-rgb', b.primary_color],
    ['--brand-accent-rgb', b.accent_color],
    ['--brand-dark-rgb', b.dark_color],
  ];
  for (const [varName, hex] of map) {
    const rgb = hexToRgbTriplet(hex);
    if (rgb) {
      root.style.setProperty(varName, rgb);
    } else {
      root.style.removeProperty(varName);
    }
  }
}

function initBranding() {
  const { subscribe, set } = writable<Branding>(emptyBranding);

  async function load() {
    try {
      const res = await fetch(`${PathPrefix}/api/branding`, { credentials: 'same-origin' });
      if (!res.ok) return;
      const result = await res.json();
      const b: Branding = { ...emptyBranding, ...result.data };
      applyBrandingToDocument(b);
      set(b);
    } catch {
      // Branding is best-effort; defaults already applied via CSS.
    }
  }

  return {
    subscribe,
    load,
    set: (b: Branding) => {
      applyBrandingToDocument(b);
      set(b);
    },
  };
}

export const branding = initBranding();
