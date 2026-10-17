<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';
  import { moveItem, type Playlist } from './music';
  import type { Entry } from './types';
  export let show: boolean;
  export let token: string;
  export let entries: Entry[];
  export let pending: Entry[] | null = null;
  export let play: (entries: Entry[], start?: number) => void;
  export let append: (entries: Entry[]) => void;
  let playlists: Playlist[] = [];
 let coverInput:HTMLInputElement;
 async function cover(event:Event){const file=(event.currentTarget as HTMLInputElement).files?.[0];if(!file||!current)return;if(file.size>8*1024*1024){error="Image trop volumineuse (8 Mo maximum).";return;}busy=true;error="";try{const r=await fetch(`/api/v1/music/playlists/${current.id}/cover?revision=${current.revision}`,{method:"PUT",headers:{"X-Mosaune-Token":token},body:file});const data=await r.json();if(!r.ok)throw new Error(data.error);playlists=playlists.map(p=>p.id===current?.id?{...p,...data}:p);}catch(e){error=e instanceof Error?e.message:"Pochette non enregistrée.";}finally{busy=false;coverInput.value="";}}
  let selected = '';
  let name = '';
  let newName = '';
  let target = '';
  let busy = false;
  let error = '';
  let message = '';
  let confirmDelete = false;
  let dialog: HTMLDialogElement;
  $: current = playlists.find(p => p.id === selected);
  $: lookup = new Map(entries.filter(e => e.kind === 'music').map(e => [e.id, e]));
  $: playable = current?.tracks.flatMap(t => { const e = lookup.get(t.id); return e ? [e] : []; }) ?? [];
  $: persistent = pending?.filter(e => e.libraryId && e.kind === 'music') ?? [];
  $: if (pending && dialog && !dialog.open) { dialog.showModal(); void load(); }
  onMount(() => { void load(); });

  async function request(path: string, method = 'GET', body?: unknown) {
    const r = await fetch('/api/v1/music/playlists' + path, { method, headers: { 'Content-Type': 'application/json', 'X-Mosaune-Token': token }, body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(15000) });
    if (!r.ok) throw new Error((await r.json()).error ?? 'Opération impossible.');
    return r.status === 204 ? null : await r.json();
  }
  async function load() {
    if (busy) return;
    busy = true; error = '';
    try { playlists = await request(''); name = playlists.find(p => p.id === selected)?.name ?? ''; }
    catch (e) { error = e instanceof Error ? e.message : 'Playlists indisponibles.'; }
    finally { busy = false; }
  }
  function select(p: Playlist) { selected = p.id; name = p.name; confirmDelete = false; error = ''; message = ''; }
  async function save(p: Playlist | undefined, title: string, ids: string[], closeAfter = false) {
    busy = true; error = ''; message = '';
    try {
      const saved: Playlist = await request(p ? '/' + p.id : '', p ? 'PUT' : 'POST', { name: title, trackIds: ids, revision: p?.revision ?? 0 });
      playlists = p ? playlists.map(item => item.id === p.id ? saved : item) : [...playlists, saved];
      select(saved); message = 'Playlist enregistrée sur le serveur.';
      if (closeAfter) closeDialog();
    } catch (e) { error = e instanceof Error ? e.message : 'Enregistrement impossible.'; }
    finally { busy = false; }
  }
  async function erase(p: Playlist) {
    busy = true; error = '';
    try { await request('/' + p.id, 'DELETE', { revision: p.revision }); playlists = playlists.filter(item => item.id !== p.id); selected = ''; name = ''; confirmDelete = false; message = 'Playlist supprimée. Vos fichiers restent dans la bibliothèque.'; }
    catch (e) { error = e instanceof Error ? e.message : 'Suppression impossible.'; }
    finally { busy = false; }
  }
  function reorder(p: Playlist, from: number, to: number) { void save(p, p.name, moveItem(p.tracks, from, to).map(t => t.id)); }
  function closeDialog() { pending = null; dialog.close(); newName = ''; target = ''; }
  function addPending() {
    const p = playlists.find(item => item.id === target);
    if (target && !p) { error = 'Cette playlist n’existe plus. Choisissez-en une autre.'; return; }
    void save(p, p?.name ?? newName, [...(p?.tracks.map(t => t.id) ?? []), ...persistent.map(e => e.id)], true);
  }
</script>

