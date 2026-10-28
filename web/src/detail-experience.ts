import type {Contributor} from './types';
export type DetailAction={operation:'play'|'open'|'inspect'|'mark'|'poster'|'match'|'enqueue'|'playlist'|'watchlist'|'favorite'|'request'|'unavailable';label:string;fileId?:string;primary?:boolean};
export type ExperienceRow={key:string;fileId?:string;title:string;subtitle:string;ordinal:string;duration:number;actions:DetailAction[]};
export type ExperienceView={owner:string;cover:'square'|'portrait';actions:DetailAction[];people:number[];peopleResources:Contributor[];provider:string;sections:{key:string;title:string;help:string;rows:ExperienceRow[];actions:DetailAction[]}[];episodeList:boolean;albumLink:boolean;suggestions:{enabled:boolean;title:string;help:string}};
// Resolving an action never accepts a path, script or arbitrary request from an extension.
export function actionEntries<T extends {id:string}>(action:DetailAction,entries:T[]):T[]{return action.fileId?entries.filter(e=>e.id===action.fileId):entries;}
