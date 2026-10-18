<script lang="ts">
 import {onMount} from 'svelte';import MediaShelf from './MediaShelf.svelte';import Icon from './Icon.svelte';
 import {recentGroups,unfinished,workCard,type Card} from './experience';import type {Entry,Kind,Work,Provider} from './types';
 export let playAlbum:(card:Card)=>void;export let view:string;export let entries:Entry[];export let works:Work[];export let providers:Provider[];export let kinds:Kind[];export let token:string;export let open:(card:Card)=>void;export let play:(e:Entry)=>void;export let mark:(e:Entry,v:boolean)=>void;export let settings:()=>void;
 let cinema:Work[]=[];let upcoming:Work[]=[];let series:Work[]=[];let musicUpcoming:Work[]=[];let sourceError='';let loading=false;let alive=true;const abort=new AbortController();
 $: resumed=unfinished(entries).filter(c=>view==='home'||c.kind===view);
 $: films=recentGroups(entries,'films');$: shows=recentGroups(entries,'series');$: albums=recentGroups(entries,'music');
 $: hero=(view==='music'?albums:view==='series'?shows:view==='films'?films:films.length?films:shows.length?shows:albums)[0];
 $: saved=works.filter(w=>view==='favorites'?w.favorite:w.watchlist).map(workCard);
 $: personal=works.filter(w=>w.kind===view&&w.favorite).map(workCard);
 $: later=works.filter(w=>w.kind===view&&w.watchlist).map(workCard);
 let mounted=false;let started=false;
 $: if(mounted&&!started&&providers.length){started=true;void shelves();}
 onMount(()=>{mounted=true;return()=>{alive=false;abort.abort();};});
 async function shelf(provider:string,kind:string,shelf:string,artist=''){const r=await fetch('/api/v1/shelves',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({provider,kind,shelf,artist}),signal:abort.signal});const data=await r.json();if(!r.ok)throw new Error(data.error);return data as Work[];}
 async function shelves(){if(view==='favorites'||view==='watchlist')return;loading=true;const errors:string[]=[];try{if(providers.some(p=>p.id==='tmdb'&&p.enabled&&p.configured)){if((view==='home'||view==='films')&&kinds.includes('films')){try{const value=await shelf('tmdb','films','now_playing');if(alive)cinema=value;}catch(e){if(alive)errors.push(e instanceof Error?e.message:'Sélection cinéma indisponible.');}try{const value=await shelf('tmdb','films','upcoming');if(alive)upcoming=value;}catch{}}if(view==='series'){try{const value=await shelf('tmdb','series','popular');if(alive)series=value;}catch(e){if(alive)errors.push(e instanceof Error?e.message:'Séries en ligne indisponibles.');}}}
 if((view==='home'||view==='music')&&kinds.includes('music')){const artists=[...new Set(works.filter(w=>w.kind==='music'&&w.metadata.provider==='musicbrainz'&&(w.localIds.length||w.favorite)).flatMap(w=>w.metadata.contributors?.map(a=>a.id)??[]))].slice(0,3);for(const artist of artists){try{const value=await shelf('musicbrainz','music','upcoming',artist);if(alive)musicUpcoming=[...new Map([...musicUpcoming,...value].map(w=>[w.id,w])).values()];}catch{}}}
 }finally{if(alive){sourceError=errors.join(' ');loading=false;}}}
