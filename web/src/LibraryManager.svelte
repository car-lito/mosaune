<script lang="ts">
  import ScanProgress from './ScanProgress.svelte';
  export let libraries: { id: string; name: string; path: string; kind: string; count: number }[];
  export let kinds: string[];
  export let token: string;
  export let refresh: () => Promise<void>;
  let expanded = false;
  let folder = '';
  let name = '';
  let kind = '';
  let busy = '';
  let message = '';
  let failure = '';
  const labels: Record<string,string> = { films:'Films', series:'Séries', music:'Musique' };
  $: if (!kinds.includes(kind)) kind = kinds[0] ?? '';
  async function request(id?: string) {
    if (busy) return;
    busy = id ?? 'add'; message = ''; failure = '';
    try {
      const response = await fetch(id ? `/api/v1/libraries/${id}/scan` : '/api/v1/libraries', {
        method: 'POST', headers: { 'Content-Type':'application/json','X-Mosaune-Token':token },
        body: id ? undefined : JSON.stringify({ name, path:folder, kind }),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error ?? 'Analyse impossible.');
      await refresh();
      message = `${result.name} : ${result.count} fichier${result.count > 1 ? 's' : ''} dans la bibliothèque. ${result.metadataMessage ?? ''}`;
      if (!id) { folder = ''; name = ''; expanded = false; }
    } catch (e) { failure = e instanceof Error ? e.message : 'Le serveur ne répond pas.'; }
    finally { busy = ''; }
  }
</script>

<section class="library-manager" aria-labelledby="folders-title">
  <div class="section-heading"><div><h2 id="folders-title">Vos dossiers</h2><p class="folder-help">Ajoutez un dossier de cet ordinateur. Mosaune retient son contenu, sans le copier ni le modifier.</p></div><button class="secondary" disabled={!kinds.length || !!busy} onclick={() => expanded = !expanded}>{expanded ? 'Fermer' : 'Ajouter un dossier'}</button></div>
  {#if expanded}
    <form class="folder-form" onsubmit={(e) => { e.preventDefault(); void request(); }}>
      <label class="folder-path">Chemin complet du dossier<input required bind:value={folder} placeholder="C:\Users\votre-nom\Music" disabled={!!busy} /><small>Dans l’Explorateur Windows, ouvrez le dossier puis copiez son adresse avec Ctrl+L, Ctrl+C.</small></label>
      <label>Contenu<select bind:value={kind} disabled={!!busy}>{#each kinds as k}<option value={k}>{labels[k]}</option>{/each}</select></label>
      <label>Nom (facultatif)<input bind:value={name} maxlength="150" placeholder="Mes albums" disabled={!!busy} /></label>
      <button class="primary" disabled={!!busy}>{busy === 'add' ? 'Analyse en cours…' : 'Ajouter et analyser'}</button>
    </form>
  {/if}
  {#each libraries as library}
    <div class="folder-row"><div><strong>{library.name}</strong><span>{labels[library.kind]} · {library.count} fichier{library.count > 1 ? 's' : ''}</span><small>{library.path}</small></div><button class="secondary" disabled={!!busy} onclick={() => request(library.id)}>{busy === library.id ? 'Analyse…' : 'Actualiser'}</button></div>
  {/each}
  <ScanProgress/>
  {#if busy}<p class="folder-help" role="status">Gardez cet onglet ouvert jusqu’à la fin de l’analyse des fichiers. L’identification des fiches continuera ensuite en arrière-plan.</p>{/if}
  {#if message}<p class="notice" role="status">{message}</p>{/if}
  {#if failure}<p class="playback-error" role="alert">{failure}</p>{/if}
</section>
