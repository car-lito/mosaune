import type {Provider} from './types';

// Provider names and IDs never imply support for a profile operation.
export function artistResolvers(providers:Provider[]):Provider[]{
 return providers.filter(p=>p.profiles&&p.resolveArtists&&p.kinds.includes('music')&&p.enabled&&p.configured&&p.available!==false);
}
export function artistResolutionRoute(provider:string):string{return '/api/v1/people/'+encodeURIComponent(provider)+'/resolve';}
export function personProfileRoute(provider:string,id:string,page:number):string{return '/api/v1/people/'+encodeURIComponent(provider)+'/'+encodeURIComponent(id)+'?page='+page;}
