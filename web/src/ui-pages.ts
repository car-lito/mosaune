import type {Entry,Kind,Work} from './types.ts';
import {recentGroups,workCard,unfinished} from './experience.ts';
export type UIBlock={type:'text'|'facts'|'link'|'collection'|'feed';source?:string;display?:string;provider?:string;shelf?:string;title?:string;text?:string;target?:string;view?:string;facts?:{label:string;value:string}[]};
export type UIPage={id:string;screen?:string;heading?:string;owner?:string;label:string;description?:string;placement:string;icon?:string;blocks?:UIBlock[]};
export const pageKey=(p:UIPage)=>p.owner?'extension:'+p.owner+'/'+p.id:p.id;
export function collectionCards(view:string,entries:Entry[],works:Work[],kinds:Kind[]){
 if(view==='favorites'||view==='watchlist')return works.filter(w=>kinds.includes(w.kind)&&(view==='favorites'?w.favorite:w.watchlist)).map(workCard);
 return kinds.filter(k=>view==='home'||k===view).flatMap(k=>recentGroups(entries,k));
}

export function feedCards(block:UIBlock,entries:Entry[],works:Work[],kinds:Kind[]){
 const allowed=(k:Kind)=>kinds.includes(k)&&(block.view==='home'||block.view===k);
 const local=entries.filter(e=>allowed(e.kind));
 if(block.source==='resume')return unfinished(local);
 if(block.source==='favorites'||block.source==='watchlist')return works.filter(w=>allowed(w.kind)&&(block.source==='favorites'?w.favorite:w.watchlist)).map(workCard);
 if(block.source==='recent')return kinds.filter(allowed).flatMap(k=>recentGroups(local,k));
 return [];
}
export function screenPage(pages:UIPage[],view:string){return pages.find(p=>p.screen===view);}
