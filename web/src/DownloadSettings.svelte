<script lang="ts">
 import {onMount} from 'svelte';import DownloadConnector from './DownloadConnector.svelte';import {acquisition,type DownloadConfig} from './acquisition';
 export let token:string;export let libraries:{id:string;name:string;kind:string;path?:string}[];export let refresh:()=>Promise<void>;
 let configs:DownloadConfig[]=[];let error='';let alive=true;
 async function load(){try{const value=await acquisition<DownloadConfig[]>(token,'settings');if(alive){configs=value;error='';}}catch(e){if(alive)error=e instanceof Error?e.message:'Réglages indisponibles.';}}
 onMount(()=>{void load();return()=>{alive=false;};});
</script>
<section class="download-settings"><span class="eyebrow">VOS ENVIES PRENNENT VIE</span><h2>Téléchargements</h2><p>Reliez votre installation Radarr et Sonarr pour demander un film, une série ou une saison depuis sa fiche.</p>
 <p class="folder-help">Une entrée dans « Ma liste » ne déclenche aucun téléchargement. Chaque envoi demande votre confirmation. Les clés restent dans le catalogue du serveur Mosaune.</p>
 {#if error}<p role="alert">{error} <button class="text-action" onclick={load}>Réessayer</button></p>{/if}
 {#each configs as config (config.id)}<DownloadConnector initial={config} {token} {libraries} {refresh}/>{/each}
</section>
