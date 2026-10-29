<script lang="ts">
 import {onMount} from 'svelte';
 import MediaShelf from './MediaShelf.svelte';
 import {collectionCards,type UIPage} from './ui-pages';
 import type {Entry,Work,Kind} from './types';import type {Card} from './experience';
 export let owner:string;export let id:string;export let revision=0;
 export let entries:Entry[];export let works:Work[];export let kinds:Kind[];
 export let open:(c:Card)=>void;export let playAlbum:(c:Card)=>void;export let navigate:(target:string,owner:string)=>void;
 export let titleChanged:(title:string)=>void=()=>{};
 let page:UIPage|null=null;let error='';let loading=true;let request:AbortController|null=null;let alive=true;
 async function load(owner:string,id:string,_revision:number){request?.abort();const current=new AbortController();request=current;if(page?.owner!==owner||page?.id!==id)page=null;error='';loading=!page;
  try{const r=await fetch('/api/v1/ui/pages/'+encodeURIComponent(owner)+'/'+encodeURIComponent(id),{signal:current.signal});const data=await r.json();if(!r.ok)throw Error(data.error||'Page indisponible.');if(alive&&request===current){if(JSON.stringify(page)!==JSON.stringify(data))page=data;titleChanged(data.label)}}catch(e){if(alive&&request===current&&!current.signal.aborted){page=null;error=e instanceof Error?e.message:'Page indisponible.'}}finally{if(alive&&request===current)loading=false}
 }
 $: void load(owner,id,revision);
 onMount(()=>()=>{alive=false;request?.abort()});
</script>
<section class="extension-page" aria-busy={loading}>
 {#if loading}<p role="status">Chargement de la page…</p>{/if}
 {#if error}<p class="extension-warning" role="alert">{error}</p>{/if}
 {#if page}<div class="page-heading"><span class="eyebrow">VOTRE ESPACE</span><h1>{page.label}</h1>{#if page.description}<p>{page.description}</p>{/if}</div>
 {#each page.blocks??[] as block}
  {#if block.type==='text'}<section class="extension-preview">{#if block.title}<h2>{block.title}</h2>{/if}<p>{block.text}</p></section>
  {:else if block.type==='facts'}<section class="extension-preview">{#if block.title}<h2>{block.title}</h2>{/if}<dl>{#each block.facts??[] as fact}<dt>{fact.label}</dt><dd>{fact.value}</dd>{/each}</dl></section>
  {:else if block.type==='link'}<div class="extension-actions"><button class="secondary" onclick={()=>navigate(block.target??'',owner)}>{block.title} →</button>{#if block.text}<p>{block.text}</p>{/if}</div>
  {:else if block.type==='collection'}{@const cards=collectionCards(block.view??'home',entries,works,kinds)}<MediaShelf title={block.title??'Dans votre collection'} {cards} {open} {playAlbum}/>{#if !cards.length}<p class="folder-help">Aucun contenu à afficher dans cette sélection.</p>{/if}
  {/if}
 {/each}{/if}
</section>
