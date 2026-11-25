import type {Provider} from './types';

export function albumMatchSources(providers:Provider[]):Provider[]{
 return providers.filter(p=>p.albumMatches&&p.kinds.includes('music')&&p.enabled&&p.configured&&p.available!==false);
}
export function albumMatchRoute(provider:string):string{return 'music/matches/'+encodeURIComponent(provider);}
