<script lang="ts">
  import { onMount } from 'svelte';
  import type { Entry, Provider } from './types';
  import Icon from './Icon.svelte';
  export let entry: Entry;
  export let providers: Provider[];
  export let token: string;
  export let refresh: () => Promise<void>;
  export let close: () => void;
  export let discover: (entry: Entry) => void;
  let dialog: HTMLDialogElement;
  let query=entry.metadata?.series || entry.metadata?.title || entry.name;
  let provider='';
  let busy=false;
  let message='';
  let results: {id:string;title:string;year:number;overview:string}[]=[];
  $: compatible=providers.filter(p=>p.enabled&&p.kinds.includes(entry.kind));
  $: if (!compatible.some(p=>p.id===provider)) provider=compatible[0]?.id ?? '';
  $: configured=compatible.find(p=>p.id===provider)?.configured;
  $: m=entry.metadata;
  onMount(()=>{dialog.showModal();});
  async function search() {
    busy=true;message='';results=[];
    try { const r=await fetch(`/api/v1/media/${entry.id}/search-metadata`,{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({provider,query})});const data=await r.json();if(!r.ok) throw new Error(data.error);results=data;if(!results.length) message='Aucune correspondance. Essayez le titre original.'; }
    catch(e){message=e instanceof Error ? e.message : 'Recherche impossible.';}
    finally{busy=false;}
  }
  async function identify(id:string) {
    busy=true;message='';
    try { const r=await fetch(`/api/v1/media/${entry.id}/identify`,{method:'PUT',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({provider,id})});if(!r.ok) throw new Error((await r.json()).error);await refresh();results=[];message='Correspondance enregistrée. Les prochaines analyses la conserveront.'; }
    catch(e){message=e instanceof Error ? e.message : 'Identification impossible.';}
    finally{busy=false;}
  }
</script>

<dialog bind:this={dialog} class="media-dialog" onclose={close} aria-labelledby="detail-title">
  <div class="dialog-heading"><span class="eyebrow">VOTRE COLLECTION</span><button class="icon-button" aria-label="Fermer la fiche" onclick={()=>dialog.close()}><Icon name="close"/></button></div>
  <div class="detail-intro">
    {#if m?.imageId}<img src={`/api/v1/artwork/${m.imageId}`} alt={`Couverture de ${m.series || m.album || m.title}`} />{/if}
    <div><h2 id="detail-title">{m?.series || m?.title || entry.name}</h2>{#if m?.series}<h3>S{String(m.season??0).padStart(2,'0')} E{String(m.episode??0).padStart(2,'0')} · {m.title}</h3>{/if}
    <p>{[m?.artist,m?.album,m?.year].filter(Boolean).join(' · ')}</p><p>{m?.genres?.join(' · ')}</p></div>
  </div>
  {#if m?.overview}<p class="synopsis">{m.overview}</p>{/if}
  {#if m?.cast?.length}<h3>Distribution</h3><p class="synopsis">{m.cast.join(' · ')}</p>{/if}
  <p class="folder-help">{m?.providerId ? `Source : ${providers.find(p=>p.id===m.provider)?.name ?? m.provider}${m.manual ? ' · Correspondance choisie manuellement' : ''}` : m?.provider==='tags' ? 'Informations intégrées au fichier musical' : 'Informations déduites du nom du fichier'}.</p>
  <details class="file-detail"><summary>Fichier d’origine</summary><p>{entry.filename}</p><p>{(entry.size/1024**2).toFixed(1)} Mo</p></details>
  {#if entry.kind!=='music'}
    {#if m?.providerId}<div class="work-actions"><button class="primary" onclick={() => discover(entry)}>Favoris, À voir et suggestions</button></div>{:else}<p class="folder-help">Identifiez cette œuvre pour la retrouver dans vos favoris et votre liste À voir.</p>{/if}
    <section class="identify-section"><h3>{m?.providerId ? 'Corriger la correspondance' : 'Identifier ce média'}</h3>
      {#if compatible.length}
        <form class="identify-form" onsubmit={e=>{e.preventDefault();void search();}}>
          <label>Fournisseur<select bind:value={provider} disabled={busy}>{#each compatible as p}<option value={p.id}>{p.name}</option>{/each}</select></label>
          <label>Titre à rechercher<input bind:value={query} minlength="2" maxlength="250" required disabled={busy}/></label>
          <button class="primary" disabled={busy||!configured}>{busy ? 'Patientez…' : 'Rechercher'}</button>
        </form>
        {#if !configured}<p class="folder-help">Configurez ce fournisseur dans « Affiches et fournisseurs » pour lancer la recherche.</p>{/if}
      {:else}<p>Aucun fournisseur compatible n’est installé.</p>{/if}
      {#each results as candidate}<div class="candidate"><strong>{candidate.title}{candidate.year ? ` (${candidate.year})` : ''}</strong><p>{candidate.overview || 'Résumé indisponible.'}</p><button class="secondary" disabled={busy} onclick={()=>identify(candidate.id)}>Choisir pour ce fichier</button></div>{/each}
      {#if message}<p class="notice" role="status">{message}</p>{/if}
    </section>
  {/if}
</dialog>
