import type {Entry,Kind,Work} from './types.ts';
import {recentGroups,workCard} from './experience.ts';
export type UIBlock={type:'text'|'facts'|'link'|'collection';title?:string;text?:string;target?:string;view?:string;facts?:{label:string;value:string}[]};
export type UIPage={id:string;owner?:string;label:string;description?:string;placement:string;icon?:string;blocks?:UIBlock[]};
export const pageKey=(p:UIPage)=>p.owner?'extension:'+p.owner+'/'+p.id:p.id;
export function collectionCards(view:string,entries:Entry[],works:Work[],kinds:Kind[]){
 if(view==='favorites'||view==='watchlist')return works.filter(w=>kinds.includes(w.kind)&&(view==='favorites'?w.favorite:w.watchlist)).map(workCard);
 return kinds.filter(k=>view==='home'||k===view).flatMap(k=>recentGroups(entries,k));
}
