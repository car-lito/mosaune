<script lang="ts">
  import { onMount, tick } from 'svelte';
  import WorkPoster from './WorkPoster.svelte';
  import WorkDetails from './WorkDetails.svelte';
  import type { Entry, Kind, Provider, Work } from './types';
  export let token: string;
  export let providers: Provider[];
  export let kinds: Kind[];
  export let entries: Entry[];
  export let focusMedia = '';
  export let play: (entry: Entry) => void;
  type Tab = 'suggestions' | 'search' | 'watchlist' | 'favorites';
  let tab: Tab = 'suggestions';
  const tabs: { id: Tab; title: string }[] = [{id:'suggestions',title:'Pour vous'},{id:'search',title:'Rechercher'},{id:'watchlist',title:'À voir'},{id:'favorites',title:'Favoris'}];
  let library: Work[] = [];
  let results: Work[] = [];
  let selected: Work | null = null;
  let kind = kinds.includes('films') ? 'films' : 'series';
  let provider = '';
  let query = '';
  let seed = '';
  let busy = true;
  let error = '';
  let reason = '';
  let searched = false;
  let page = 1;
  let pages = 1;
  let alive = true;
  let searchSpec: { provider: string; kind: string; query: string; seed: string } | null = null;
  const controller = new AbortController();
  $: compatible = providers.filter(p => p.enabled && p.discovery && p.kinds.includes(kind as Kind));
  $: if (!compatible.some(p => p.id === provider)) provider = compatible[0]?.id ?? '';
  $: configured = compatible.find(p => p.id === provider)?.configured;
  $: seeds = library.filter(w => w.kind === kind && w.metadata.provider === provider).sort((a,b) => Number(b.favorite) - Number(a.favorite) || Number(b.watched) - Number(a.watched));
  $: if (!seeds.some(w => w.id === seed)) seed = seeds[0]?.id ?? '';
  $: shown = tab === 'watchlist' ? library.filter(w => w.watchlist) : tab === 'favorites' ? library.filter(w => w.favorite) : searched ? results : seeds;
  onMount(() => { void load(); return () => { alive = false; controller.abort(); }; });
  async function load() {
    busy = true; error = '';
    try {
      const response = await fetch('/api/v1/discovery/library', { signal: controller.signal });
      const data = await response.json(); if (!response.ok) throw new Error(data.error);
      if (!alive) return;
      library = data;
      if (focusMedia) selected = library.find(w => w.localIds.includes(focusMedia)) ?? null;
      const first = [...library].sort((a,b) => Number(b.favorite) - Number(a.favorite) || Number(b.watched) - Number(a.watched)).find(w => providers.some(p => p.id === w.metadata.provider && p.enabled && p.discovery));
      if (first) {
        kind = first.kind; provider = first.metadata.provider; seed = first.id;
        await tick();
        if (!focusMedia && (first.favorite || first.watched) && configured && alive) { busy = false; await browse(); }
      }
    } catch(e) { if (alive) error = e instanceof Error ? e.message : 'Listes indisponibles.'; }
    finally { if (alive) busy = false; }
  }
  function changeTab(next: Tab) { tab = next; reset(); }
  function reset() { searched = false; results = []; error = ''; reason = ''; page = 1; pages = 1; searchSpec = null; }
  async function browse(nextPage = 1, reuse = false) {
    if (busy) return;
    const spec = reuse && searchSpec ? searchSpec : { provider, kind, query: tab === 'search' ? query.trim() : '', seed: tab === 'suggestions' ? seed : '' };
    busy = true; error = ''; searchSpec = spec;
    try {
      const response = await fetch('/api/v1/discovery/browse', { method: 'POST', headers: { 'Content-Type':'application/json', 'X-Mosaune-Token':token }, body:JSON.stringify({...spec,page:nextPage}), signal:controller.signal });
      const data = await response.json(); if (!response.ok) throw new Error(data.error);
      if (!alive) return;
      results = data.items; page = data.page; pages = data.pages; reason = data.reason; searched = true; searchSpec = spec;
    } catch(e) { if (alive) error = e instanceof Error ? e.message : 'Recherche indisponible.'; }
    finally { if (alive) busy = false; }
  }
  function changed(work: Work) {
    library = library.some(w => w.id === work.id) ? library.map(w => w.id === work.id ? work : w) : [...library, work];
    results = results.map(w => w.id === work.id ? work : w);
  }
  function recommend(work: Work) {
    changed(work); selected = null; kind = work.kind; provider = work.metadata.provider; seed = work.id; tab = 'suggestions'; reset();
    void browse();
  }
</script>

