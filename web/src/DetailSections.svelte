<script lang="ts">
 import {onMount} from 'svelte';import DetailSectionBlock from './DetailSectionBlock.svelte';import {regionBlocks,type SectionView} from './detail-sections';import type {Entry,Kind,Contributor} from './types';
 export let entries:Entry[];export let workId='';export let kind:Kind;export let token:string;export let revision='';export let person:(provider:string,p:Contributor)=>void;
 let view:SectionView|null=null;let warning='';let loading=false;let mounted=false;let previous='';let controller:AbortController|undefined;
 $: payload=JSON.stringify({kind,workId,ids:entries.map(e=>e.id)});
 $: signature=token+payload+revision;
 $: if(mounted&&previous!==signature){previous=signature;void load(payload);}
 $: main=regionBlocks(view,'main');$: aside=regionBlocks(view,'aside');$: full=regionBlocks(view,'full');
 onMount(()=>{mounted=true;const changed=()=>{previous='';};window.addEventListener('mosaune:detail-changed',changed);return()=>{mounted=false;controller?.abort();window.removeEventListener('mosaune:detail-changed',changed)}});
 async function load(body:string){controller?.abort();const request=new AbortController();controller=request;loading=true;view=null;warning='';try{const response=await fetch('/api/v1/detail-sections',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body,signal:request.signal});const data=await response.json();if(!response.ok)throw new Error(data.error||'Rubriques indisponibles.');if(!mounted||request.signal.aborted)return;view=data.view;warning=data.warning||'';}catch(e){if(mounted&&!request.signal.aborted)warning='Rubriques indisponibles. Vos fichiers et métadonnées sont conservés.';}finally{if(mounted&&!request.signal.aborted)loading=false}}
</script>
{#if loading}<p class="folder-help" role="status">Les rubriques se préparent…</p>{/if}
{#if warning}<p class="notice" role="status">{warning}</p>{/if}
{#if view}
 {#if main.length||aside.length}<div class="content-description"><div>{#each main as block (block.key)}<DetailSectionBlock {block} {view} {person}/>{/each}</div>{#if aside.length}<aside>{#each aside as block (block.key)}<DetailSectionBlock {block} {view} {person}/>{/each}</aside>{/if}</div>{/if}
 {#each full as block (block.key)}<DetailSectionBlock {block} {view} {person}/>{/each}
{/if}
