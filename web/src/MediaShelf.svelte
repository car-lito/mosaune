<script lang="ts">
 import Icon from './Icon.svelte';
 import { timeLabel, type Card } from './experience';
 import type { Entry } from './types';
 export let title:string;
 export let subtitle='';
 export let cards:Card[]=[];
 export let open:(card:Card)=>void;
 export let play:((entry:Entry)=>void)|undefined=undefined;
 export let mark:((entry:Entry,completed:boolean)=>void)|undefined=undefined;
 export let grid=false;
 export let playAlbum:((card:Card)=>void)|undefined=undefined;
 let rail:HTMLDivElement;
 let failed:Record<string,boolean>={};
</script>
<section class="shelf" aria-label={title}>
 <div class="shelf-heading"><div><h2>{title}</h2>{#if subtitle}<p>{subtitle}</p>{/if}</div>{#if !grid&&cards.length>4}<div class="rail-controls"><button class="icon-button" aria-label={`Défiler ${title} vers la gauche`} onclick={()=>rail.scrollBy({left:-rail.clientWidth*.8,behavior:'smooth'})}>←</button><button class="icon-button" aria-label={`Défiler ${title} vers la droite`} onclick={()=>rail.scrollBy({left:rail.clientWidth*.8,behavior:'smooth'})}>→</button></div>{/if}</div>
 <div bind:this={rail} class="media-rail" class:media-grid={grid}>
 {#each cards as card(card.key)}{@const picture=card.image&&!failed[card.image]?card.image:card.fallbackImage&&!failed[card.fallbackImage]?card.fallbackImage:undefined}<article class="media-tile" class:resume-tile={card.entry}>
  <button class="tile-open" onclick={()=>open(card)} aria-label={`Ouvrir ${card.title}`}>
   <div class="tile-art" class:album-art={card.kind==='music'} class:resume-art={card.entry}>
    {#if picture}<img src={picture} alt="" loading="lazy" onerror={()=>failed={...failed,[picture!]:true}}/>{:else}<div class="tile-placeholder"><Icon name={card.kind} size={34}/><span>{card.title}</span></div>{/if}
    <span class="tile-status" class:local={card.local}><i></i>{card.local?'Chez vous':'À découvrir'}</span>
    {#if card.position&&card.duration}<progress aria-label={`Progression de ${card.title}`} value={card.position} max={card.duration}></progress>{/if}
   </div><strong>{card.title}</strong><small>{card.subtitle}</small>
  </button>
  {#if playAlbum&&card.kind==='music'&&card.local&&!card.entry}<button class="album-quick-play" aria-label={`Lire l’album ${card.title}`} onclick={()=>playAlbum?.(card)}><Icon name="play" size={23}/></button>{/if}
  {#if card.entry&&play&&mark}<div class="resume-actions"><button class="text-action" onclick={()=>play?.(card.entry!)}>▶ Reprendre · {timeLabel(card.position??0)}</button><details class="item-menu"><summary aria-label={`Options de ${card.title}`}>•••</summary><div><button onclick={()=>mark?.(card.entry!,true)}>Marquer comme {card.kind==='music'?'écouté':'vu'}</button><button onclick={()=>mark?.(card.entry!,false)}>Marquer non {card.kind==='music'?'écouté':'vu'} · repartir de zéro</button></div></details></div>{/if}
 </article>{/each}
 </div>
</section>
