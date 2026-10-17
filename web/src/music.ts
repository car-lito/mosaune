import type { Entry } from './types';

export type QueueItem = { key: string; id: string };
export type MusicQueue = { items: QueueItem[]; current: string | null; repeat: 'off' | 'all' | 'one' };
export type Playlist = { imageId?:string; id: string; name: string; revision: number; tracks: { id: string; name: string }[] };
export const queueLimit = 200;
export const emptyQueue = (): MusicQueue => ({ items: [], current: null, repeat: 'off' });
export function makeQueue(ids: string[], start = 0, repeat: MusicQueue['repeat'] = 'off'): MusicQueue {
  if (ids.length > queueLimit) throw new Error('La file peut contenir au maximum 200 morceaux.');
  const items = ids.map(id => ({ id, key: crypto.randomUUID() }));
  return { items, current: items[start]?.key ?? null, repeat };
}
export function appendQueue(queue: MusicQueue, ids: string[]): MusicQueue {
  if (queue.items.length + ids.length > queueLimit) throw new Error('La file peut contenir au maximum 200 morceaux.');
  return { ...queue, items: [...queue.items, ...makeQueue(ids).items] };
}
// The visible order is the playback order. Shuffling leaves the playing track
// and its history in place, and only mixes the upcoming tracks.
export function shuffleQueue(queue: MusicQueue, random = Math.random): MusicQueue {
  const items = [...queue.items];
  const first = items.findIndex(item => item.key === queue.current) + 1;
  for (let i = items.length - 1; i > first; i--) {
    const j = first + Math.floor(random() * (i - first + 1));
    [items[i], items[j]] = [items[j], items[i]];
  }
  return { ...queue, items };
}
export function moveItem<T>(items: T[], from: number, to: number): T[] {
  if (from < 0 || to < 0 || from >= items.length || to >= items.length) return items;
  const result = [...items]; const [item] = result.splice(from, 1); result.splice(to, 0, item); return result;
}
export function nextInQueue(queue: MusicQueue, available: Set<string>, direction = 1, ended = false): QueueItem | undefined {
  const index = queue.items.findIndex(item => item.key === queue.current);
  if (ended && queue.repeat === 'one' && index >= 0 && available.has(queue.items[index].id)) return queue.items[index];
  for (let offset = 1; offset <= queue.items.length; offset++) {
    let i = index + direction * offset;
    if (queue.repeat === 'all') i = (i % queue.items.length + queue.items.length) % queue.items.length;
    if (i < 0 || i >= queue.items.length) break;
    if (available.has(queue.items[i].id)) return queue.items[i];
  }
  return undefined;
}
export function savedQueue(queue: MusicQueue, entries: Entry[]): string {
  const temporary = new Set(entries.filter(e => !e.libraryId || e.kind !== 'music').map(e => e.id));
  const items = queue.items.filter(item => !temporary.has(item.id));
  return JSON.stringify({ ...queue, items, current: items.some(item => item.key === queue.current) ? queue.current : null });
}
export function restoreQueue(raw: string | null): MusicQueue {
  if (!raw) return emptyQueue();
  try {
    const q = JSON.parse(raw);
    if (!q || !Array.isArray(q.items) || q.items.length > queueLimit || !['off', 'all', 'one'].includes(q.repeat)) return emptyQueue();
    if (!q.items.every((item: QueueItem) => item && typeof item.id === 'string' && /^[a-zA-Z0-9_-]{1,80}$/.test(item.id) && typeof item.key === 'string' && /^[a-zA-Z0-9_-]{1,80}$/.test(item.key))) return emptyQueue();
    if (new Set(q.items.map((item: QueueItem) => item.key)).size !== q.items.length) return emptyQueue();
    return { items: q.items.map((item: QueueItem) => ({ id: item.id, key: item.key })), current: q.items.some((item: QueueItem) => item.key === q.current) ? q.current : null, repeat: q.repeat };
  } catch { return emptyQueue(); }
}
