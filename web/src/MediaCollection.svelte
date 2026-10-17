<script lang="ts">
  import Icon from './Icon.svelte';
  import { groupMedia } from './collection';
  import type { Entry } from './types';
  export let entries: Entry[];
  export let activeId: string | undefined;
  export let play: (entry: Entry, restart?: boolean) => void;
  export let inspect: (entry: Entry) => void;
  export let remove: (entry: Entry) => void;
  export let enqueue: (entries: Entry[]) => void;
  export let playlist: (entries: Entry[]) => void;
  let selected = '';
  $: groups = groupMedia(entries);
  $: group = groups.find(g => g.key === selected);
  const time = (s: number) => `${Math.floor(s/60)}:${String(Math.floor(s)%60).padStart(2,'0')}`;
  const format = (e: Entry) => e.filename.split('.').pop()?.toUpperCase() ?? '';
</script>

{#if group}
  <button class="back-link" onclick={() => selected = ''}>← Toute la collection</button>
  <div class="album-heading">
    {#if group.imageId}<img class:poster={group.kind !== 'music'} src={`/api/v1/artwork/${group.imageId}`} alt={`Couverture de ${group.title}`} />{:else}<div class="cover-placeholder small"><Icon name={group.kind} size={35} /></div>{/if}
    <div><span class="eyebrow">{group.subtitle}</span><h3>{group.title}</h3><p>{group.entries.length} {group.kind === 'music' ? 'piste(s)' : group.kind === 'series' ? 'épisode(s)' : 'fichier(s)'}</p><button class="primary" onclick={() => play(group.entries.find(e => !e.completed) ?? group.entries[0])}>Lire <Icon name="play" size={16}/></button></div>
  </div>
  {#if group.kind === 'music'}<div class="music-actions"><button class="secondary" onclick={() => enqueue(group.entries)}>Album à la file</button><button class="secondary" onclick={() => playlist(group.entries)}>Album dans une playlist</button></div>{/if}
  <div class="file-list">
    {#each group.entries as entry, index (entry.id)}
      {#if group.kind === 'series' && (index === 0 || entry.metadata?.season !== group.entries[index-1].metadata?.season)}<h4 class="season-heading">{entry.metadata?.season === 0 ? 'Épisodes spéciaux' : `Saison ${entry.metadata?.season ?? '?'}`}</h4>{/if}
      <div class:playing={activeId === entry.id} class="file-row">
        <button class="file-play {entry.kind}" aria-label={`Lire ${entry.name}`} onclick={() => play(entry)}><Icon name="play" size={20}/></button>
        <button class="file-title" onclick={() => play(entry)}><strong>{entry.metadata?.episode ? `${entry.metadata.episode}${entry.metadata.episodeEnd ? '–'+entry.metadata.episodeEnd : ''}. ` : entry.metadata?.track ? `${entry.metadata.track}. ` : ''}{entry.metadata?.title || entry.name}</strong><span>{entry.metadata?.artist ? entry.metadata.artist+' · ' : ''}{format(entry)} · {entry.libraryId ? 'Bibliothèque' : 'Temporaire'}{entry.completed ? ' · Vu' : entry.position ? ` · Reprendre à ${time(entry.position)}` : ''}</span></button>
        {#if entry.libraryId}<button class="restart-button" onclick={() => inspect(entry)}>Fiche</button>{/if}
        {#if entry.kind === 'music'}<button class="restart-button" onclick={() => enqueue([entry])}>À la file</button>{#if entry.libraryId}<button class="restart-button" onclick={() => playlist([entry])}>Playlist</button>{/if}{/if}
        {#if entry.libraryId && entry.kind !== 'music' && (entry.position || entry.completed)}<button class="restart-button" onclick={() => play(entry,true)}>Du début</button>{/if}
        {#if !entry.libraryId}<button class="icon-button" aria-label={`Retirer ${entry.name}`} onclick={() => remove(entry)}><Icon name="close" size={17}/></button>{/if}
      </div>
    {/each}
  </div>
{:else}
  <div class="cover-grid">
    {#each groups as item (item.key)}
      <button class="cover-card" onclick={() => selected = item.key}>
        <div class="cover-image" class:square={item.kind === 'music'}>
          {#if item.imageId}<img src={`/api/v1/artwork/${item.imageId}`} alt="" loading="lazy" />{:else}<div class="cover-placeholder {item.kind}"><Icon name={item.kind} size={44}/><span>{item.kind === 'music' ? 'IMAGE & SON' : 'LE GOÛT DES HISTOIRES'}</span></div>{/if}
          <span class="cover-count">{item.entries.length} {item.kind === 'music' ? 'piste(s)' : item.kind === 'series' ? 'épisode(s)' : 'film'}</span>
        </div>
        <strong>{item.title}</strong><small>{item.subtitle}</small>
      </button>
    {/each}
  </div>
{/if}
