<script lang="ts">
  import Icon from './Icon.svelte';
  import type { Work } from './types';
  export let work: Work;
  let failed = false;
  let last = '';
  $: if (last !== work.id) { last = work.id; failed = false; }
</script>
<div class="work-poster">
  {#if work.poster && !failed}<img src={`/api/v1/discovery/works/${work.id}/poster`} alt="" loading="lazy" onerror={() => failed = true}/>
  {:else}<Icon name={work.kind} size={38}/><span>{work.metadata.title}</span>{/if}
</div>
