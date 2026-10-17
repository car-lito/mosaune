<script lang="ts">
 import {accountRequest} from './accounts';
 export let token:string;export let state:{package:{manifest:{module:{version:string}};checksum:string}|null;previousVersion:string;previousChecksum:string}|null;
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
 <div class="extension-actions"><button class="secondary" disabled={busy} onclick={browse}>{state?.package?'Installer une mise à jour':'Voir les paquets disponibles'}</button>{#if state?.previousVersion}<button class="text-action" disabled={busy} onclick={()=>confirm='rollback'}>Restaurer la version {state.previousVersion}</button>{/if}{#if state?.package}<button class="text-action" disabled={busy} onclick={()=>confirm='remove'}>Désinstaller le fournisseur</button>{/if}</div>
 {#if confirm}<div class="extension-warning"><strong>{confirm==='rollback'?'Restaurer la version précédente ?':'Désinstaller ce paquet ?'}</strong><p>{confirm==='rollback'?'Elle sera vérifiée avant remplacement. La version actuelle deviendra la version de retour.':'Le paquet et sa version de retour seront supprimés. Le module sera désactivé et le fournisseur intégré prendra le relais. La clé reste partagée avec celui-ci ; vos médias, affiches et corrections sont conservés.'}</p><div class="extension-actions"><button class="secondary" disabled={busy} onclick={act}>{busy?'Vérification…':confirm==='rollback'?'Confirmer la restauration':'Confirmer la désinstallation'}</button><button class="text-action" disabled={busy} onclick={()=>confirm=''}>Annuler</button></div></div>{/if}
 {#if message}<p class="extension-notice" role="status">{message}</p>{/if}{#if error}<p class="form-error" role="alert">{error}</p>{/if}
</section>
