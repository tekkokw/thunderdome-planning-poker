<script lang="ts">
  import SolidButton from '../global/SolidButton.svelte';
  import SelectInput from '../forms/SelectInput.svelte';
  import { AppConfig, appRoutes } from '../../config';
  import { user } from '../../stores';
  import LL from '../../i18n/i18n-svelte';
  import { createEventDispatcher, onMount } from 'svelte';
  import FeatureSubscribeBanner from '../global/FeatureSubscribeBanner.svelte';

  import type { NotificationService } from '../../types/notifications';
  import type { ApiClient } from '../../types/apiclient';
  import type { SessionUser } from '../../types/user';
  import { SearchIcon } from '@lucide/svelte';

  const dispatch = createEventDispatcher();

  interface Props {
    handleImport?: any;
    notifications: NotificationService;
    xfetch: ApiClient;
  }

  let { handleImport = (story: any) => {}, notifications, xfetch }: Props = $props();

  // Linear's numeric priorities (0=No priority, 1=Urgent ... 4=Low)
  // Map them to Thunderdome's 1..6 scale (1 highest urgency, 6 lowest).
  const linearPriorityMap: Record<number, number> = {
    1: 1, // Urgent  -> Blocker
    2: 3, // High    -> High
    3: 4, // Medium  -> Medium
    4: 5, // Low     -> Low
    0: 99, // No priority
  };

  // Map Linear label/state to a Thunderdome plan type. We default to Story.
  function mapPlanType(state?: { name?: string; type?: string }): string {
    const name = state?.name?.toLowerCase() ?? '';
    if (name.includes('bug')) return $LL.planTypeBug();
    if (name.includes('epic')) return $LL.planTypeEpic();
    if (name.includes('spike')) return $LL.planTypeSpike();
    if (name.includes('task')) return $LL.planTypeTask();
    if (name.includes('subtask')) return $LL.planTypeSubtask();
    return $LL.planTypeStory();
  }

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
  interface LinearIssue {
    id: string;
    identifier: string;
    title: string;
    description?: string;
    url: string;
    priority: number;
    estimate?: number | null;
    state?: { id: string; name: string; type: string } | null;
    team?: { id: string; key: string; name: string } | null;
    labels?: Array<{ id: string; name: string }>;
  }

  let linearInstances = $state<LinearInstance[]>([]);
  let linearTeams = $state<LinearTeam[]>([]);
  let linearIssues = $state<LinearIssue[]>([]);
  let selectedInstance: string = $state('');
  let selectedTeamKey: string = $state('');
  let searchQuery: string = $state('');
  let searchError: string = $state('');
  let importedIssueIds = $state<string[]>([]);
  let searchCompleted: boolean = $state(false);
  let loadingTeams: boolean = $state(false);
  let searching: boolean = $state(false);

  function handleAuthOrError(error: any, fallbackMsg: string, setMsg: (s: string) => void) {
    if (Array.isArray(error)) {
      error[1].json().then(function (result: any) {
        if (result.error === 'REQUIRES_SUBSCRIBED_USER') {
          user.update({
            id: $user.id,
            name: $user.name,
            email: $user.email,
            rank: $user.rank,
            avatar: $user.avatar,
            verified: $user.verified,
            notificationsEnabled: $user.notificationsEnabled,
            locale: $user.locale,
            theme: $user.theme,
            subscribed: false,
          } as SessionUser);
          setMsg('subscription(s) expired');
        } else {
          setMsg(result.error || fallbackMsg);
        }
      });
    } else {
      setMsg(fallbackMsg);
    }
  }

  function getLinearInstances() {
    xfetch(`/api/users/${$user.id}/linear-instances`)
      .then(res => res.json())
      .then(function (result) {
        linearInstances = result.data;
      })
      .catch(function (error) {
        handleAuthOrError(error, 'error getting Linear workspaces', notifications.danger);
      });
  }

  function loadTeams() {
    const instance = linearInstances[Number(selectedInstance)];
    if (!instance) return;

    loadingTeams = true;
    linearTeams = [];
    linearIssues = [];
    importedIssueIds = [];
    searchCompleted = false;

    xfetch(`/api/users/${$user.id}/linear-instances/${instance.id}/teams`)
      .then(res => res.json())
      .then(function (result) {
        linearTeams = result.data.teams || [];
        loadingTeams = false;
      })
      .catch(function (error) {
        loadingTeams = false;
        handleAuthOrError(error, 'error loading Linear teams', notifications.danger);
      });
  }

  function handleSearch(event: Event) {
    event.preventDefault();

    const instance = linearInstances[Number(selectedInstance)];
    if (!instance) {
      notifications.danger('Select a Linear workspace');
      return;
    }

    linearIssues = [];
    importedIssueIds = [];
    searchCompleted = false;
    searching = true;
    searchError = '';

    xfetch(`/api/users/${$user.id}/linear-instances/${instance.id}/issue-search`, {
      body: {
        query: searchQuery,
        teamKey: selectedTeamKey,
        first: 50,
      },
    })
      .then(res => res.json())
      .then(function (result) {
        linearIssues = result.data?.issues || [];
        searchCompleted = true;
        searching = false;
      })
      .catch(function (error) {
        searching = false;
        searchCompleted = true;
        handleAuthOrError(error, 'Linear search error', m => (searchError = m));
      });
  }

  function buildStoryFromIssue(issue: LinearIssue) {
    return {
      name: issue.title,
      type: mapPlanType(issue.state ?? undefined),
      referenceId: issue.identifier,
      link: issue.url,
      description: issue.description || '',
      priority: linearPriorityMap[issue.priority] ?? 99,
    };
  }

  function importIssue(idx: number) {
    return function () {
      const issue = linearIssues[idx];
      handleImport(buildStoryFromIssue(issue));
      importedIssueIds = [...importedIssueIds, issue.id];
    };
  }

  function importAllIssues() {
    const remaining = linearIssues.filter(i => !importedIssueIds.includes(i.id));
    remaining.forEach(issue => {
      handleImport(buildStoryFromIssue(issue));
      importedIssueIds = [...importedIssueIds, issue.id];
    });
  }

  onMount(() => {
    if ((AppConfig.SubscriptionsEnabled && $user.subscribed) || !AppConfig.SubscriptionsEnabled) {
      getLinearInstances();
    }
  });
