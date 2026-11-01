<script lang="ts">
 import {onDestroy} from 'svelte';
 import Icon from './Icon.svelte';
 import SearchResultContent from './SearchResultContent.svelte';
 import {searchPresentation} from './search-presentation';
 import {localSearch,mergeSearch} from './catalogue-search';
 import {searchJobs,searchPlaceholder,searchScopeLabel,searchUnavailable,interleave} from './remote-discovery';
 import type {Card} from './experience';
 import type {Entry,Kind,Provider,Work} from './types';
 export let works:Work[]=[];export let entries:Entry[];export let providers:Provider[];export let kinds:Kind[];export let context:string;export let token:string;export let open:(card:Card)=>void;
 let query='';let expanded=false;let searching=false;let remote:Work[]=[];let warning='';let timer:ReturnType<typeof setTimeout>;let abort:AbortController|undefined;let serial=0;let activeIndex=-1;let input:HTMLInputElement;
 let scope:Kind[]=[];let results:Card[]=[];
 $: scope=kinds.includes(context as Kind)?[context as Kind]:kinds;
 $: label=searchPlaceholder(providers,scope);
 $: local=localSearch(entries,scope,query,works);
 $: results=[...local,...mergeSearch(local,remote)];
 $: providerStamp=JSON.stringify(providers.map(p=>[p.id,p.enabled,p.configured,p.discovery,p.catalogue]));
 let previousProviders='';
 $: if(providerStamp!==previousProviders){previousProviders=providerStamp;close();remote=[];warning='';}
 let previousContext=context;
 $: if(context!==previousContext){previousContext=context;close();query='';remote=[];}
 function close(){expanded=false;activeIndex=-1;serial++;abort?.abort();clearTimeout(timer);searching=false;}
 function update(){expanded=true;activeIndex=-1;remote=[];warning='';clearTimeout(timer);abort?.abort();const id=++serial;searching=false;if(query.trim().length<2)return;timer=setTimeout(()=>void search(id),400);}
 async function search(id:number){abort=new AbortController();searching=true;const term=query.trim();const jobs=searchJobs(providers,scope);try{
  const batches=await Promise.all(jobs.map(async job=>{try{const r=await fetch('/api/v1/discovery/browse',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({provider:job.provider,kind:job.kind,query:term,page:1}),signal:abort?.signal});const data=await r.json();return r.ok?{items:data.items as Work[],error:''}:{items:[],error:job.name+' : '+(data.error||'Recherche indisponible.')};}catch{return {items:[],error:job.name+' : recherche momentanément indisponible.'};}}));
  if(id===serial){remote=interleave(batches.map(b=>b.items),72);warning=[...new Set([searchUnavailable(providers,scope),...batches.map(b=>b.error)].filter(Boolean))].join(' ');}
 }finally{if(id===serial)searching=false;}}
 function choose(card:Card){close();query='';remote=[];open(card);}
 function key(event:KeyboardEvent){if(event.key==='Escape'){close();input.focus();}else if(event.key==='ArrowDown'||event.key==='ArrowUp'){event.preventDefault();expanded=true;activeIndex=Math.max(0,Math.min(results.length-1,activeIndex+(event.key==='ArrowDown'?1:-1)));}else if(event.key==='Enter'&&results[activeIndex]){event.preventDefault();choose(results[activeIndex]);}}
 onDestroy(close);
</script>
<svelte:window onclick={(e)=>{if(!(e.target as HTMLElement).closest('.global-search'))close();}}/>
<div class="global-search"><Icon name="search" size={20}/><input bind:this={input} bind:value={query} oninput={update} onfocus={()=>expanded=!!query} onkeydown={key} role="combobox" aria-label={label} aria-expanded={expanded} aria-controls="search-results" aria-autocomplete="list" aria-activedescendant={activeIndex>=0?`search-item-${activeIndex}`:undefined} placeholder={label}/>{#if query}<button class="icon-button" aria-label="Effacer la recherche" onclick={()=>{query='';remote=[];close();input.focus();}}><Icon name="close" size={16}/></button>{/if}
 {#if expanded&&query.trim()}<div class="search-dropdown"><div class="search-caption">{searchScopeLabel(providers,scope)} · collection et catalogue en ligne</div><div id="search-results" role="listbox" aria-label="Résultats de recherche">{#each results as card,index(card.key)}<button id={`search-item-${index}`} role="option" aria-selected={index===activeIndex} class:highlighted={index===activeIndex} class="search-result" onclick={()=>choose(card)}><SearchResultContent title={card.title} subtitle={card.subtitle} image={card.image} presentation={searchPresentation(card,works)} icon={card.kind}/><span class="search-availability" class:local={card.local}>{card.local?'● Chez vous':'○ À découvrir'}</span></button>{/each}</div>{#if searching}<p role="status">Recherche en ligne…</p>{:else if !results.length}<p role="status">{query.trim().length<2?'Saisissez au moins deux caractères.':'Aucun résultat. Essayez le titre original ou un artiste.'}</p>{/if}{#if warning}<p class="folder-help">{warning}</p>{/if}<div class="search-footnote">Entrée pour ouvrir · Échap pour fermer</div></div>{/if}
</div>
