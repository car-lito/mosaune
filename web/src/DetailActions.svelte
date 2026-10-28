<script lang="ts">
 import Icon from './Icon.svelte';import type {DetailAction} from './detail-experience';
 export let actions:DetailAction[];export let run:(a:DetailAction)=>void;export let allowed:(a:DetailAction)=>boolean;export let busy=false;export let watchlist=false;export let favorite=false;export let compact=false;
 const label=(a:DetailAction)=>a.operation==='watchlist'&&watchlist?'✓ Dans ma liste':a.operation==='favorite'&&favorite?'♥ Favori':a.label;
</script>
{#each actions as a,index (index)}
 <button class:primary={a.primary} class:secondary={!a.primary&&!compact} class:text-action={!a.primary&&compact} disabled={busy||!allowed(a)} aria-pressed={a.operation==='watchlist'?watchlist:a.operation==='favorite'?favorite:undefined} onclick={()=>run(a)}>{#if a.operation==='play'}<Icon name="play" size={17}/>{/if}{label(a)}</button>
{/each}
