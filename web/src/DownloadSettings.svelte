<script lang="ts">
 import {onMount} from 'svelte';import DownloadConnector from './DownloadConnector.svelte';import {acquisition,type DownloadConfig} from './acquisition';
 export let token:string;export let libraries:{id:string;name:string;kind:string;path?:string}[];export let refresh:()=>Promise<void>;
 let configs:DownloadConfig[]=[];let error='';let alive=true;
 async function load(){try{const value=await acquisition<DownloadConfig[]>(token,'settings');if(alive){configs=value;error='';}}catch(e){if(alive)error=e instanceof Error?e.message:'Réglages indisponibles.';}}
 onMount(()=>{void load();return()=>{alive=false;};});
</script>
<section class="download-settings"><span class="eyebrow">VOS ENVIES PRENNENT VIE</span><h2>Téléchargements</h2><p>Reliez votre installation Radarr et Sonarr pour demander un film, une série ou une saison depuis sa fiche.</p>
 <details class="download-guide"><summary>Première configuration · le guide</summary><ol><li>Ouvrez Radarr ou Sonarr sur votre NAS. Dans <strong>Settings → General → Security</strong>, copiez la clé API.</li><li>Indiquez ci-dessous l’adresse accessible depuis l’ordinateur qui exécute Mosaune : l’adresse de votre NAS et le port du service.</li><li>Collez la clé puis cliquez sur <strong>Tester la connexion</strong>. Ce test ne lance aucun téléchargement.</li><li>Choisissez le profil qualité et le dossier déjà configurés dans le service.</li><li>Associez le dossier Mosaune contenant les mêmes fichiers. Si la liste est vide, utilisez « Ajouter un dossier » juste sous celle-ci, puis enregistrez.</li></ol><p>Gardez vos clés dans ces champs : inutile de les partager dans le chat. Elles sont enregistrées dans le catalogue local de Mosaune. Votre installation Radarr/Sonarr doit déjà fonctionner avec ses indexeurs et son client de téléchargement.</p><p>Une entrée dans « Ma liste » ne déclenche aucun téléchargement. L’envoi nécessite le bouton « Demander », puis sa confirmation. Les demandes musicales arriveront avec un autre connecteur.</p></details>
 {#if error}<p role="alert">{error} <button class="text-action" onclick={load}>Réessayer</button></p>{/if}
 {#each configs as config (config.id)}<DownloadConnector initial={config} {token} {libraries} {refresh}/>{/each}
</section>
