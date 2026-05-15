<script lang="ts">
  import { Bot } from '@lucide/svelte';

  interface Props {
    label?: string;
    class?: string;
  }

  let { label = 'Bot', class: klass = '' }: Props = $props();
</script>

<span
  class="inline-flex items-center gap-1 rounded-full bg-purple-100 dark:bg-purple-900/40 px-2 py-0.5 text-xs font-semibold text-purple-700 dark:text-purple-300 {klass}"
  title="Service account"
>
  <Bot class="w-3 h-3" />
  {label}
</span>
