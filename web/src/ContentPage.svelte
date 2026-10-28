<script lang="ts">
 import {permitted,canRequest,currentAccount} from './accounts';
 import DetailActions from './DetailActions.svelte';import DetailExperienceSections from './DetailExperienceSections.svelte';import {actionEntries,type DetailAction,type ExperienceView} from './detail-experience';import DetailHeader from './DetailHeader.svelte';import DetailList from './DetailList.svelte';import DetailSections from './DetailSections.svelte';
 import RequestDownload from './RequestDownload.svelte';import PosterEditor from './PosterEditor.svelte';
 import {onMount} from 'svelte';import AlbumMatch from './AlbumMatch.svelte';import Icon from './Icon.svelte';import MediaShelf from './MediaShelf.svelte';import {groupMedia,compareCatalogueEntries} from './collection';import {groupCard,workCard,type Card} from './experience';import type {Entry,Work,Provider,Contributor} from './types';
 export let card:Card;export let entries:Entry[];export let works:Work[];export let providers:Provider[];export let token:string;export let play:(e:Entry,restart?:boolean)=>void;export let enqueue:(e:Entry[])=>void;export let playlist:(e:Entry[])=>void;export let inspect:(e:Entry)=>void;export let open:(c:Card)=>void;export let person:(provider:string,person:Contributor)=>void;export let artist:(name:string)=>void;export let changed:(w:Work)=>void;export let refresh:()=>Promise<void>;export let mark:(e:Entry,v:boolean)=>void;
 let editingPoster=false;let editingEntry:Entry|undefined;let matchingEntry:Entry|undefined;let view:ExperienceView|null=null;let experienceWarning='';let mounted=false;let experienceSignature='';let experienceRequest:AbortController|undefined;let posterEpoch=0;let downloadSeason:number|null=null;let matching=false;let failedImage='';let work:Work|undefined=card.work;let busy=false;let warning='';let related:Work[]=[];let alive=true;let requested='';const abort=new AbortController();
 $: local=work?entries.filter(e=>work?.localIds.includes(e.id)):card.group?entries.filter(e=>card.group?.entries.some(a=>a.id===e.id)):card.entry?[entries.find(e=>e.id===card.entry?.id)||card.entry]:[];
 $: sorted=[...local].sort(compareCatalogueEntries);
 $: trackMode=!!card.entry&&card.entry.kind==='music';
 $: m=work?.metadata||card.group?.entries[0]?.metadata||card.entry?.metadata;
 $: title=trackMode?(card.entry?.metadata?.title||card.title):(work?.metadata.title||card.title);
 $: automaticPicture=work?.poster?`/api/v1/discovery/works/${work.id}/poster?automatic=1`:local.find(e=>e.metadata?.imageId)?.metadata?.imageId?`/api/v1/artwork/${local.find(e=>e.metadata?.imageId)!.metadata!.imageId}`:card.image?.startsWith('/api/v1/posters/')?undefined:card.image;
 $: picture=local.find(e=>e.posterUrl)?.posterUrl||work?.posterUrl||automaticPicture;
 $: fallbackImage=card.group?.imageId?`/api/v1/artwork/${card.group.imageId}`:card.entry?.metadata?.imageId?`/api/v1/artwork/${card.entry.metadata.imageId}`:undefined;
 $: displayedImage=failedImage===picture?fallbackImage:picture;
 $: album=groupMedia(entries).find(g=>g.entries.some(e=>e.id===card.entry?.id));
 $: experienceBody=JSON.stringify({kind:card.kind,workId:work?.id||'',ids:local.map(e=>e.id),trackId:trackMode?card.entry?.id||'':''});
 $: signature=token+experienceBody+JSON.stringify([m,local.map(e=>[e.metadata,e.completed,e.revision])]);
 $: if(mounted&&signature!==experienceSignature){experienceSignature=signature;void loadExperience(experienceBody);}
 onMount(()=>{mounted=true;void load();const changed=()=>{experienceSignature='';};window.addEventListener('mosaune:detail-changed',changed);return()=>{mounted=false;alive=false;abort.abort();experienceRequest?.abort();window.removeEventListener('mosaune:detail-changed',changed)};});
 async function loadExperience(body:string){experienceRequest?.abort();const controller=new AbortController();experienceRequest=controller;view=null;related=[];experienceWarning='';try{const r=await fetch('/api/v1/detail-experience',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body,signal:controller.signal});const data=await r.json();if(!r.ok)throw Error(data.error||'Fiche indisponible.');if(!alive||controller.signal.aborted)return;view=data.view;experienceWarning=data.warning||'';
 if(view?.suggestions.enabled&&work&&providers.some(p=>p.id===work?.metadata.provider&&p.enabled&&p.configured&&p.discovery)){const response=await fetch('/api/v1/discovery/browse',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({provider:work.metadata.provider,kind:work.kind,seed:work.id,page:1}),signal:controller.signal});if(!response.ok)throw Error('Suggestions indisponibles.');const more=await response.json();if(alive&&!controller.signal.aborted)related=more.items;}
 }catch(e){if(alive&&!controller.signal.aborted)experienceWarning=e instanceof Error?e.message:'Présentation indisponible.';}}
 function allowed(a:DetailAction){const selected=actionEntries(a,local);const entry=selected[0];switch(a.operation){case 'play':return !!entry&&permitted(entry,'play');case 'enqueue':case 'playlist':return selected.length>0&&selected.every(e=>permitted(e,'play'));case 'open':case 'inspect':case 'mark':return !!entry;case 'poster':return entry?permitted(entry,'edit'):!!work&&!!currentAccount?.admin;case 'match':return !!entry&&permitted(entry,'edit');case 'request':return !!work&&canRequest(card.kind);case 'watchlist':case 'favorite':return !!work;case 'unavailable':return true;default:return false}}
 function runAction(a:DetailAction){if(!allowed(a))return;const selected=actionEntries(a,sorted),entry=selected[0];switch(a.operation){case 'play':play(entry);break;case 'open':openTrack(entry);break;case 'enqueue':enqueue(selected);break;case 'playlist':playlist(selected);break;case 'mark':mark(entry,!entry.completed);break;case 'inspect':inspect(entry);break;case 'poster':editingEntry=entry;editingPoster=true;break;case 'match':matchingEntry=entry;matching=true;break;case 'watchlist':case 'favorite':void toggle(a.operation);break;case 'request':downloadSeason=-1;break;case 'unavailable':requestLater(title);break;}}

 async function request(path:string,method='GET',body?:unknown){const r=await fetch('/api/v1/'+path,{method,headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:body===undefined?undefined:JSON.stringify(body),signal:abort.signal});if(r.status===204)return null;const data=await r.json();if(!r.ok)throw new Error(data.error||'Opération indisponible.');return data;}
 async function load(){busy=true;try{
 if(!work){const entry=card.group?.entries[0]||card.entry;work=works.find(w=>w.localIds.includes(entry?.id??''));}
 if(work){const data=await request('discovery/works/'+work.id);if(!alive)return;work=data.work;warning=data.warning;changed(data.work);}
 }catch(e){if(alive)warning=e instanceof Error?e.message:'Fiche indisponible.';}finally{if(alive)busy=false;}}
 async function toggle(field:'favorite'|'watchlist'){if(!work)return;busy=true;try{const updated=await request('discovery/works/'+work.id,'PATCH',{[field]:!work[field]});if(alive){work=updated;changed(updated);}}catch(e){if(alive)warning=e instanceof Error?e.message:'Liste non enregistrée.';}finally{if(alive)busy=false;}}
 async function posterSaved(){await refresh();if(work){const data=await request('discovery/works/'+work.id);work=data.work;changed(data.work);}failedImage='';posterEpoch++;}
 async function matched(candidate:Work){await refresh();const data=await request('discovery/works/'+candidate.id);work=data.work;changed(data.work);await load();}

 function openTrack(entry:Entry){open({key:entry.id,title:entry.metadata?.title||entry.name,subtitle:entry.metadata?.artist||'',kind:entry.kind,local:true,entry,image:picture});}
 function requestLater(title:string){requested=`« ${title} » : les demandes de téléchargement ne sont pas encore raccordées. Aucune demande n’a été envoyée.`;}
