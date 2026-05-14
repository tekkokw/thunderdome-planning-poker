<script lang="ts">
  import { onMount } from 'svelte';

  import Toggle from '../forms/Toggle.svelte';
  import { AppConfig, appRoutes } from '../../config';
  import type { NotificationService } from '../../types/notifications';
  import type { ApiClient } from '../../types/apiclient';
  import { Settings } from '@lucide/svelte';

  interface Props {
    xfetch: ApiClient;
    notifications: NotificationService;
  }

  let { xfetch, notifications }: Props = $props();

  let registrationOpen = $state(true);
  let loading = $state(true);
  let saving = $state(false);

  async function load() {
    try {
      const res = await xfetch('/api/admin/application-settings');
      const result = await res.json();
      registrationOpen = result.data.registration_open;
    } catch {
      notifications.danger('Could not load workspace settings');
    } finally {
      loading = false;
    }
  }

  async function toggleRegistration() {
    // bind:checked has already updated registrationOpen to the user's new
    // intended value. Persist that and fall back to the server's response.
    const desired = registrationOpen;
    saving = true;
    try {
      const res = await xfetch('/api/admin/application-settings', {
        method: 'PUT',
        body: { registration_open: desired },
      });
      const result = await res.json();
      registrationOpen = result.data.registration_open;
      notifications.success(registrationOpen ? 'Registration opened' : 'Registration closed');
    } catch {
      registrationOpen = !desired; // revert toggle visually
      notifications.danger('Could not update workspace settings');
    } finally {
      saving = false;
    }
  }

  onMount(load);
</script>

<div class="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-100 dark:border-gray-700 p-6 mb-6">
  <div class="flex items-center gap-3 mb-4">
    <div class="w-8 h-8 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-lg flex items-center justify-center">
      <Settings class="w-4 h-4 text-white" />
    </div>
    <div>
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Workspace settings</h2>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        Branding lives on the <a href={appRoutes.adminBranding ?? '/admin/branding'} class="underline">Branding</a> page.
      </p>
    </div>
  </div>

  {#if loading}
    <p class="text-sm text-gray-500 dark:text-gray-400">Loading…</p>
  {:else}
    <div class="flex items-start justify-between gap-4">
      <div>
        <div class="font-medium text-gray-900 dark:text-white">Open registration</div>
        <p class="text-sm text-gray-500 dark:text-gray-400 max-w-xl">
          When off, the public sign-up form is hidden and new accounts can only be added by an admin.
          {#if !AppConfig.AllowRegistration}
            <span class="block text-amber-600 dark:text-amber-400 mt-1">
              Note: registration is also disabled at the server level (CONFIG_ALLOW_REGISTRATION=false).
            </span>
          {/if}
        </p>
      </div>
      <Toggle
        bind:checked={registrationOpen}
        changeHandler={toggleRegistration}
        name="registration_open"
        id="registration_open"
      />
    </div>
  {/if}
</div>
