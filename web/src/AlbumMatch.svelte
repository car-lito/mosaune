<script lang="ts">
 import {onMount} from 'svelte';
 import Icon from './Icon.svelte';
 import type {Entry,Work} from './types';
 export let entry:Entry;
 export let token:string;
 export let matched:(work:Work)=>Promise<void>;
 export let close:()=>void;
 let dialog:HTMLDialogElement;
 let title=entry.metadata?.album||'';
 let artist=entry.metadata?.albumArtist||entry.metadata?.artist||'';
 let releaseId='';let page=1;let pages=1;
 let candidates:Work[]=[];let selected:Work|undefined;
 let busy=false;let saving=false;let message='';let searched=false;let alive=true;
 let failed:Record<string,boolean>={};
 const abort=new AbortController();
 onMount(()=>{dialog.showModal();void search(1);return()=>{alive=false;abort.abort();};});
 async function request(path:string,method='GET',body?:unknown){const r=await fetch('/api/v1/'+path,{method,headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:body===undefined?undefined:JSON.stringify(body),signal:abort.signal});if(r.status===204)return null;const data=await r.json();if(!r.ok)throw new Error(data.error||'Opération indisponible.');return data;}
 async function search(next:number){busy=true;message='';selected=undefined;try{const data=await request('music/matches','POST',{title,artist,id:releaseId,page:next});if(alive){candidates=data.items;page=data.page;pages=data.pages;searched=true;}}catch(e){if(alive)message=e instanceof Error?e.message:'Recherche impossible.';}finally{if(alive)busy=false;}}
 async function preview(work:Work){busy=true;message='';try{const data=await request('discovery/works/'+work.id);if(data.warning)throw new Error(data.warning);if(alive)selected=data.work;}catch(e){if(alive)message=e instanceof Error?e.message:'Fiche indisponible.';}finally{if(alive)busy=false;}}
 async function associate(){if(!selected)return;saving=true;busy=true;message='';try{await request('music/associate','POST',{mediaId:entry.id,workId:selected.id});await matched(selected);if(alive)dialog.close();}catch(e){if(alive)message=e instanceof Error?e.message:'Association impossible.';}finally{if(alive){busy=false;saving=false;}}}
</script>
<dialog bind:this={dialog} class="media-dialog album-match" onclose={close} oncancel={e=>{if(saving)e.preventDefault();}} aria-labelledby="album-match-title">
 <div class="dialog-heading"><div><span class="eyebrow">LES BONNES INFORMATIONS</span><h2 id="album-match-title">Corriger l’association</h2></div><button class="icon-button" disabled={saving} aria-label="Fermer" onclick={()=>dialog.close()}><Icon name="close"/></button></div>
 <p>Choisissez l’édition de votre album. Vos fichiers et leurs tags restent intacts.</p>
 <form class="match-search" onsubmit={e=>{e.preventDefault();void search(1);}}>
  <label>Album<input bind:value={title} maxlength="250" disabled={busy||!!releaseId} required={!releaseId}/></label>
  <label>Artiste<input bind:value={artist} maxlength="250" disabled={busy||!!releaseId} placeholder="Facultatif"/></label>
  <details><summary>Utiliser un lien ou un identifiant MusicBrainz</summary><label>Édition MusicBrainz<input bind:value={releaseId} maxlength="200" disabled={busy} placeholder="https://musicbrainz.org/release/…"/></label><small>Effacez ce champ pour rechercher par album et artiste.</small></details>
  <button class="primary" disabled={busy}>{busy?'Patientez…':'Rechercher'}</button>
 </form>
 {#if message}<p class="notice" role="alert">{message}</p>{/if}
 {#if selected}<section class="match-preview"><div class="match-preview-title"><h3>{selected.metadata.title}</h3><p>{selected.metadata.artist} · {selected.metadata.edition}</p></div><ol>{#each selected.metadata.tracks||[] as track}<li>{track.disc>1?`${track.disc}.`:''}{track.number}. {track.title}</li>{/each}</ol><button class="primary" disabled={busy||!selected.metadata.tracks?.length} onclick={associate}>Associer cet album</button><button class="text-action" disabled={busy} onclick={()=>selected=undefined}>Revenir aux résultats</button></section>{:else}
  <div class="match-results">{#each candidates as work}<button class="match-candidate" disabled={busy} onclick={()=>preview(work)}>{#if work.poster&&!failed[work.id]}<img src={`/api/v1/discovery/works/${work.id}/poster`} alt="" loading="lazy" onerror={()=>failed={...failed,[work.id]:true}}/>{:else}<span class="match-cover-placeholder"><Icon name="music"/></span>{/if}<span><strong>{work.metadata.title}</strong><span>{work.metadata.artist}</span><small>{work.metadata.edition||'Édition sans précision'}</small></span><span>Vérifier →</span></button>{/each}</div>
  {#if searched&&!busy&&!candidates.length}<p>Aucune édition trouvée. Essayez un titre plus court, sans « édition deluxe », ou retirez le filtre artiste.</p>{/if}
  {#if pages>1}<div class="discovery-pagination"><button class="secondary" disabled={busy||page<=1} onclick={()=>search(page-1)}>Précédent</button><span>{page} / {pages}</span><button class="secondary" disabled={busy||page>=pages} onclick={()=>search(page+1)}>Suivant</button></div>{/if}
 {/if}
</dialog>