</script>
{#if editingPoster}<PosterEditor scope={editingEntry?.libraryId?'media':'works'} id={editingEntry?.libraryId?editingEntry.id:work!.id} {title} current={displayedImage} {token} saved={posterSaved} close={()=>editingPoster=false}/>{/if}
<article class="content-page">
 <section class="content-hero" class:album-hero={view?.cover==='square'}>
 <div class="content-cover">{#if displayedImage}<img src={displayedImage+(displayedImage.includes('?')?'&':'?')+'display='+posterEpoch} alt={`Affiche de ${title}`} onerror={()=>{if(displayedImage===fallbackImage)fallbackImage=undefined;failedImage=picture||'';}}/>{:else}<Icon name={card.kind} size={64}/>{/if}</div>
 <div class="content-copy"><DetailHeader entries={local} workId={work?.id||''} kind={card.kind} trackId={trackMode?card.entry?.id||'':''} {token} revision={JSON.stringify(m)} fallbackTitle={title}>
 <svelte:fragment slot="people">{#if view?.people.length}<div class="artist-links">{#each view.people as index}{@const p=view.peopleResources[index]}{#if p}<button onclick={()=>p.id?person(view!.provider,p):artist(p.name)}>{p.name}</button>{/if}{/each}</div>{/if}</svelte:fragment>
 <div slot="commands" class="content-actions">{#if view}<DetailActions actions={view.actions} run={runAction} {allowed} {busy} watchlist={work?.watchlist||false} favorite={work?.favorite||false}/>{/if}</div>
 </DetailHeader></div></section>
 {#if busy}<p class="folder-help" role="status">La fiche se complète…</p>{/if}{#if warning}<p class="notice" role="status">{warning}</p>{/if}{#if requested}<p class="notice" role="status">{requested}</p>{/if}
 <DetailSections entries={local} workId={work?.id||''} kind={card.kind} {token} {person} revision={JSON.stringify(m)}/>
 {#if experienceWarning}<p class="notice" role="status">{experienceWarning}</p>{/if}
 {#if view}<DetailExperienceSections {view} run={runAction} {allowed} {busy}/>{#if view.episodeList}<DetailList entries={local} workId={work?.id||''} kind={card.kind} {token} {play} {mark} {inspect} requestGroup={season=>downloadSeason=season}/>{/if}
 {#if view.albumLink&&album}<button class="album-return" onclick={()=>open(groupCard(album))}>Retrouver « {album.title} » →</button>{/if}
 {:else if experienceWarning&&local.length}<section class="track-section"><h2>Vos fichiers</h2>{#each local as entry}<button class="secondary" disabled={!permitted(entry,'play')} onclick={()=>play(entry)}>{entry.name}</button>{/each}</section>{/if}
 {#if view?.suggestions.enabled&&related.length}<MediaShelf title={view.suggestions.title} subtitle={view.suggestions.help} cards={related.map(workCard)} {open}/>{/if}
 <p class="content-source">{work?`Informations : ${providers.find(p=>p.id===work?.metadata.provider)?.name||work.metadata.provider}`:'Informations issues de vos fichiers'} · Votre collection reste chez vous.</p>
</article>

{#if matching&&matchingEntry}<AlbumMatch entry={matchingEntry} {token} {matched} close={()=>matching=false}/>{/if}

{#if downloadSeason!==null&&work}<RequestDownload {work} season={downloadSeason} {token} close={()=>downloadSeason=null}/>{/if}