<section class="playlist-manager" hidden={!show} aria-labelledby="playlists-title">
  {#if !current}<div class="section-heading"><div><span class="eyebrow">LES MORCEAUX QUI VONT ENSEMBLE</span><h2 id="playlists-title">Vos playlists</h2></div><button class="secondary" disabled={busy} onclick={() => { pending = []; }}>Nouvelle playlist</button></div>
  <p class="folder-help">Enregistrées sur votre serveur. Ajoutez des morceaux depuis un album, ou enregistrez votre file d’écoute.</p>
  {#if error && !pending}<p class="playback-error" role="alert">{error} <button class="restart-button" disabled={busy} onclick={load}>Recharger les playlists</button></p>{/if}
  {#if message}<p class="notice" role="status">{message}</p>{/if}
  <div class="playlist-tabs">{#each playlists as p (p.id)}<button class:selected={selected === p.id} aria-pressed={selected === p.id} disabled={busy} onclick={() => select(p)}>{#if p.imageId}<img src={`/api/v1/music/playlists/${p.id}/cover?v=${p.revision}`} alt=""/>{:else}<span class="playlist-thumb"><Icon name="music" size={32}/></span>{/if}<strong>{p.name}</strong><span>{p.tracks.length} pistes</span></button>{/each}</div>
  {#if !playlists.length && !busy && !error}<p class="folder-help">Un trajet, une soirée, un album à retrouver… Créez votre première playlist.</p>{/if}
  {/if}
  {#if current}
    <button class="back-link" onclick={()=>selected=""}>← Toutes les playlists</button>
    <div class="playlist-hero"><button class="playlist-cover" aria-label="Changer la pochette de la playlist" disabled={busy} onclick={()=>coverInput.click()}>{#if current.imageId}<img src={`/api/v1/music/playlists/${current.id}/cover?v=${current.revision}`} alt=""/>{:else}<Icon name="music" size={64}/>{/if}<span>Changer la pochette</span></button><div><span class="eyebrow">VOTRE PLAYLIST</span><h1>{current.name}</h1><p>{current.tracks.length} morceaux · dans votre bibliothèque</p></div></div>
    <form class="playlist-name" onsubmit={e => { e.preventDefault(); if (current) void save(current, name, current.tracks.map(t => t.id)); }}><label>Nom de la playlist<input bind:value={name} required maxlength="80" disabled={busy}/></label><button class="secondary" disabled={busy || !name.trim() || name.trim() === current.name}>Renommer</button></form>
    <div class="music-actions"><button class="primary" disabled={!playable.length || busy} onclick={() => play(playable)}>Lire la playlist</button><button class="secondary" disabled={!playable.length || busy} onclick={() => append(playable)}>Ajouter à la file</button><button class="restart-button" disabled={busy} onclick={() => confirmDelete = !confirmDelete}>Supprimer la playlist</button></div>
    {#if playable.length !== current.tracks.length}<p class="folder-help">{current.tracks.length - playable.length} morceau(x) indisponible(s) : conservés dans la playlist et ignorés pendant la lecture.</p>{/if}
    {#if confirmDelete}<div class="delete-playlist"><p>Supprimer « {current.name} » ? Les fichiers musicaux seront conservés.</p><button class="secondary" disabled={busy} onclick={() => { if (current) void erase(current); }}>Confirmer la suppression</button><button class="restart-button" disabled={busy} onclick={() => confirmDelete = false}>Annuler</button></div>{/if}
    <ol class="music-track-list">
      {#each current.tracks as track, index}
        {@const entry = lookup.get(track.id)}
        <li><span class="track-number">{index + 1}</span><button class="track-title" disabled={!entry || busy} onclick={() => { if (current) play(playable, current.tracks.slice(0, index).filter(t => lookup.has(t.id)).length); }}><strong>{entry?.metadata?.title ?? entry?.name ?? track.name}</strong><small>{entry ? `${entry.metadata?.artist ?? ''} · ${entry.filename.split('.').pop()?.toUpperCase()}` : 'Indisponible'}</small></button><div class="track-actions"><button disabled={busy || index === 0} aria-label={`Monter la piste ${index + 1}`} onclick={() => { if (current) reorder(current, index, index - 1); }}>↑</button><button disabled={busy || index === current.tracks.length - 1} aria-label={`Descendre la piste ${index + 1}`} onclick={() => { if (current) reorder(current, index, index + 1); }}>↓</button><button disabled={busy} aria-label={`Retirer la piste ${index + 1} de la playlist`} onclick={() => { if (current) void save(current, current.name, current.tracks.filter((_, i) => i !== index).map(t => t.id)); }}>×</button></div></li>
      {/each}
    </ol>
  {/if}
  <input class="file-input" bind:this={coverInput} type="file" accept="image/jpeg,image/png" onchange={cover} aria-label="Choisir une pochette"/>
  {#if current&&error}<p class="playback-error" role="alert">{error}<button onclick={load}>Recharger</button></p>{/if}
</section>

<dialog bind:this={dialog} class="media-dialog" aria-labelledby="playlist-dialog-title" onclose={() => { pending = null; }} oncancel={e => { if (busy) e.preventDefault(); }}>
  <div class="dialog-heading"><h2 id="playlist-dialog-title">{pending?.length ? 'Ajouter à une playlist' : 'Nouvelle playlist'}</h2><button class="icon-button" disabled={busy} aria-label="Fermer les playlists" onclick={closeDialog}><Icon name="close"/></button></div>
  <p class="folder-help">{persistent.length} morceau(x) de la bibliothèque à enregistrer. Les playlists peuvent contenir jusqu’à 200 pistes.</p>
  {#if pending && persistent.length !== pending.length}<p class="folder-help">Les fichiers temporaires ne peuvent pas être conservés dans une playlist. Ajoutez d’abord leur dossier à la bibliothèque.</p>{/if}
  <form class="identify-form" onsubmit={e => { e.preventDefault(); addPending(); }}>
    {#if pending?.length}<label>Playlist<select bind:value={target} disabled={busy}><option value="">Créer une nouvelle playlist</option>{#each playlists as p}<option value={p.id}>{p.name} ({p.tracks.length} pistes)</option>{/each}</select></label>{/if}
    {#if !target}<label>Nom<input bind:value={newName} required maxlength="80" disabled={busy} placeholder="Les soirs tranquilles"/></label>{/if}
    <button class="primary" disabled={busy || (!target && !newName.trim()) || (!!pending?.length && !persistent.length)}>{busy ? 'Patientez…' : 'Enregistrer'}</button>
  </form>
  {#if error}<p class="playback-error" role="alert">{error} <button class="restart-button" disabled={busy} onclick={load}>Recharger les playlists</button></p>{/if}
</dialog>
