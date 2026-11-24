<script lang="ts">
 import {onMount} from 'svelte';
 import {accountRequest} from './accounts';
 import {providerRoute,providerPackageRoute,providerStatus,type ProviderManagement} from './provider-management';
 import type {ProviderPackageState} from './extensions';
 import ProviderPackagePanel from './ProviderPackagePanel.svelte';
 import ExtensionConfiguration from './ExtensionConfiguration.svelte';
 export let provider:ProviderManagement;export let token:string;
 export let changed:()=>Promise<void>;export let browse:()=>void;
 let state:ProviderPackageState|null=null;let loading=true;let busy=false;let error='';let mounted=false;let generation=0;
 async function load(snapshot=provider){const request=++generation;loading=true;error='';try{const next=snapshot.packages?await accountRequest(token,providerPackageRoute(snapshot.id)):null;if(request===generation)state=next;}catch(e){if(request===generation)error=e instanceof Error?e.message:'Paquet indisponible.';}finally{if(request===generation)loading=false;}}
 onMount(()=>{mounted=true;return()=>{mounted=false;generation++;}});
 $: if(mounted&&provider)void load(provider);
 async function reload(){await changed();}
 async function toggle(action:'state'|'runtime'){
  if(busy)return;busy=true;error='';
  try{await accountRequest(token,providerRoute(provider.id,action),'PUT',{enabled:action==='state'?!provider.enabled:!provider.runtime?.enabled});await reload();}
  catch(e){error=e instanceof Error?e.message:'Réglage non enregistré.';}finally{busy=false;}
 }
</script>
<section aria-busy={busy||loading}>
 <p>Ce fournisseur enrichit les médias pris en charge par ses extensions. Ses accès et réglages sont gérés ci-dessous.</p>
 <div class="extension-native-choice"><div><strong>{provider.name} · {providerStatus(provider)}</strong><p>Les informations déjà enregistrées restent disponibles lorsque le fournisseur est arrêté.</p></div>
 {#if provider.activation}<button class="secondary" role="switch" aria-label={'Activer '+provider.name} aria-checked={provider.enabled} disabled={busy||provider.runtime?.installed===false} onclick={()=>toggle('state')}>{provider.enabled?'Désactiver':'Activer'}</button>{/if}</div>
 {#if provider.runtimeControl}<details class="extension-import"><summary>Module de traitement</summary><p>La pause du module conserve les réglages du fournisseur et ses autorisations.</p><button class="secondary" disabled={busy||provider.runtime?.installed===false||provider.runtime?.safeMode&&!provider.runtime?.enabled} onclick={()=>toggle('runtime')}>{provider.runtime?.enabled?'Mettre le module en pause':'Activer le module installé'}</button></details>{/if}
 {#if loading}<p role="status">Chargement du fournisseur…</p>{:else if !error}
 {#if provider.packages}<ProviderPackagePanel {token} {state} providerId={provider.id} providerName={provider.name} changed={reload} {browse}/>{/if}
 {#if provider.configurationOwner&&(!provider.packages||state?.package)}{#key state?.package?.checksum}<ExtensionConfiguration id={provider.configurationOwner} {token} refresh={reload}/>{/key}
 {:else if !provider.packages}<p>Ce fournisseur ne déclare pas de formulaire supplémentaire. Ses composants se gèrent dans les réglages de la famille média correspondante.</p>{/if}
 {/if}
 {#if error}<p class="form-error" role="alert">{error}</p><button class="secondary" disabled={busy} onclick={()=>load()}>Actualiser le fournisseur</button>{/if}
</section>
