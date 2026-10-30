<script lang="ts">
 import {accountRequest} from './accounts';
 import type {ProviderPackageState} from './extensions';
 export let token:string;export let state:ProviderPackageState|null;
 export let changed:()=>Promise<void>;export let browse:()=>void;
 let busy=false;let error='';let message='';let confirm:''|'rollback'|'remove'='';
 async function act(){if(busy||!state?.package||!confirm)return;busy=true;error='';message='';const action=confirm;try{
  await accountRequest(token,'provider-packages/tmdb'+(action==='rollback'?'/rollback':''),action==='rollback'?'POST':'DELETE',{checksum:state.package.checksum,...(action==='rollback'?{previousChecksum:state.previousChecksum}:{})});
  confirm='';await changed();message=action==='rollback'?'Version précédente restaurée.':'Paquet désinstallé. Le fournisseur intégré est disponible ; vos fiches et votre connexion sont conservées.';
 }catch(e){error=e instanceof Error?e.message:'Action impossible.'}finally{busy=false}}
</script>
<section class="extension-warning provider-package-panel" aria-busy={busy}>
 <h3>{state?.package?'Paquet installé · '+state.package.manifest.module.version:'Module fourni avec Mosaune'}</h3>
 <p>{state?.package?'Ce programme provient du paquet importé. Il peut être actualisé sans remplacer Mosaune.':'Vous pouvez maintenant installer le fournisseur TMDB depuis un paquet indépendant.'}</p>
 <p>{state?.package&&!state.package.manifest.module.capabilities.includes('metadata.video-catalog.v1')?'Ce paquet ne prend pas en charge la découverte, les profils et les affiches alternatives. Installez TMDB 0.5.0 ou supérieur pour utiliser ces fonctions avec le moteur extension.':'Avec le moteur extension choisi : recherche, fiches, suggestions, sélections, profils et affiches alternatives sont traités par le module.'}</p>
 {#if state?.registrationWarning}<p class="form-error" role="alert">{state.registrationWarning}</p>{/if}
 {#if state?.registration}<details class="extension-import"><summary>Dépendances et démarrage</summary>{#if state.registration.dependencies.length}<ul>{#each state.registration.dependencies as dependency}<li>{dependency} · {state.registration.choices[dependency]??'À choisir'}</li>{/each}</ul>{:else}<p>Ce paquet ne nécessite aucune autre extension.</p>{/if}<p>{state.registration.wanted?'Le moteur extension est choisi. Il démarrera à la demande si TMDB est actif.':'Le paquet est conservé, mais le moteur intégré est choisi.'} Ce choix est conservé après un redémarrage.</p></details>{/if}
 <div class="extension-actions"><button class="secondary" disabled={busy} onclick={browse}>{state?.package?'Installer une mise à jour':'Voir les paquets disponibles'}</button>{#if state?.previousVersion}<button class="text-action" disabled={busy} onclick={()=>confirm='rollback'}>Restaurer la version {state.previousVersion}</button>{/if}{#if state?.package}<button class="text-action" disabled={busy} onclick={()=>confirm='remove'}>Désinstaller le paquet</button>{/if}</div>
 {#if confirm}<div class="extension-warning"><strong>{confirm==='rollback'?'Restaurer la version précédente ?':'Désinstaller ce paquet ?'}</strong><p>{confirm==='rollback'?'Elle sera vérifiée avant remplacement. La version actuelle deviendra la version de retour.':'Le paquet et sa version de retour seront supprimés. Le moteur intégré sera choisi ; TMDB restera désactivé si vous l’avez coupé. La clé reste partagée avec celui-ci ; vos médias, affiches et corrections sont conservés.'}</p><div class="extension-actions"><button class="secondary" disabled={busy} onclick={act}>{busy?'Vérification…':confirm==='rollback'?'Confirmer la restauration':'Confirmer la désinstallation'}</button><button class="text-action" disabled={busy} onclick={()=>confirm=''}>Annuler</button></div></div>{/if}
 {#if message}<p class="extension-notice" role="status">{message}</p>{/if}{#if error}<p class="form-error" role="alert">{error}</p>{/if}
</section>