</script>
{#if view==='watchlist'||view==='favorites'}
 <div class="page-heading"><span class="eyebrow">VOTRE SÉLECTION</span><h1>{view==='watchlist'?'À garder pour plus tard.':'Les histoires qui restent.'}</h1><p>{view==='watchlist'?'Films, séries et albums à retrouver, disponibles ou à découvrir.':'Vos favoris, réunis au même endroit.'}</p></div>
 {#if saved.length}<MediaShelf title={view==='watchlist'?'À voir / À écouter':'Favoris'} cards={saved} {open} grid/>{:else}<div class="quiet-empty"><Icon name="plus" size={28}/><h2>La prochaine envie commence par une fiche.</h2><p>Ouvrez un contenu et ajoutez-le à votre liste ou à vos favoris.</p></div>{/if}
{:else}
 <div class="page-heading compact"><div><span class="eyebrow">{view==='home'?'BIENVENUE CHEZ VOUS':view==='music'?'VOTRE BANDE-SON':view==='series'?'ENCORE UN ÉPISODE':'LE GOÛT DU CINÉMA'}</span><h1>{view==='home'?'Qu’est-ce qui vous tente ?':view==='music'?'Musique':view==='series'?'Séries':'Films'}</h1></div><span class="home-signature">Une collection. Toutes vos envies.</span></div>
 {#if resumed.length}<MediaShelf title="Reprendre le fil" subtitle="Là où vous vous êtes arrêté." cards={resumed} {open} {play} {mark}/>{/if}
 {#if hero}<section class="collection-spotlight" class:music-spotlight={hero.kind==='music'}><div class="spotlight-copy"><span class="eyebrow">{hero.kind==='music'?'UN ALBUM, UN MOMENT À SOI':'CE SOIR, DANS VOTRE COLLECTION'}</span><h2>{hero.title}</h2><p>{hero.subtitle}</p><div><button class="primary" onclick={()=>open(hero)}>Explorer {hero.kind==='music'?'l’album':'la fiche'} <Icon name="arrow" size={17}/></button></div><small>Déjà chez vous · prêt à {hero.kind==='music'?'écouter':'regarder'}</small></div><button class="spotlight-art" aria-label={`Ouvrir ${hero.title}`} onclick={()=>open(hero)}>{#if hero.image}<img src={hero.image} alt=""/>{:else}<Icon name={hero.kind} size={75}/>{/if}</button><span class="spotlight-word" aria-hidden="true">mosaune</span></section>{/if}
 {#if (view==='home'||view==='films')&&films.length}<MediaShelf title="Films dans votre collection" subtitle="Les derniers ajouts en premier." cards={films} {open}/>{/if}
 {#if cinema.length}<MediaShelf title="En ce moment au cinéma" subtitle="Sorties en Belgique · sélection TMDB. Gardez les titres qui vous tentent." cards={cinema.map(workCard)} {open}/>{/if}
 {#if (view==='home'||view==='series')&&shows.length}<MediaShelf title="Vos séries, épisode après épisode" subtitle="Les derniers ajouts en premier." cards={shows} {open}/>{/if}
 {#if (view==='home'||view==='music')&&albums.length}<MediaShelf title="À portée d’oreille" subtitle="Les albums de votre collection." cards={albums} {open} {playAlbum}/>{/if}
 {#if personal.length}<MediaShelf title={view==='music'?'Vos disques de cœur':'Vos valeurs sûres'} subtitle="Les œuvres que vous avez ajoutées aux favoris." cards={personal} {open} {playAlbum}/>{/if}
 {#if later.length}<MediaShelf title={view==='music'?'Pour une prochaine écoute':'Vos prochaines séances'} subtitle="Votre liste, à retrouver sans chercher." cards={later} {open} {playAlbum}/>{/if}
 {#if upcoming.length}<MediaShelf title="Bientôt sur grand écran" subtitle="Dates annoncées par TMDB · elles peuvent évoluer." cards={upcoming.map(workCard)} {open}/>{/if}
 {#if series.length}<MediaShelf title="Les séries du moment" subtitle="Popularité TMDB, pas encore un classement personnalisé." cards={series.map(workCard)} {open}/>{/if}
 {#if musicUpcoming.length}<MediaShelf title="À l’horizon, côté musique" subtitle="Éditions annoncées pour les artistes de vos albums identifiés ou favoris · MusicBrainz." cards={musicUpcoming.map(workCard)} {open}/>{/if}
 {#if view==='music'&&!musicUpcoming.length}<div class="editorial-note"><span class="eyebrow">LE PROCHAIN DISQUE</span><h3>De la place pour vos prochaines découvertes.</h3><p>Associez un album à son édition MusicBrainz depuis sa fiche. Les sorties annoncées de ses artistes apparaîtront ici lorsqu’elles sont renseignées.</p></div>{/if}
 {#if !hero&&!resumed.length}<div class="quiet-empty"><Icon name={view==='home'?'plus':view} size={32}/><h2>Votre collection commence ici.</h2><p>Ajoutez un dossier depuis Réglages, ou cherchez une première œuvre en haut de la page.</p><button class="primary" onclick={settings}>Ajouter mes dossiers</button></div>{/if}
 {#if loading}<p class="folder-help" role="status">Les sélections en ligne arrivent…</p>{/if}{#if sourceError}<p class="folder-help" role="status">{sourceError} Votre collection locale reste disponible.</p>{/if}
{/if}
