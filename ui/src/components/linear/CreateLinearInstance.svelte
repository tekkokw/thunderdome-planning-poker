<script lang="ts">
  import SolidButton from '../global/SolidButton.svelte';
  import Modal from '../global/Modal.svelte';
  import LL from '../../i18n/i18n-svelte';
  import { user } from '../../stores';
  import TextInput from '../forms/TextInput.svelte';

  import type { NotificationService } from '../../types/notifications';
  import { onMount } from 'svelte';
  import type { SessionUser } from '../../types/user';

  interface Props {
    handleCreate?: any;
    toggleClose?: any;
    xfetch?: any;
    notifications: NotificationService;
  }

  let { handleCreate = () => {}, toggleClose = () => {}, xfetch = () => {}, notifications }: Props = $props();

  let label = $state('');
  let access_token = $state('');
  let submitting = $state(false);

  function handleSubmit(event: Event) {
    event.preventDefault();

    if (label.trim() === '') {
      notifications.danger('Workspace label is required');
      return false;
    }
    if (access_token.trim() === '') {
      notifications.danger('Linear API key is required');
      return false;
    }

    submitting = true;
    const body = {
      label: label.trim(),
      access_token: access_token.trim(),
    };

    xfetch(`/api/users/${$user.id}/linear-instances`, { body })
      .then((res: Response) => res.json())
      .then(function () {
        handleCreate();
        toggleClose();
      })
      .catch(function (error: any) {
        submitting = false;
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
              notifications.danger('subscription(s) expired');
            } else {
              notifications.danger(result.error || 'failed to create Linear workspace');
            }
          });
        } else {
          notifications.danger('failed to create Linear workspace');
        }
      });
  }

  let focusInput: any;
  onMount(() => {
    focusInput?.focus();
  });
</script>

<Modal closeModal={toggleClose} ariaLabel={$LL.modalCreateLinearInstance()}>
  <form onsubmit={handleSubmit} name="createlinearinstance">
    <div class="mb-4">
      <label class="block dark:text-gray-400 font-bold mb-2" for="label">
        {$LL.linearWorkspaceLabel()}
      </label>
      <TextInput
        id="label"
        name="label"
        bind:value={label}
        bind:this={focusInput}
        placeholder={$LL.linearWorkspaceLabelPlaceholder()}
        required
      />
      <span class="text-sm dark:text-gray-400">
        {$LL.linearWorkspaceLabelHelp()}
      </span>
    </div>
    <div class="mb-4">
      <label class="block dark:text-gray-400 font-bold mb-2" for="access_token">
        {$LL.linearApiKeyLabel()}
      </label>
      <TextInput
        id="access_token"
        name="access_token"
        bind:value={access_token}
        placeholder="lin_api_..."
        required
      />
      <span class="text-sm dark:text-gray-400">
        {$LL.linearApiKeyHelp()}
        <a
          href="https://linear.app/settings/account/security"
          target="_blank"
          rel="noopener noreferrer"
          class="underline">linear.app/settings/account/security</a
        >.
      </span>
    </div>
    <div class="text-right">
      <div>
        <SolidButton type="submit" disabled={submitting}>
          {$LL.create()}
        </SolidButton>
      </div>
    </div>
  </form>
</Modal>
