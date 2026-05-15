<script lang="ts">
  import { onMount } from 'svelte';

  import AdminPageLayout from '../../components/admin/AdminPageLayout.svelte';
  import SolidButton from '../../components/global/SolidButton.svelte';
  import HollowButton from '../../components/global/HollowButton.svelte';
  import TextInput from '../../components/forms/TextInput.svelte';
  import Modal from '../../components/global/Modal.svelte';
  import { user } from '../../stores';
  import { appRoutes, AppConfig } from '../../config';
  import { validateUserIsAdmin } from '../../validationUtils';

  import type { NotificationService } from '../../types/notifications';
  import type { ApiClient } from '../../types/apiclient';
  import { Bot, Copy } from '@lucide/svelte';

  interface Props {
    xfetch: ApiClient;
    router: any;
    notifications: NotificationService;
  }

  let { xfetch, router, notifications }: Props = $props();

  interface APIKey {
    id: string;
    name: string;
    prefix: string;
    active: boolean;
    createdDate: string;
  }
  interface ServiceAccount {
    id: string;
    name: string;
    email: string;
    createdDate: string;
    apiKeys: APIKey[];
  }

  let accounts = $state<ServiceAccount[]>([]);
  let loading = $state(true);

  let showCreate = $state(false);
  let newName = $state('');
  let newEmail = $state('');
  let creating = $state(false);

  // One-time plaintext key reveal
  let revealedKey = $state('');
  let revealedFor = $state('');

  async function load() {
    loading = true;
    try {
      const res = await xfetch('/api/admin/service-accounts');
      const result = await res.json();
      accounts = result.data || [];
    } catch {
      notifications.danger('Could not load service accounts');
    } finally {
      loading = false;
    }
  }

  async function createAccount() {
    if (newName.trim() === '' || newEmail.trim() === '') {
      notifications.danger('Name and email are required');
      return;
    }
    creating = true;
    try {
      const res = await xfetch('/api/admin/service-accounts', {
        method: 'POST',
        body: { name: newName.trim(), email: newEmail.trim() },
      });
      const result = await res.json();
      revealedKey = result.data.apiKey;
      revealedFor = result.data.name;
      showCreate = false;
      newName = '';
      newEmail = '';
      await load();
    } catch (err: any) {
      if (Array.isArray(err)) {
        const result = await err[1].json();
        notifications.danger(result.error || 'Could not create service account');
      } else {
        notifications.danger('Could not create service account');
      }
    } finally {
      creating = false;
    }
  }

  async function deleteAccount(id: string, name: string) {
    if (!confirm(`Delete service account "${name}"? Its API keys stop working immediately.`)) return;
    try {
      await xfetch(`/api/admin/service-accounts/${id}`, { method: 'DELETE' });
      notifications.success('Service account deleted');
      await load();
    } catch {
      notifications.danger('Could not delete service account');
    }
  }

  async function generateKey(id: string, name: string) {
    try {
      const res = await xfetch(`/api/admin/service-accounts/${id}/apikeys`, {
        method: 'POST',
        body: { name: 'key' },
      });
      const result = await res.json();
      revealedKey = result.data.apiKey ?? result.data.key;
      revealedFor = name;
      await load();
    } catch {
      notifications.danger('Could not generate API key');
    }
  }

  function copyKey() {
    navigator.clipboard?.writeText(revealedKey).then(
      () => notifications.success('Copied to clipboard'),
      () => notifications.danger('Copy failed — select and copy manually'),
    );
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

<AdminPageLayout activePage="Service Accounts">
  <div class="bg-white dark:bg-gray-800 rounded-lg shadow-md p-6">
    <div class="flex items-start justify-between mb-4">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <Bot class="w-6 h-6" /> Service Accounts
        </h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 max-w-2xl">
          Non-human users for agents and automation. They authenticate only with API keys and can be added to teams
          like any member — e.g. to post daily checkins or read cycle data via the API.
          {#if !AppConfig.ExternalAPIEnabled}
            <span class="block text-amber-600 dark:text-amber-400 mt-1">
              Heads up: the external API is disabled (CONFIG_ALLOW_EXTERNAL_API=false), so these keys won't authenticate
              until it's enabled.
            </span>
          {/if}
        </p>
      </div>
      <SolidButton onClick={() => (showCreate = true)}>New service account</SolidButton>
    </div>

    {#if loading}
      <p class="text-gray-500 dark:text-gray-400">Loading…</p>
    {:else if accounts.length === 0}
      <p class="text-gray-500 dark:text-gray-400 italic py-6 text-center">No service accounts yet.</p>
    {:else}
      <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead>
          <tr class="text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
            <th class="px-4 py-3">Name</th>
            <th class="px-4 py-3">Email</th>
            <th class="px-4 py-3">API keys</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 dark:divide-gray-700 dark:text-white">
          {#each accounts as a}
            <tr>
              <td class="px-4 py-3 font-medium flex items-center gap-2">
                <Bot class="w-4 h-4 text-purple-600 dark:text-purple-400" />
                {a.name}
              </td>
              <td class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{a.email}</td>
              <td class="px-4 py-3 text-sm">
                {#if a.apiKeys && a.apiKeys.length}
                  {#each a.apiKeys as k}
                    <span class="inline-block font-mono text-xs bg-gray-100 dark:bg-gray-700 rounded px-2 py-0.5 me-1">
                      {k.prefix}…
                    </span>
                  {/each}
                {:else}
                  <span class="text-gray-400 italic">none</span>
                {/if}
              </td>
              <td class="px-4 py-3 text-right whitespace-nowrap">
                <HollowButton onClick={() => generateKey(a.id, a.name)}>New key</HollowButton>
                <HollowButton color="red" onClick={() => deleteAccount(a.id, a.name)}>Delete</HollowButton>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</AdminPageLayout>

{#if showCreate}
  <Modal closeModal={() => (showCreate = false)} ariaLabel="Create service account">
    <h2 class="text-xl font-bold mb-4 dark:text-white">New service account</h2>
    <div class="mb-4">
      <label class="block text-sm font-bold mb-2 dark:text-gray-300" for="sa_name">Name</label>
      <TextInput id="sa_name" name="sa_name" bind:value={newName} placeholder="Standup Bot" />
    </div>
    <div class="mb-4">
      <label class="block text-sm font-bold mb-2 dark:text-gray-300" for="sa_email">Email (identifier)</label>
      <TextInput id="sa_email" name="sa_email" bind:value={newEmail} placeholder="standup-bot@workspace.internal" />
      <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
        Used only as a unique identifier — no email is ever sent to it.
      </p>
    </div>
    <div class="text-right">
      <SolidButton type="button" onClick={createAccount} disabled={creating}>
        {creating ? 'Creating…' : 'Create & generate key'}
      </SolidButton>
    </div>
  </Modal>
{/if}

{#if revealedKey}
  <Modal closeModal={() => (revealedKey = '')} ariaLabel="API key created">
    <h2 class="text-xl font-bold mb-2 dark:text-white">API key for {revealedFor}</h2>
    <p class="text-sm text-amber-600 dark:text-amber-400 mb-4">
      Copy this now — it is shown only once and cannot be retrieved again.
    </p>
    <div
      class="flex items-center gap-2 bg-gray-100 dark:bg-gray-900 border border-gray-300 dark:border-gray-700 rounded p-3 mb-4"
    >
      <code class="flex-1 break-all text-sm dark:text-gray-200">{revealedKey}</code>
      <button onclick={copyKey} class="text-gray-500 hover:text-gray-800 dark:hover:text-white" aria-label="Copy">
        <Copy class="w-4 h-4" />
      </button>
    </div>
    <div class="text-right">
      <SolidButton onClick={() => (revealedKey = '')}>Done</SolidButton>
    </div>
  </Modal>
{/if}
