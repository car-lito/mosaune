<script lang="ts">
  import type { MusicQueue, QueueItem } from './music';
  import type { Entry } from './types';
  import Icon from './Icon.svelte';
  export let queue: MusicQueue;
  export let entries: Entry[];
  export let activeKey: string | null;
  export let play: (item: QueueItem) => void;
  export let move: (from: number, to: number) => void;
  export let remove: (key: string) => void;
  export let clear: () => void;
  export let shuffle: () => void;
  export let save: () => void;
  export let close: () => void;
  $: lookup = new Map(entries.filter(e => e.kind === 'music').map(e => [e.id, e]));
  $: resume = queue.items.find(item => item.key === queue.current && lookup.has(item.id)) ?? queue.items.find(item => lookup.has(item.id));
</script>

<section class="queue-panel" aria-label="File d’écoute">
  <div class="dialog-heading"><div><span class="eyebrow">À VOTRE RYTHME</span><h2>File d’écoute <small>{queue.items.length}/200</small></h2></div><button class="icon-button" aria-label="Fermer la file d’écoute" onclick={close}><Icon name="close"/></button></div>
  <p class="folder-help">La file suit votre navigation. Seuls les morceaux de la bibliothèque sont conservés au rechargement, sans lecture automatique.</p>
  <div class="music-actions">{#if !activeKey}<button class="primary" disabled={!resume} onclick={() => { if (resume) play(resume); }}>Reprendre la file</button>{/if}<button class="secondary" disabled={queue.items.length < 2} onclick={shuffle}>Mélanger la suite</button><button class="secondary" disabled={!queue.items.length} onclick={save}>Créer une playlist</button><button class="restart-button" disabled={!queue.items.length} onclick={clear}>Vider la file</button></div>
  {#if !queue.items.length}<p class="folder-help">Lancez un album ou ajoutez des morceaux avec « À la file ».</p>{/if}
  <ol class="music-track-list">
    {#each queue.items as item, index (item.key)}
      {@const entry = lookup.get(item.id)}
      <li class:playing={activeKey === item.key}>
        <span class="track-number">{index + 1}</span>
        <button class="track-title" disabled={!entry} onclick={() => play(item)} aria-label={`Lire ${entry?.name ?? 'morceau indisponible'}`}><strong>{entry?.metadata?.title ?? entry?.name ?? 'Morceau indisponible'}</strong><small>{activeKey === item.key ? 'En lecture · ' : ''}{entry ? `${entry.metadata?.artist ?? ''} · ${entry.filename.split('.').pop()?.toUpperCase()}${!entry.libraryId ? ' · Temporaire' : ''}` : 'Vérifiez le dossier puis actualisez-le.'}</small></button>
        <div class="track-actions"><button disabled={index === 0} aria-label={`Monter le morceau ${index + 1}`} onclick={() => move(index, index - 1)}>↑</button><button disabled={index === queue.items.length - 1} aria-label={`Descendre le morceau ${index + 1}`} onclick={() => move(index, index + 1)}>↓</button><button aria-label={`Retirer le morceau ${index + 1} de la file`} onclick={() => remove(item.key)}>×</button></div>
      </li>
    {/each}
  </ol>
</section>
