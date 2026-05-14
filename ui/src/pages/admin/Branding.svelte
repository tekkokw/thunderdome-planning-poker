<script lang="ts">
  import { onMount } from 'svelte';

  import AdminPageLayout from '../../components/admin/AdminPageLayout.svelte';
  import SolidButton from '../../components/global/SolidButton.svelte';
  import HollowButton from '../../components/global/HollowButton.svelte';
  import TextInput from '../../components/forms/TextInput.svelte';
  import { user, branding } from '../../stores';
  import { appRoutes, PathPrefix } from '../../config';
  import { validateUserIsAdmin } from '../../validationUtils';

  import type { NotificationService } from '../../types/notifications';
  import type { ApiClient } from '../../types/apiclient';

  interface Props {
    xfetch: ApiClient;
    router: any;
    notifications: NotificationService;
  }

  let { xfetch, router, notifications }: Props = $props();

  type LogoVariant = 'main' | 'dark' | 'favicon' | 'email';
  const variants: { id: LogoVariant; label: string; help: string }[] = [
    { id: 'main', label: 'Primary logo', help: 'Used in the global header on light backgrounds.' },
    { id: 'dark', label: 'Dark-mode logo', help: 'Optional. Shown when the app is in dark mode.' },
    { id: 'favicon', label: 'Favicon', help: 'Browser tab icon. PNG or ICO, 32x32 or 64x64.' },
    { id: 'email', label: 'Email logo', help: 'Used in transactional email headers.' },
  ];

  let brandName = $state('');
  let primaryColor = $state('');
  let accentColor = $state('');
  let darkColor = $state('');

  // Per-variant: whether the server has a logo + a cache-buster bumped on upload
  let logoState = $state<Record<LogoVariant, { present: boolean; cacheBust: number }>>({
    main: { present: false, cacheBust: 0 },
    dark: { present: false, cacheBust: 0 },
    favicon: { present: false, cacheBust: 0 },
    email: { present: false, cacheBust: 0 },
  });

  let saving = $state(false);
  let loading = $state(true);

  function applyFromServer(data: any) {
    brandName = data.brand_name ?? '';
    primaryColor = data.primary_color ?? '';
    accentColor = data.accent_color ?? '';
    darkColor = data.dark_color ?? '';
    logoState = {
      main: { present: !!data.has_logo_main, cacheBust: logoState.main.cacheBust + 1 },
      dark: { present: !!data.has_logo_dark, cacheBust: logoState.dark.cacheBust + 1 },
      favicon: { present: !!data.has_favicon, cacheBust: logoState.favicon.cacheBust + 1 },
      email: { present: !!data.has_email_logo, cacheBust: logoState.email.cacheBust + 1 },
    };
    // Refresh the global branding store so the live UI updates immediately.
    branding.load();
  }

  async function load() {
    loading = true;
    try {
      const res = await xfetch('/api/branding');
      const result = await res.json();
      applyFromServer(result.data);
    } catch {
      notifications.danger('Could not load branding settings');
    } finally {
      loading = false;
    }
  }

  async function saveMeta() {
    saving = true;
    try {
      const res = await xfetch('/api/admin/branding', {
        method: 'PUT',
        body: {
          brand_name: brandName,
          primary_color: primaryColor,
          accent_color: accentColor,
          dark_color: darkColor,
        },
      });
      const result = await res.json();
      applyFromServer(result.data);
      notifications.success('Branding saved');
    } catch (err: any) {
      if (Array.isArray(err)) {
        const result = await err[1].json();
        notifications.danger(result.error || 'Could not save branding');
      } else {
        notifications.danger('Could not save branding');
      }
    } finally {
      saving = false;
    }
  }

  async function uploadLogo(variant: LogoVariant, file: File) {
    const form = new FormData();
    form.append('file', file);
    try {
      const res = await fetch(`${PathPrefix}/api/admin/branding/logo?variant=${variant}`, {
        method: 'POST',
        body: form,
        credentials: 'same-origin',
      });
      if (!res.ok) {
        const result = await res.json().catch(() => ({}));
        notifications.danger(result.error || 'Upload failed');
        return;
      }
      const result = await res.json();
      applyFromServer(result.data);
      notifications.success(`${variant} logo uploaded`);
    } catch {
      notifications.danger('Upload failed');
    }
  }

  async function clearLogo(variant: LogoVariant) {
    try {
      const res = await xfetch(`/api/admin/branding/logo?variant=${variant}`, { method: 'DELETE' });
      const result = await res.json();
      applyFromServer(result.data);
      notifications.success(`${variant} logo cleared`);
    } catch {
      notifications.danger('Could not clear logo');
    }
  }

  async function resetAll() {
    if (!confirm('Reset all branding to defaults? This clears your logos and colors.')) return;
    try {
      const res = await xfetch('/api/admin/branding', { method: 'DELETE' });
      const result = await res.json();
      applyFromServer(result.data);
      notifications.success('Branding reset to defaults');
    } catch {
      notifications.danger('Could not reset branding');
    }
  }

  function logoUrl(variant: LogoVariant) {
    return `${PathPrefix}/api/branding/logo?variant=${variant}&v=${logoState[variant].cacheBust}`;
  }

  function onFilePicked(variant: LogoVariant) {
    return (e: Event) => {
      const target = e.target as HTMLInputElement;
      const file = target.files?.[0];
      if (file) uploadLogo(variant, file);
      target.value = '';
    };
  }

  onMount(() => {
    if (!$user.id) {
      router.route(appRoutes.login);
      return;
    }
    if (!validateUserIsAdmin($user)) {
      router.route(appRoutes.landing);
      return;
    }
    load();
  });
