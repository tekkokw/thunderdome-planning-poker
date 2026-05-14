<script lang="ts">
  import { onMount } from 'svelte';

  import HollowButton from '../global/HollowButton.svelte';
  import SolidButton from '../global/SolidButton.svelte';
  import SelectInput from '../forms/SelectInput.svelte';
  import { user } from '../../stores';

  import type { NotificationService } from '../../types/notifications';
  import type { ApiClient } from '../../types/apiclient';

  interface Props {
    xfetch: ApiClient;
    notifications: NotificationService;
    teamId: string;
    isEntityAdmin: boolean;
  }

  let { xfetch, notifications, teamId, isEntityAdmin }: Props = $props();

  interface LinearInstance {
    id: string;
    label: string;
    workspace_url_key: string;
  }
  interface LinearTeam {
    id: string;
    key: string;
    name: string;
  }
  interface TeamLinearLink {
    team_id: string;
    linear_instance_id: string;
    linear_team_id: string;
    linear_team_key: string;
    linear_team_name: string;
  }

  let link = $state<TeamLinearLink | null>(null);
  let loading = $state(true);
  let editing = $state(false);
  let saving = $state(false);

  // form state
  let myInstances = $state<LinearInstance[]>([]);
  let selectedInstanceId = $state('');
  let teamsByInstance = $state<LinearTeam[]>([]);
  let loadingTeams = $state(false);
  let selectedTeamId = $state('');

  async function loadLink() {
    loading = true;
    try {
      const res = await xfetch(`/api/teams/${teamId}/linear-link`);
      const result = await res.json();
      link = result.data;
    } catch (err: any) {
      // 404 is the expected "no link yet" case; surface anything else
      if (Array.isArray(err) && err[0]?.status === 404) {
        link = null;
      } else {
        link = null;
      }
    } finally {
      loading = false;
    }
  }

  async function loadInstances() {
    try {
      const res = await xfetch(`/api/users/${$user.id}/linear-instances`);
      const result = await res.json();
      myInstances = result.data || [];
    } catch {
      notifications.danger('Could not load your Linear workspaces');
    }
  }

  async function loadInstanceTeams(instanceId: string) {
    if (!instanceId) {
      teamsByInstance = [];
      return;
    }
    loadingTeams = true;
    try {
      const res = await xfetch(`/api/users/${$user.id}/linear-instances/${instanceId}/teams`);
      const result = await res.json();
      teamsByInstance = result.data?.teams || [];
    } catch {
      notifications.danger('Could not load Linear teams');
      teamsByInstance = [];
    } finally {
      loadingTeams = false;
    }
  }

  function openEdit() {
    editing = true;
    selectedInstanceId = link?.linear_instance_id ?? '';
    selectedTeamId = link?.linear_team_id ?? '';
    loadInstances().then(() => {
      if (selectedInstanceId) loadInstanceTeams(selectedInstanceId);
    });
  }

  function cancelEdit() {
    editing = false;
    selectedInstanceId = '';
    selectedTeamId = '';
    teamsByInstance = [];
  }

  async function save() {
    const team = teamsByInstance.find(t => t.id === selectedTeamId);
    if (!selectedInstanceId || !team) {
      notifications.danger('Pick a workspace and a team');
      return;
    }
    saving = true;
    try {
      const res = await xfetch(`/api/teams/${teamId}/linear-link`, {
        method: 'PUT',
        body: {
          linear_instance_id: selectedInstanceId,
          linear_team_id: team.id,
          linear_team_key: team.key,
          linear_team_name: team.name,
        },
      });
      const result = await res.json();
      link = result.data;
      editing = false;
      notifications.success('Linear team linked');
    } catch (err: any) {
      if (Array.isArray(err)) {
        const result = await err[1].json();
        notifications.danger(result.error || 'Could not save Linear link');
      } else {
        notifications.danger('Could not save Linear link');
      }
    } finally {
      saving = false;
    }
  }

  async function disconnect() {
    if (!confirm('Disconnect this team from Linear? Daily checkins will lose their cycle context.')) return;
    try {
      await xfetch(`/api/teams/${teamId}/linear-link`, { method: 'DELETE' });
      link = null;
      notifications.success('Linear link removed');
    } catch {
      notifications.danger('Could not remove Linear link');
    }
  }

  onMount(() => {
    loadLink();
  });
</script>

<div class="bg-white dark:bg-gray-800 shadow-lg rounded-lg p-4 md:p-6">
  <div class="flex items-center justify-between mb-4">
    <h2 class="text-2xl md:text-3xl font-semibold font-rajdhani uppercase dark:text-white">Linear Cycle Integration</h2>
    {#if isEntityAdmin && !editing}
      {#if link}
        <div class="flex gap-2">
          <HollowButton onClick={openEdit}>Change</HollowButton>
          <HollowButton color="red" onClick={disconnect}>Disconnect</HollowButton>
        </div>
      {:else}
        <SolidButton onClick={openEdit}>Connect Linear team</SolidButton>
      {/if}
    {/if}
  </div>

  {#if loading}
    <p class="text-gray-500 dark:text-gray-400">Loading…</p>
  {:else if editing}
    <div class="space-y-4">
      <div>
        <label for="linear-workspace" class="block text-sm font-bold mb-2 dark:text-gray-300">Workspace</label>
        <SelectInput
          id="linear-workspace"
          bind:value={selectedInstanceId}
          onchange={() => {
            selectedTeamId = '';
            loadInstanceTeams(selectedInstanceId);
          }}
        >
          <option value="" disabled>Select one of your connected workspaces</option>
          {#each myInstances as inst}
            <option value={inst.id}>{inst.label}</option>
          {/each}
        </SelectInput>
        {#if myInstances.length === 0}
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
            No Linear workspaces yet. Add one from your profile page first.
          </p>
        {/if}
      </div>

      {#if selectedInstanceId}
        <div>
          <label for="linear-team" class="block text-sm font-bold mb-2 dark:text-gray-300">Linear team</label>
          <SelectInput id="linear-team" bind:value={selectedTeamId} disabled={loadingTeams}>
            <option value="" disabled>
              {loadingTeams ? 'Loading teams…' : 'Select a Linear team'}
            </option>
            {#each teamsByInstance as t}
              <option value={t.id}>{t.key} — {t.name}</option>
            {/each}
          </SelectInput>
        </div>
      {/if}

      <div class="flex gap-2 justify-end">
        <HollowButton onClick={cancelEdit}>Cancel</HollowButton>
        <SolidButton onClick={save} disabled={saving || !selectedInstanceId || !selectedTeamId}>
          {saving ? 'Saving…' : 'Save'}
        </SolidButton>
      </div>
    </div>
  {:else if link}
    <p class="text-gray-700 dark:text-gray-300">
      Linked to Linear team
      <span class="font-mono font-bold text-purple-700 dark:text-purple-300">{link.linear_team_key}</span>
      {#if link.linear_team_name}
        <span class="text-gray-500 dark:text-gray-400">— {link.linear_team_name}</span>
      {/if}
    </p>
    <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
      Active-cycle context will appear on every team checkin.
    </p>
  {:else}
    <p class="text-gray-500 dark:text-gray-400">
      Connect this team to a Linear team to show active-cycle context on daily checkins.
    </p>
  {/if}
</div>
