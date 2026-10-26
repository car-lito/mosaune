<script lang="ts">
 import {onMount} from 'svelte';import Icon from './Icon.svelte';import {timeLabel} from './experience';import {canRequest} from './accounts';import {detailRows,groupCount,type DetailView} from './detail-list';import type {Entry,Kind} from './types';
 export let entries:Entry[];export let workId='';export let kind:Kind;export let token:string;export let play:(entry:Entry)=>void;export let mark:(entry:Entry,value:boolean)=>void;export let inspect:(entry:Entry)=>void;export let requestGroup:(season:number)=>void;
 let view:DetailView|null=null;let selected:string|null=null;let warning='';let loading=false;let mounted=false;let previous='';let controller:AbortController|undefined;let failed:Record<string,boolean>={};
 $: payload=JSON.stringify({kind,workId,ids:entries.map(e=>e.id)});
 $: signature=token+payload;
 $: if(mounted&&previous!==signature){previous=signature;void load(payload);}
 $: rows=detailRows(view,entries,selected);
 $: group=view?.groups.find(g=>g.key===selected);
 onMount(()=>{mounted=true;const changed=()=>{previous='';};window.addEventListener('mosaune:detail-changed',changed);return()=>{mounted=false;controller?.abort();window.removeEventListener('mosaune:detail-changed',changed)}});
 async function load(body:string){controller?.abort();const request=new AbortController();controller=request;loading=true;view=null;selected=null;failed={};warning='';try{const response=await fetch('/api/v1/detail-list',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body,signal:request.signal});const data=await response.json();if(!response.ok)throw new Error(data.error||'Présentation indisponible.');if(!mounted||request.signal.aborted)return;view=data.view;warning=data.warning||'';}catch(e){if(mounted&&!request.signal.aborted)warning='Présentation indisponible. Vos fichiers restent accessibles individuellement.';}finally{if(mounted&&!request.signal.aborted)loading=false}}
</script>
<section class="episode-section" aria-busy={loading}>
 {#if loading}<p class="folder-help" role="status">La présentation se prépare…</p>{/if}
 {#if warning}<p class="notice" role="status">{warning}</p>{/if}
 {#if view?.groups.length}
 <div class="shelf-heading"><h2>{view.title}</h2><button class="text-action" onclick={()=>selected=null}>{view.all}</button></div>
 <div class="season-rail">{#each view.groups as g (g.key)}{@const count=groupCount(view,g.key,entries)}{@const image=g.artwork===undefined?'':view.images[g.artwork]}
 <button class="season-card" class:selected={selected===g.key} aria-pressed={selected===g.key} onclick={()=>selected=g.key}><span class="season-cover">{#if image&&!failed[g.key]}<img src={image} alt="" loading="lazy" onerror={()=>failed={...failed,[g.key]:true}}/>{:else}<Icon name={kind} size={32}/>{/if}<span>{count?'Chez vous':'À découvrir'}</span></span><strong>{g.title}</strong><small>{count} disponible(s){g.total?' / '+g.total:''}</small></button>
 {/each}</div>{/if}
 <h2>{group?.title||view?.files||'Vos fichiers'}</h2>{#if view?.help}<p class="folder-help">{view.help}</p>{/if}
 {#if !rows.length&&!loading}<p class="folder-help">{view?.empty||'Aucun fichier disponible.'}</p>{/if}
 {#if group?.requestSeason!==undefined&&workId}<button class="secondary" disabled={!canRequest(kind)} onclick={()=>group?.requestSeason!==undefined&&requestGroup(group.requestSeason)}>Demander {group.title}</button>{/if}
 {#each rows as row,index (row.id)}
 {#if selected===null&&row.group&&(index===0||row.group!==rows[index-1].group)}<h3>{view?.groups.find(g=>g.key===row.group)?.title}</h3>{/if}
 <div class="episode-item"><button class="episode-title" onclick={()=>play(row.entry)}><span>{row.ordinal}</span><div><strong>{row.title}</strong><small>{row.entry.completed?'Vu':row.entry.position?'Reprendre à '+timeLabel(row.entry.position):'Prêt à regarder'}</small></div></button><div class="episode-actions"><button class="text-action" onclick={()=>mark(row.entry,!row.entry.completed)}>{row.entry.completed?'Marquer non vu':'Marquer vu'}</button><button class="text-action" onclick={()=>inspect(row.entry)}>Infos fichier</button></div></div>
 {/each}
</section>