</script>

{#if AppConfig.SubscriptionsEnabled && !$user.subscribed}
  <FeatureSubscribeBanner salesPitch="Import your stories for Poker Planning from Linear." />
{:else if !AppConfig.SubscriptionsEnabled || (AppConfig.SubscriptionsEnabled && $user.subscribed)}
  {#if linearInstances.length === 0}
    <p class="info-banner">
      Visit your <a href={appRoutes.profile} class="info-banner-link" target="_blank">profile page</a> to connect a
      Linear workspace.
    </p>
  {:else}
    <div class="select-wrapper">
      <SelectInput
        id="linearinstance"
        bind:value={selectedInstance}
        onchange={() => {
          dispatch('instance_selected');
          loadTeams();
        }}
      >
        <option value="" disabled>Select Linear workspace to import from</option>
        {#each linearInstances as li, idx}
          <option value={idx}>{li.label}</option>
        {/each}
      </SelectInput>
    </div>

    {#if selectedInstance !== ''}
      <form onsubmit={handleSearch} class="search-form">
        <div class="filter-row">
          <SelectInput id="linear-team" bind:value={selectedTeamKey}>
            <option value="">All teams</option>
            {#each linearTeams as t}
              <option value={t.key}>{t.key} — {t.name}</option>
            {/each}
          </SelectInput>
        </div>

        <label for="linear-search" class="search-label">Search</label>
        <div class="search-container">
          <div class="search-icon-wrapper">
            <SearchIcon class="w-4 h-4 text-gray-500 dark:text-gray-400" />
          </div>
          <input
            type="search"
            id="linear-search"
            class="search-input"
            placeholder="Search by title (leave blank for recent issues)..."
            bind:value={searchQuery}
          />
          <button type="submit" class="search-button" disabled={searching || loadingTeams}>
            {searching ? 'Searching…' : 'Search'}
          </button>
        </div>
      </form>

      <div class="stories-wrapper">
        {#if searchError !== ''}
          <div class="error-message">
            {searchError}
          </div>
        {/if}
        {#if searchCompleted && linearIssues.length === 0 && searchError === ''}
          <p class="no-stories-message">No issues found.</p>
        {/if}
        {#if linearIssues.length > 0 && importedIssueIds.length === linearIssues.length}
          <p class="all-imported-message">All issues have been imported!</p>
        {/if}
        {#if linearIssues.length > 0 && importedIssueIds.length < linearIssues.length}
          <div class="search-result-header">
            <span class="text-lg font-semibold">
              Search Results ({linearIssues.length - importedIssueIds.length})
            </span>
            <SolidButton onClick={importAllIssues}>Import All</SolidButton>
          </div>
        {/if}
        {#each linearIssues as issue, idx}
          <div
            class="story-item"
            class:hidden={importedIssueIds.includes(issue.id)}
            aria-hidden={importedIssueIds.includes(issue.id)}
          >
            <div class="issue-meta">
              <span class="identifier">{issue.identifier}</span>
              <span class="title">{issue.title}</span>
              {#if issue.state}
                <span class="state-badge">{issue.state.name}</span>
              {/if}
            </div>
            <div>
              <SolidButton onClick={importIssue(idx)}>Import</SolidButton>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
{/if}

<style lang="postcss">
  .info-banner {
    @apply bg-yellow-thunder text-gray-900 p-4 rounded font-bold;
  }

  :root.dark .info-banner {
    @apply bg-yellow-600 text-white;
  }

  .info-banner-link {
    @apply underline;
  }

  .select-wrapper {
    @apply mb-4;
  }

  .filter-row {
    @apply mb-3;
  }

  .search-form {
    @apply mb-4;
  }

  .search-label {
    @apply mb-2 text-sm font-medium text-gray-900 sr-only;
  }

  :root.dark .search-label {
    @apply text-white;
  }

  .search-container {
    @apply relative;
  }

  .search-icon-wrapper {
    @apply absolute inset-y-0 start-0 flex items-center ps-3 pointer-events-none;
  }

  :root.dark .search-icon-wrapper {
    @apply text-gray-400;
  }

  .search-input {
    @apply block w-full p-4 ps-10 text-sm text-gray-900 border border-gray-300 rounded-lg bg-gray-50;
  }

  :root.dark .search-input {
    @apply bg-gray-700 border-gray-600 placeholder-gray-400 text-white;
  }

  .search-input:focus {
    @apply ring-purple-500 border-purple-500;
  }

  .search-button {
    @apply text-white absolute end-2.5 bottom-2.5 bg-purple-600 font-medium rounded-lg text-sm px-4 py-2;
  }

  .search-button:hover {
    @apply bg-purple-700;
  }

  .search-button:disabled {
    @apply opacity-60 cursor-not-allowed;
  }

  .search-button:focus {
    @apply ring-4 outline-none ring-purple-300;
  }

  .search-result-header {
    @apply flex justify-between items-center mb-4 text-gray-700 w-full;
  }

  :root.dark .search-result-header {
    @apply text-gray-300;
  }

  .stories-wrapper {
    @apply flex flex-wrap;
  }

  .error-message {
    @apply p-4 bg-red-50 border-red-500 text-red-800 font-semibold rounded-lg w-full;
  }

  :root.dark .error-message {
    @apply bg-red-900 border-red-700 text-red-200;
  }

  .no-stories-message {
    @apply p-4 text-gray-700 text-center italic w-full rounded-lg bg-gray-200;
  }

  :root.dark .no-stories-message {
    @apply text-gray-200 bg-gray-700;
  }

  .all-imported-message {
    @apply p-4 text-green-700 text-center font-semibold bg-green-50 rounded-lg w-full;
  }

  :root.dark .all-imported-message {
    @apply text-green-200 bg-green-900;
  }

  .story-item {
    padding: 0.5rem;
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    background-color: theme('colors.gray.200');
    border-radius: 0.5rem;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    align-items: center;
    transition: all 200ms ease-out;
    margin-bottom: 0.5rem;
  }

  :root.dark .story-item {
    background-color: theme('colors.gray.700');
    color: white;
  }

  .issue-meta {
    @apply flex items-center gap-2 flex-wrap;
  }

  .identifier {
    @apply font-mono text-xs font-bold px-2 py-1 rounded;
    background-color: theme('colors.purple.100');
    color: theme('colors.purple.800');
  }

  :root.dark .identifier {
    background-color: theme('colors.purple.900');
    color: theme('colors.purple.200');
  }

  .title {
    @apply font-medium;
  }

  .state-badge {
    @apply text-xs px-2 py-1 rounded-full;
    background-color: theme('colors.gray.300');
    color: theme('colors.gray.700');
  }

  :root.dark .state-badge {
    background-color: theme('colors.gray.600');
    color: theme('colors.gray.200');
  }

  .story-item.hidden {
    pointer-events: none;
    max-height: 0;
    padding: 0;
    overflow: hidden;
    margin-bottom: 0;
    transition:
      padding 200ms ease-out,
      max-height 200ms ease-out,
      margin-bottom 200ms ease-out;
  }
</style>
