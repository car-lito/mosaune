<script lang="ts">
 import {onMount} from 'svelte';import MediaShelf from './MediaShelf.svelte';import Icon from './Icon.svelte';
 import {feedCards,type UIBlock} from './ui-pages';import {workCard,type Card} from './experience';import type {Entry,Work,Kind,Provider} from './types';
 export let block:UIBlock;export let entries:Entry[];export let works:Work[];export let kinds:Kind[];export let providers:Provider[];export let token:string;
 export let open:(c:Card)=>void;export let playAlbum:(c:Card)=>void;export let play:((e:Entry)=>void)|undefined;export let mark:((e:Entry,v:boolean)=>void)|undefined;
 let remote:Work[]=[];let error='';let loading=false;let controller:AbortController|null=null;let alive=true;let requestKey='';
 $: provider=providers.find(p=>p.id===block.provider&&p.enabled&&p.configured&&p.kinds.includes(block.view as Kind));
 $: artists=block.provider==='musicbrainz'?[...new Set(works.filter(w=>w.kind==='music'&&w.metadata.provider===block.provider&&(w.localIds.length||w.favorite)).flatMap(w=>w.metadata.contributors?.map(a=>a.id)??[]))].slice(0,3):[];
 $: key=JSON.stringify([block.source,block.provider,block.view,block.shelf,!!provider,kinds.includes(block.view as Kind),artists]);
 $: if(key!==requestKey){requestKey=key;void load();}
 $: cards=block.source==='remote'?remote.filter(w=>kinds.includes(w.kind)&&w.kind===block.view).map(workCard):feedCards(block,entries,works,kinds);
 async function load(){controller?.abort();const current=new AbortController();controller=current;remote=[];error='';loading=false;if(block.source!=='remote'||!provider||!kinds.includes(block.view as Kind))return;
 loading=true;const fetched:Work[]=[];try{for(const artist of block.provider==='musicbrainz'?artists:['']){const r=await fetch('/api/v1/shelves',{method:'POST',headers:{'Content-Type':'application/json','X-Mosaune-Token':token},body:JSON.stringify({provider:block.provider,kind:block.view,shelf:block.shelf,artist}),signal:current.signal});const data=await r.json();if(!r.ok)throw Error(data.error||'Sélection indisponible.');fetched.push(...data)}if(alive&&controller===current)remote=[...new Map(fetched.map(w=>[w.id,w])).values()];}catch(e){if(alive&&controller===current&&!current.signal.aborted)error=e instanceof Error?e.message:'Sélection indisponible.'}finally{if(alive&&controller===current)loading=false}
 }
 onMount(()=>()=>{alive=false;controller?.abort()});
</script>
{#if cards.length}
 {#if block.display==='spotlight'}{@const hero=cards[0]}<section class="collection-spotlight" class:music-spotlight={hero.kind==='music'}><div class="spotlight-copy"><span class="eyebrow">{block.title}</span><h2>{hero.title}</h2><p>{hero.subtitle}</p><div><button class="primary" onclick={()=>open(hero)}>Explorer {hero.kind==='music'?'l’album':'la fiche'} <Icon name="arrow" size={17}/></button></div><small>{#if hero.local}Déjà chez vous · prêt à {hero.kind==='music'?'écouter':'regarder'}{:else}À découvrir{/if}</small></div><button class="spotlight-art" aria-label={'Ouvrir '+hero.title} onclick={()=>open(hero)}>{#if hero.image}<img src={hero.image} alt=""/>{:else}<Icon name={hero.kind} size={75}/>{/if}</button><span class="spotlight-word" aria-hidden="true">mosaune</span></section>
 {:else}<MediaShelf title={block.title??'Votre sélection'} subtitle={block.text??''} {cards} {open} {playAlbum} {play} {mark} grid={block.display==='grid'}/>{/if}
{:else if block.source==='recent'&&block.display!=='spotlight'}<p class="folder-help">{block.title} · aucun média pour le moment.</p>
{:else if block.source==='favorites'||block.source==='watchlist'}<p class="folder-help">{block.title} · ajoutez des œuvres depuis leurs fiches.</p>{/if}
{#if loading}<p role="status" class="folder-help">{block.title} · chargement…</p>{/if}{#if error}<p role="status" class="folder-help">{block.title} · {error}</p>{/if}
