<script lang="ts">
 import {onMount} from 'svelte';import {headerRequest,type HeaderView} from './detail-header';import type {Entry,Kind} from './types';
 export let entries:Entry[];export let workId='';export let kind:Kind;export let trackId='';export let token:string;export let revision='';export let fallbackTitle:string;
 let view:HeaderView|null=null;let warning='';let mounted=false;let previous='';let controller:AbortController|undefined;
 $: payload=headerRequest(kind,workId,entries.map(e=>e.id),trackId);
 $: signature=token+payload+revision;
 $: if(mounted&&previous!==signature){previous=signature;void load(payload);}
 onMount(()=>{mounted=true;const changed=()=>{previous='';};window.addEventListener('mosaune:detail-changed',changed);return()=>{mounted=false;controller?.abort();window.removeEventListener('mosaune:detail-changed',changed)}});
 async function load(body:string){controller?.abort();const request=new AbortController();controller=request;view=null;warning='';try{const response=await fetch('/api/v1/detail-header',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body,signal:request.signal});const data=await response.json();if(!response.ok)throw new Error(data.error||'En-tête indisponible.');if(!mounted||request.signal.aborted)return;view=data.view;warning=data.warning||'';}catch{if(mounted&&!request.signal.aborted)warning='L’en-tête enrichi est indisponible. Vos fichiers et métadonnées sont conservés.';}}
</script>
{#if view?.eyebrow}<span class="eyebrow">{view.eyebrow}</span>{/if}
<h1>{view?.title||fallbackTitle}</h1>
{#if view?.facts.length}<div class="content-facts">{#each view.facts as fact (fact.key)}<span>{fact.text}{#if fact.note} <small>{fact.note}</small>{/if}</span>{/each}</div>{/if}
<slot name="people"/>
{#if view?.tagline}<p class="tagline">{view.tagline}</p>{/if}
<slot name="commands"/>
{#if view?.footnote}<p class="folder-help">{view.footnote}</p>{/if}
{#if warning}<p class="notice" role="status">{warning}</p>{/if}
<slot/>