<section class="discovery-section" aria-labelledby="discovery-title">
  <span class="eyebrow">LA CURIOSITÉ A SA PLACE</span><h1 id="discovery-title">La prochaine belle découverte.</h1><p class="discovery-lead">Les histoires que vous aimez, celles qui vous attendent.</p>
  <nav class="discovery-tabs" aria-label="Découverte et listes">{#each tabs as item}<button class:chosen={tab === item.id} aria-current={tab === item.id ? 'page' : undefined} disabled={busy} onclick={() => changeTab(item.id)}>{item.title}{item.id === 'watchlist' ? ` · ${library.filter(w => w.watchlist).length}` : item.id === 'favorites' ? ` · ${library.filter(w => w.favorite).length}` : ''}</button>{/each}</nav>
  {#if tab === 'suggestions' || tab === 'search'}
    <form class="discovery-form" onsubmit={event => { event.preventDefault(); void browse(); }}>
      <div class="discovery-filters"><label>Univers<select bind:value={kind} disabled={busy} onchange={reset}>{#if kinds.includes('films')}<option value="films">Films</option>{/if}{#if kinds.includes('series')}<option value="series">Séries</option>{/if}</select></label><label>Fournisseur<select bind:value={provider} disabled={busy} onchange={reset}>{#each compatible as p}<option value={p.id}>{p.name}</option>{/each}</select></label></div>
      {#if tab === 'search'}<label class="discovery-query">Un titre, même absent de votre serveur<input type="search" bind:value={query} minlength="2" maxlength="250" placeholder="Iron Man, Avengers…" required disabled={busy}/></label>
      {:else}<label class="discovery-query">Votre point de départ<select bind:value={seed} disabled={busy || !seeds.length} onchange={reset}>{#each seeds as work}<option value={work.id}>{work.metadata.title}{work.favorite ? ' · Favori' : work.watched ? ' · Déjà regardé' : ' · Votre collection / sélection'}</option>{/each}</select></label>{/if}
      <button class="primary" disabled={busy || !configured || (tab === 'suggestions' && !seed)}>{busy ? 'Patientez…' : tab === 'search' ? 'Rechercher' : 'Trouver des suggestions'}</button>
    </form>
    {#if !compatible.length}<p class="notice">Aucun fournisseur de découverte pour cet univers.</p>{:else if !configured}<p class="notice">Configurez {compatible.find(p => p.id === provider)?.name} dans « Affiches et fournisseurs » ci-dessus. Vos listes enregistrées restent accessibles.</p>{/if}
    {#if tab === 'suggestions' && !seeds.length && !busy}<div class="empty-state compact"><h3>Commençons par une histoire que vous aimez.</h3><p>Recherchez un film ou une série, ouvrez sa fiche et ajoutez-le aux favoris. Une œuvre locale identifiée peut aussi servir de point de départ.</p><button class="secondary" onclick={() => changeTab('search')}>Chercher un premier titre</button></div>{/if}
  {/if}
  {#if error}<p class="playback-error" role="alert">{error}</p><button class="secondary" disabled={busy} onclick={() => searchSpec ? browse(page, true) : load()}>Réessayer</button>{/if}
  {#if busy}<p class="folder-help" role="status">Chargement en cours…</p>{/if}
  {#if reason && (tab === 'search' || tab === 'suggestions')}<h2 class="discovery-reason">{reason}</h2><p class="folder-help">Suggestions et résultats fournis par {providers.find(p => p.id === searchSpec?.provider)?.name ?? provider}. Vos favoris restent sur votre serveur.</p>
  {:else if tab === 'suggestions' && seeds.length}<h2 class="discovery-reason">Vos points de départ</h2>{/if}
  {#if !busy && !shown.length && (searched || tab === 'watchlist' || tab === 'favorites')}<div class="empty-state compact"><h3>{tab === 'watchlist' ? 'Les prochaines soirées commencent ici.' : tab === 'favorites' ? 'Quelles histoires vous ressemblent ?' : 'Aucun résultat pour cette recherche.'}</h3><p>{tab === 'watchlist' || tab === 'favorites' ? 'Ouvrez une fiche pour ajouter une œuvre à cette liste.' : 'Essayez un autre titre ou un autre point de départ.'}</p></div>{/if}
  {#if tab !== 'search' || searched}<div class="work-grid">{#each shown as work (work.id)}<button class="work-card" disabled={busy} onclick={() => selected = work}><WorkPoster {work}/><span class="work-caption"><strong>{work.metadata.title}</strong><small>{work.kind === 'films' ? 'Film' : 'Série'}{work.metadata.year ? ` · ${work.metadata.year}` : ''}{work.favorite ? ' · ♥' : ''}{work.watchlist ? ' · À voir' : ''}</small><span class="availability" class:available={work.localIds.length > 0}>{work.localIds.length ? work.kind === 'series' ? `${work.localIds.length} fichier(s) d’épisode` : 'Dans votre bibliothèque' : 'À découvrir'}</span></span></button>{/each}</div>{/if}
  {#if searched && (tab === 'suggestions' || tab === 'search')}<div class="discovery-pagination"><button class="secondary" disabled={busy || page <= 1} onclick={() => browse(page - 1, true)}>Précédent</button><span>Page {page} sur {pages}</span><button class="secondary" disabled={busy || page >= pages} onclick={() => browse(page + 1, true)}>Suivant</button></div>{/if}
</section>
{#if selected}{#key selected.id}<WorkDetails work={selected} {token} {entries} {changed} {recommend} {play} close={() => selected = null}/>{/key}{/if}
