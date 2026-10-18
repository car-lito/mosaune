<script lang="ts">
 import Icon from './Icon.svelte';
 import {resolvePresentation,searchSecondary,type ResultPresentation} from './search-presentation';
 export let title:string;export let subtitle='';export let image:string|undefined=undefined;
 export let presentation:ResultPresentation;export let icon='search';
 const fallback:ResultPresentation={version:1,label:'Résultat',preview:{aspectRatio:1,shape:'rectangle',fit:'cover'}};
 $: display=resolvePresentation(presentation,fallback);
 $: preview=display.preview!;
 let failed='';
</script>
<span class="search-preview-slot" aria-hidden="true"><span class="search-preview" class:round-preview={preview.shape==='circle'} class:contain-preview={preview.fit==='contain'} style:width={`${Math.min(56,56*preview.aspectRatio)}px`} style:height={`${Math.min(56,56/preview.aspectRatio)}px`}>{#if image&&failed!==image}<img src={image} alt="" loading="lazy" onerror={()=>failed=image??''}/>{:else}<Icon name={icon}/>{/if}</span></span>
<span class="search-result-copy"><strong>{title}</strong><small>{searchSecondary(display,subtitle)}</small></span>
