<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import WorkPoster from './WorkPoster.svelte';
  import type { Entry, Work } from './types';
  export let work: Work;
  export let token: string;
  export let entries: Entry[];
  export let changed: (work: Work) => void;
  export let recommend: (work: Work) => void;
  export let play: (entry: Entry) => void;
  export let close: () => void;
  let dialog: HTMLDialogElement;
  let busy = true;
  let message = '';
  let alive = true;
  const controller = new AbortController();
  $: local = entries.filter(e => work.localIds.includes(e.id)).sort((a,b) => (a.metadata?.season ?? 0) - (b.metadata?.season ?? 0) || (a.metadata?.episode ?? 0) - (b.metadata?.episode ?? 0) || a.filename.localeCompare(b.filename));
  onMount(() => { dialog.showModal(); void load(); return () => { alive = false; controller.abort(); }; });
  async function load() {
    try {
      const response = await fetch(`/api/v1/discovery/works/${work.id}`, { signal: controller.signal });
      const data = await response.json(); if (!response.ok) throw new Error(data.error);
      if (alive) { work = data.work; message = data.warning; changed(work); }
    } catch(e) { if (alive) message = e instanceof Error ? e.message : 'Fiche indisponible.'; }
    finally { if (alive) busy = false; }
  }
  async function toggle(field: 'favorite' | 'watchlist') {
    busy = true; message = '';
    try {
      const response = await fetch(`/api/v1/discovery/works/${work.id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json', 'X-Mosaune-Token': token }, body: JSON.stringify({ [field]: !work[field] }), signal: controller.signal });
      const data = await response.json(); if (!response.ok) throw new Error(data.error);
      if (alive) { work = data; changed(work); }
    } catch(e) { if (alive) message = e instanceof Error ? e.message : 'Liste non enregistrée.'; }
    finally { if (alive) busy = false; }
  }
</script>
<dialog class="media-dialog work-dialog" bind:this={dialog} onclose={close} aria-labelledby="work-title">
  <div class="dialog-heading"><span class="eyebrow">{work.kind === 'films' ? 'CINÉMA' : 'SÉRIES'} · {work.metadata.provider}</span><button class="icon-button" aria-label="Fermer la fiche" onclick={() => dialog.close()}><Icon name="close"/></button></div>
  <div class="detail-intro"><WorkPoster {work}/><div><h2 id="work-title">{work.metadata.title}</h2><p>{work.metadata.year || ''}</p><p>{work.metadata.genres?.join(' · ')}</p><span class="availability" class:available={local.length > 0}>{local.length ? work.kind === 'series' ? `${local.length} fichier(s) d’épisode disponible(s)` : 'Dans votre bibliothèque' : 'À découvrir · absent du serveur'}</span></div></div>
  <div class="work-actions"><button class="secondary" aria-pressed={work.favorite} disabled={busy} onclick={() => toggle('favorite')}>{work.favorite ? '♥ Dans vos favoris' : '♡ Ajouter aux favoris'}</button><button class="secondary" aria-pressed={work.watchlist} disabled={busy} onclick={() => toggle('watchlist')}>{work.watchlist ? '✓ Dans À voir' : '+ Garder dans À voir'}</button><button class="secondary" disabled={busy} onclick={() => recommend(work)}>Trouver des œuvres similaires</button></div>
  {#if busy}<p class="folder-help" role="status">Chargement…</p>{/if}
  {#if message}<p class="notice" role="status">{message}</p>{/if}
  <p class="synopsis">{work.metadata.overview || 'Résumé non disponible.'}</p>
  {#if work.metadata.cast?.length}<h3>Distribution</h3><p class="synopsis">{work.metadata.cast.join(' · ')}</p>{/if}
  {#if local.length}<section class="work-local"><h3>{work.kind === 'series' ? 'Vos épisodes disponibles' : 'Disponible chez vous'}</h3>
    {#if work.kind === 'series'}<p class="folder-help">Cette liste décrit vos fichiers, pas la disponibilité de la série complète.</p>{/if}
    {#each local as entry}<div class="work-episode"><div><strong>{entry.kind === 'series' ? `S${String(entry.metadata?.season ?? 0).padStart(2,'0')} · E${String(entry.metadata?.episode ?? 0).padStart(2,'0')}${entry.metadata?.episodeEnd ? `–${entry.metadata.episodeEnd}` : ''} — ` : ''}{entry.metadata?.title || entry.name}</strong><small>{entry.completed ? 'Déjà vu' : (entry.position ?? 0) > 0 ? 'Lecture en cours' : 'Prêt à lire'}</small></div><button class="secondary" onclick={() => { play(entry); close(); }}>{(entry.position ?? 0) > 0 && !entry.completed ? 'Reprendre' : 'Lire'}</button></div>{/each}
  </section>{:else}<p class="folder-help">Gardez cette œuvre dans À voir pour la retrouver. La demande de téléchargement via Sonarr/Radarr arrivera dans une prochaine étape.</p>{/if}
</dialog>
