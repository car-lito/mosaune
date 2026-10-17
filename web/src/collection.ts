import type { Entry, Kind } from './types.ts';
export type Group = { key: string; title: string; subtitle: string; kind: Kind; imageId?: string; entries: Entry[] };
export function groupMedia(entries: Entry[]): Group[] {
  const groups = new Map<string,Group>();
  for (const entry of entries) {
    const m = entry.metadata;
    let key = entry.id, title = m?.title || entry.name, subtitle = entry.kind === 'films' ? (m?.year ? String(m.year) : 'Film') : '';
    if (entry.kind === 'series' && m?.series) { key = `series:${m.providerId ? `${m.provider}:${m.providerId}` : m.series.toLocaleLowerCase()}`; title = m.series; subtitle = 'Série'; }
    if (entry.kind === 'music') {
      const artist = m?.albumArtist || m?.artist || 'Artiste inconnu'; subtitle = artist;
      if (m?.album) { key = `music:${JSON.stringify([artist.toLocaleLowerCase(),m.album.toLocaleLowerCase()])}`; title = m.album; }
    }
    const existing = groups.get(key);
    if (existing) { existing.entries.push(entry); existing.imageId ||= m?.imageId; }
    else groups.set(key,{key,title,subtitle,kind:entry.kind,imageId:m?.imageId,entries:[entry]});
  }
  for (const group of groups.values()) group.entries.sort((a,b) => {
    const x=a.metadata,y=b.metadata;
    if (a.kind==='series') return (x?.season??0)-(y?.season??0) || (x?.episode??0)-(y?.episode??0) || a.filename.localeCompare(b.filename);
    if (a.kind==='music') return (x?.disc??0)-(y?.disc??0) || (x?.track??0)-(y?.track??0) || a.filename.localeCompare(b.filename);
    return a.name.localeCompare(b.name);
  });
  return [...groups.values()].sort((a,b)=>a.title.localeCompare(b.title,'fr'));
}