</script>

<AdminPageLayout activePage="Branding">
  <div class="space-y-6">
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md p-6">
      <div class="flex items-start justify-between mb-4">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Branding</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            Replace the default Thunderdome branding with your workspace's identity. Changes apply immediately for new
            page loads.
          </p>
        </div>
        <HollowButton color="red" onClick={resetAll}>Reset to defaults</HollowButton>
      </div>

      {#if loading}
        <p class="text-gray-500 dark:text-gray-400">Loading…</p>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label class="block text-sm font-bold mb-2 dark:text-gray-300" for="brand_name">Brand name</label>
            <TextInput id="brand_name" name="brand_name" bind:value={brandName} placeholder="Agile Central" />
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Shown alongside the logo where the app uses a wordmark.
            </p>
          </div>
          <div></div>
          <div>
            <label class="block text-sm font-bold mb-2 dark:text-gray-300" for="primary_color">Primary color</label>
            <div class="flex items-center gap-2">
              <input type="color" id="primary_color" bind:value={primaryColor} class="h-10 w-12 rounded" />
              <TextInput
                id="primary_color_hex"
                name="primary_color_hex"
                bind:value={primaryColor}
                placeholder="#ffdd57"
              />
            </div>
          </div>
          <div>
            <label class="block text-sm font-bold mb-2 dark:text-gray-300" for="accent_color">Accent color</label>
            <div class="flex items-center gap-2">
              <input type="color" id="accent_color" bind:value={accentColor} class="h-10 w-12 rounded" />
              <TextInput
                id="accent_color_hex"
                name="accent_color_hex"
                bind:value={accentColor}
                placeholder="#6366f1"
              />
            </div>
          </div>
          <div>
            <label class="block text-sm font-bold mb-2 dark:text-gray-300" for="dark_color">Dark color</label>
            <div class="flex items-center gap-2">
              <input type="color" id="dark_color" bind:value={darkColor} class="h-10 w-12 rounded" />
              <TextInput id="dark_color_hex" name="dark_color_hex" bind:value={darkColor} placeholder="#111827" />
            </div>
          </div>
        </div>

        <div class="flex justify-end mt-6">
          <SolidButton onClick={saveMeta} disabled={saving}>
            {saving ? 'Saving…' : 'Save brand & colors'}
          </SolidButton>
        </div>
      {/if}
    </div>

    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md p-6">
      <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Logos</h2>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-6">SVG or PNG, max 512KB each. Uploads replace any existing logo for that slot.</p>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {#each variants as v}
          <div class="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
            <div class="flex items-start justify-between gap-3 mb-3">
              <div>
                <div class="font-bold text-gray-900 dark:text-white">{v.label}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">{v.help}</div>
              </div>
            </div>

            <div class="h-24 flex items-center justify-center bg-gray-100 dark:bg-gray-900 rounded mb-3">
              {#if logoState[v.id].present}
                <img src={logoUrl(v.id)} alt="{v.label} preview" class="max-h-20 max-w-full object-contain" />
              {:else}
                <span class="text-xs text-gray-400 italic">No logo uploaded</span>
              {/if}
            </div>

            <div class="flex items-center gap-2">
              <label
                class="inline-block px-3 py-1.5 text-sm font-medium border border-indigo-500 text-indigo-600 dark:text-indigo-300 dark:border-indigo-400 rounded cursor-pointer hover:bg-indigo-50 dark:hover:bg-indigo-900/30"
              >
                Upload
                <input type="file" accept="image/*" onchange={onFilePicked(v.id)} class="hidden" />
              </label>
              {#if logoState[v.id].present}
                <HollowButton color="red" onClick={() => clearLogo(v.id)}>Clear</HollowButton>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    </div>
  </div>
</AdminPageLayout>
