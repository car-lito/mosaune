<script lang="ts">
  import { onMount } from 'svelte';
  import type { Provider } from './types';
  export let token: string;
  export let refresh: () => Promise<void>;
  export let providers: Provider[] = [];
  export let onlyProvider=''; export let expanded=false;
  let secrets: Record<string,string> = {};
  let busy = '';
  let message = '';
  let error = '';
  async function load() {
    try { const r=await fetch('/api/v1/metadata/providers'); if(!r.ok) throw new Error('Impossible de charger les fournisseurs.'); providers=await r.json(); }
    catch(e) { error=e instanceof Error ? e.message : 'Serveur indisponible.'; }
  }
  onMount(() => { void load();const reload=()=>{void load()};window.addEventListener('mosaune-provider-settings',reload);return()=>window.removeEventListener('mosaune-provider-settings',reload); });
  async function action(url: string, method: string, body?: unknown) {
    if(busy) return; busy=url; error=''; message='';
    try {
      const response=await fetch(url,{method,headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:body ? JSON.stringify(body) : undefined});
      const data=response.status===204 ? {} : await response.json(); if(!response.ok) throw new Error(data.error ?? 'Action impossible.');
      secrets={}; await load(); await refresh(); message=data.message ?? (method==='DELETE' ? 'Fournisseur déconnecté. Les fiches déjà enregistrées restent disponibles.' : 'Clé enregistrée. L’identification démarre automatiquement si elle n’est pas en pause.');
    } catch(e) { error=e instanceof Error ? e.message : 'Serveur indisponible.'; }
    finally { busy=''; }
  }
</script>

<details class="metadata-settings" open={expanded}>
  <summary>{onlyProvider?'Connexion au fournisseur':'Affiches et fournisseurs'} <span>{providers.some(p=>(!onlyProvider||p.id===onlyProvider)&&p.configured) ? 'Configuré' : 'À configurer'}</span></summary>
  <p>{onlyProvider?'Retrouvez ici la connexion utilisée par ce fournisseur.':'Les tags musicaux sont lus sur place. Pour les films et séries, un fournisseur peut compléter les titres avec des affiches, résumés et acteurs.'}</p>
  {#each providers.filter(p=>!onlyProvider||p.id===onlyProvider) as provider}
    <div class="provider-config"><h3>{provider.name} <small>{provider.configured ? 'Clé enregistrée' : 'Non connecté'}</small></h3>
      {#if provider.id==='tmdb'}
        <ol><li>Créez un compte sur <a href="https://www.themoviedb.org" target="_blank" rel="noreferrer">TMDB</a>, puis ouvrez <a href="https://www.themoviedb.org/settings/api" target="_blank" rel="noreferrer">Paramètres → API</a>.</li><li>Demandez un accès développeur pour votre application personnelle Mosaune. Copiez le <strong>API Read Access Token</strong> (ou la clé API v3).</li><li>Collez-le ci-dessous. Il sera conservé dans le catalogue local, jamais renvoyé dans les fiches du navigateur.</li></ol>
      {/if}
      {#if provider.requiresCredential}<form class="provider-form" onsubmit={e=>{e.preventDefault();void action(`/api/v1/metadata/providers/${provider.id}`,'PUT',{secret:secrets[provider.id]});}}>
        <label>Jeton de lecture ou clé API<input type="password" autocomplete="off" required bind:value={secrets[provider.id]} placeholder={provider.configured ? 'Saisir une nouvelle clé pour la remplacer' : 'Votre clé personnelle'} disabled={!!busy}/></label>
        <button class="secondary" disabled={!!busy}>Enregistrer</button>
        {#if provider.configured}<button type="button" class="back-link" disabled={!!busy} onclick={()=>action(`/api/v1/metadata/providers/${provider.id}`,'DELETE')}>Déconnecter</button>{/if}
      </form>{:else}<p>Ce fournisseur fonctionne sans clé.</p>{/if}
    </div>
  {/each}
  <p>Une fois connecté, l’identification accompagne l’actualisation des dossiers. Les titres recherchés et les identifiants des œuvres servant aux suggestions sont envoyés au fournisseur ; les médias et vos listes restent chez vous.</p>
  <p class="folder-help">Le suivi « Affiches et fiches » permet de mettre l’identification en pause ou de relancer les erreurs. Pour une correspondance incertaine, ouvrez le média et choisissez le bon résultat.</p>
  {#if message}<p class="notice" role="status">{message}</p>{/if}{#if error}<p class="playback-error" role="alert">{error}</p>{/if}
</details>
