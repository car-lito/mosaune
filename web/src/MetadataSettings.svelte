<script lang="ts">
  import { onMount } from 'svelte';
  import type { Provider } from './types';
  import { providerSettingsView } from './provider-management';
  import ExtensionConfiguration from './ExtensionConfiguration.svelte';
  export let token: string;
  export let refresh: () => Promise<void>;
  export let providers: Provider[] = [];
  export let onlyProvider=''; export let expanded=false;
  let error = '';
  async function load() {
    try { const r=await fetch('/api/v1/metadata/providers'); if(!r.ok) throw new Error('Impossible de charger les fournisseurs.'); providers=await r.json(); error=''; }
    catch(e) { error=e instanceof Error ? e.message : 'Serveur indisponible.'; }
  }
  onMount(() => { void load();const reload=()=>{void load()};window.addEventListener('mosaune-provider-settings',reload);return()=>window.removeEventListener('mosaune-provider-settings',reload); });
  // Each provider declares its own form (fields, guide, links) in its settings
  // schema. This screen only lists providers and renders what they declare.
  async function changed() { await load(); await refresh(); }
</script>

<details class="metadata-settings" open={expanded}>
  <summary>{onlyProvider?'Connexion au fournisseur':'Affiches et fournisseurs'} <span>{providers.some(p=>(!onlyProvider||p.id===onlyProvider)&&p.configured) ? 'Configuré' : 'À configurer'}</span></summary>
  <p>{onlyProvider?'Retrouvez ici la connexion utilisée par ce fournisseur.':'Les tags musicaux sont lus sur place. Les fournisseurs installés peuvent compléter vos médias avec des affiches, résumés et crédits.'}</p>
  {#each providers.filter(p=>!onlyProvider||p.id===onlyProvider) as provider (provider.id)}
    {@const view=providerSettingsView(provider)}
    <div class="provider-config"><h3>{provider.name} {#if !provider.enabled}<small>Désactivé · activation dans Extensions</small>{/if}<small>{provider.configured ? (provider.requiresCredential ? 'Clé enregistrée' : 'Prêt') : 'Non connecté'}</small></h3>
      {#if view.kind==='declared'}<ExtensionConfiguration id={view.owner} {token} refresh={changed}/>
      {:else if view.kind==='missing'}<p class="form-error">Ce fournisseur demande une connexion mais ne déclare pas de formulaire de réglages. Vérifiez son paquet dans Extensions.</p>
      {:else}<p>Ce fournisseur fonctionne sans clé.</p>{/if}
    </div>
  {/each}
  <p>Une fois connecté, l’identification accompagne l’actualisation des dossiers. Les titres recherchés et les identifiants des œuvres servant aux suggestions sont envoyés au fournisseur ; les médias et vos listes restent chez vous.</p>
  <p class="folder-help">Le suivi « Affiches et fiches » permet de mettre l’identification en pause ou de relancer les erreurs. Pour une correspondance incertaine, ouvrez le média et choisissez le bon résultat.</p>
  {#if error}<p class="playback-error" role="alert">{error}</p>{/if}
</details>
