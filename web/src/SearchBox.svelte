<script lang="ts">
 import {onDestroy} from 'svelte';
 import Icon from './Icon.svelte';
 import {groupMedia} from './collection';
 import {groupCard,workCard,type Card} from './experience';
 import type {Entry,Kind,Provider,Work} from './types';
 export let entries:Entry[];export let providers:Provider[];export let kinds:Kind[];export let context:string;export let token:string;export let open:(card:Card)=>void;
 let query='';let expanded=false;let searching=false;let remote:Work[]=[];let warning='';let timer:ReturnType<typeof setTimeout>;let abort:AbortController|undefined;let serial=0;let activeIndex=-1;let input:HTMLInputElement;
 let scope:Kind[]=[];let results:Card[]=[];
 $: scope=kinds.includes(context as Kind)?[context as Kind]:kinds;
 $: label=scope.length===1?(scope[0]==='music'?'Rechercher un album, un artiste, un morceau…':scope[0]==='series'?'Rechercher une série…':'Rechercher un film…'):'Rechercher dans Mosaune…';
 $: groupResults=query.trim()?groupMedia(entries.filter(e=>scope.includes(e.kind))).filter(g=>[g.title,g.subtitle,...g.entries.map(e=>e.metadata?.title||e.name)].join(' ').toLocaleLowerCase('fr').includes(query.toLocaleLowerCase('fr').trim())).slice(0,8).map(groupCard):[];
 $: trackResults=query.trim()&&scope.includes('music')?entries.filter(e=>e.kind==='music'&&(e.metadata?.title||e.name).toLocaleLowerCase('fr').includes(query.toLocaleLowerCase('fr').trim())).slice(0,4).map<Card>(e=>({key:'track:'+e.id,title:e.metadata?.title||e.name,subtitle:`Morceau · ${e.metadata?.artist||''}`,kind:e.kind,local:true,entry:e,image:e.metadata?.imageId?`/api/v1/artwork/${e.metadata.imageId}`:undefined})):[];
 $: local=[...groupResults,...trackResults];
 $: results=[...local,...remote.filter(w=>!local.some(c=>c.group?.entries.some(e=>e.workId===w.id||(e.metadata?.provider===w.metadata.provider&&e.metadata?.providerId===w.metadata.providerId&&e.kind===w.kind)))).slice(0,12).map(workCard)];
 let previousContext=context;
 $: if(context!==previousContext){previousContext=context;close();query='';remote=[];}
 function close(){expanded=false;activeIndex=-1;serial++;abort?.abort();clearTimeout(timer);searching=false;}
 function update(){expanded=true;activeIndex=-1;remote=[];warning='';clearTimeout(timer);abort?.abort();const id=++serial;searching=false;if(query.trim().length<2)return;timer=setTimeout(()=>void search(id),400);}
 async function search(id:number){abort=new AbortController();searching=true;const term=query.trim();const sources=providers.filter(p=>p.discovery&&p.configured);try{
  const batches=await Promise.all(scope.map(async kind=>{const p=sources.find(p=>p.kinds.includes(kind));if(!p){if(kind!=='music')return {items:[],error:'Configurez TMDB dans Réglages pour les résultats en ligne.'};return {items:[],error:''};}try{const r=await fetch('/api/v1/discovery/browse',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({provider:p.id,kind,query:term,page:1}),signal:abort?.signal});const data=await r.json();return r.ok?{items:data.items,error:''}:{items:[],error:data.error};}catch{return {items:[],error:'La recherche en ligne est momentanément indisponible.'};}}));
  if(id===serial){remote=batches.flatMap(b=>b.items.slice(0,scope.length===1?12:4));warning=[...new Set(batches.map(b=>b.error).filter(Boolean))].join(' ');}
 }finally{if(id===serial)searching=false;}}
 function choose(card:Card){close();query='';remote=[];open(card);}
 function key(event:KeyboardEvent){if(event.key==='Escape'){close();input.focus();}else if(event.key==='ArrowDown'||event.key==='ArrowUp'){event.preventDefault();expanded=true;activeIndex=Math.max(0,Math.min(results.length-1,activeIndex+(event.key==='ArrowDown'?1:-1)));}else if(event.key==='Enter'&&results[activeIndex]){event.preventDefault();choose(results[activeIndex]);}}
 onDestroy(close);
</script>
<svelte:window onclick={(e)=>{if(!(e.target as HTMLElement).closest('.global-search'))close();}}/>
<div class="global-search"><Icon name="search" size={20}/><input bind:this={input} bind:value={query} oninput={update} onfocus={()=>expanded=!!query} onkeydown={key} role="combobox" aria-label={label} aria-expanded={expanded} aria-controls="search-results" aria-autocomplete="list" aria-activedescendant={activeIndex>=0?`search-item-${activeIndex}`:undefined} placeholder={label}/>{#if query}<button class="icon-button" aria-label="Effacer la recherche" onclick={()=>{query='';remote=[];close();input.focus();}}><Icon name="close" size={16}/></button>{/if}
 {#if expanded&&query.trim()}<div class="search-dropdown"><div class="search-caption">{scope.length===1?(scope[0]==='music'?'Musique':scope[0]==='films'?'Films':'Séries'):'Tous les univers'} · collection et catalogue en ligne</div><div id="search-results" role="listbox" aria-label="Résultats de recherche">{#each results as card,index(card.key)}<button id={`search-item-${index}`} role="option" aria-selected={index===activeIndex} class:highlighted={index===activeIndex} class="search-result" onclick={()=>choose(card)}>{#if card.image}<img src={card.image} alt="" onerror={e=>((e.currentTarget as HTMLImageElement).hidden=true)}/>{:else}<span class="search-icon"><Icon name={card.kind}/></span>{/if}<span><strong>{card.title}</strong><small>{card.subtitle}</small></span><span class="search-availability" class:local={card.local}>{card.local?'● Chez vous':'○ À découvrir'}</span></button>{/each}</div>{#if searching}<p role="status">Recherche en ligne…</p>{:else if !results.length}<p role="status">{query.trim().length<2?'Saisissez au moins deux caractères.':'Aucun résultat. Essayez le titre original ou un artiste.'}</p>{/if}{#if warning}<p class="folder-help">{warning}</p>{/if}<div class="search-footnote">Entrée pour ouvrir · Échap pour fermer</div></div>{/if}
</div>
