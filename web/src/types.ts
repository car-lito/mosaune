import type {ResultPresentation} from './search-presentation';
export type Kind = 'films' | 'series' | 'music';
export type Metadata = {
  presentation?:ResultPresentation;
  runtime?: number; releaseDate?: string; tagline?: string; rating?: number; voteCount?: number; status?: string; countries?: string[]; languages?: string[]; contributors?: Contributor[]; seasons?: {number:number;name:string;image?:string;episodes:number}[]; trailers?: {name:string;url:string;thumbnail?:string;language?:string}[]; tracks?: CatalogTrack[]; edition?: string;
  title: string; series?: string; season?: number; episode?: number; episodeEnd?: number;
  artist?: string; album?: string; albumArtist?: string; track?: number; disc?: number;
  year?: number; overview?: string; cast?: string[]; genres?: string[]; imageId?: string;
  provider: string; providerId?: string; matchState: string; manual?: boolean; origins?: Record<string,string>;
};
export type Contributor = { id: string; name: string; role?: string; image?:string };
export type CatalogTrack = {id:string; title:string; disc:number; number:number; duration:number; artists?:Contributor[]};
export type Entry = { posterUrl?:string; addedAt?:number; lastPlayed?:number; workId?:string; id: string; name: string; filename: string; kind: Kind; url: string; size: number; libraryId?: string; position?: number; duration?: number; completed?: boolean; revision?: string; metadata?: Metadata };
export type Provider = { id: string; name: string; contractVersion: number; kinds: Kind[]; configured: boolean; enabled: boolean; requiresCredential: boolean; discovery?: boolean; attribution: string; website: string };
export type Work = { posterUrl?:string; id: string; kind: Kind; metadata: Metadata; favorite: boolean; watchlist: boolean; localIds: string[]; watched: boolean; poster: boolean };
