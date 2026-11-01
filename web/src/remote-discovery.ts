import type { Kind, Provider, Work } from './types.ts';
import type { UIBlock } from './ui-pages.ts';
export function offersSearch(p: Provider, kind: Kind, similar = false) {
    if (!p.discovery || !p.kinds.includes(kind))
        return false;
    if (!p.catalogue)
        return true; // legacy provider text/similar capability
    return p.catalogue.version === 1 && (p.catalogue.search ?? []).some(s => s.kind === kind && (!similar || s.similar));
}
export const searchSources = (providers: Provider[], scope: Kind[]) => providers.filter(p => scope.some(k => offersSearch(p, k)));
export function searchJobs(providers: Provider[], scope: Kind[]) {
    const seen = new Set<string>();
    const queues = [...new Set(scope)].map(kind => providers.filter(p => p.enabled && p.configured && offersSearch(p, kind)).map(p => ({ provider: p.id, name: p.name, kind })).filter(j => { const k = j.provider + ':' + j.kind; if (seen.has(k))
        return false; seen.add(k); return true; }));
    return interleave(queues, 6);
}
export function searchPlaceholder(providers: Provider[], scope: Kind[]) {
    if (scope.length === 1)
        for (const p of providers) {
            const value = p.catalogue?.search?.find(s => s.kind === scope[0])?.placeholder;
            if (value)
                return value;
        }
    return 'Rechercher dans Mosaune…';
}
export function searchScopeLabel(providers: Provider[], scope: Kind[]) {
    if (scope.length === 1)
        for (const p of providers) {
            const value = p.catalogue?.search?.find(s => s.kind === scope[0])?.label;
            if (value)
                return value;
        }
    return scope.length === 1 ? 'Cette rubrique' : 'Tous les univers';
}
export function searchUnavailable(providers: Provider[], scope: Kind[]) {
    return scope.filter(k => !searchJobs(providers, [k]).length).map(kind => {
        const candidates = searchSources(providers, [kind]);
        if (!candidates.length)
            return 'Aucun fournisseur de recherche en ligne pour cette rubrique. Votre collection reste disponible.';
        const names = [...new Set(candidates.map(p => p.name))].join(', ');
        return names + ' : ' + (candidates.some(p => p.enabled) ? 'configuration nécessaire' : 'désactivé') + '. Votre collection reste disponible.';
    }).filter((v, i, a) => a.indexOf(v) === i).join(' ');
}
export function shelfPlan(block: UIBlock, providers: Provider[], works: Work[], kinds: Kind[]) {
    if (block.source !== 'remote' || !kinds.includes(block.view as Kind))
        return [];
    const p = providers.find(p => p.id === block.provider && p.enabled && p.configured && p.kinds.includes(block.view as Kind));
    const shelf = p?.catalogue?.version === 1 ? p.catalogue.shelves?.find(s => s.kind === block.view && s.id === block.shelf) : undefined;
    if (!p || !shelf)
        return [];
    const selectors = shelf.selector === 'kind' ? [''] : [...new Set(works.filter(w => w.kind === block.view && w.metadata.provider === p.id && (w.localIds.length > 0 || w.favorite)).flatMap(w => w.metadata.contributors?.map(a => a.id).filter(Boolean) ?? []))].sort().slice(0, 3);
    return selectors.map(artist => ({ provider: p.id, kind: shelf.kind, shelf: shelf.id, artist }));
}
// Fair bounded merging: a busy first provider/category cannot hide all others.
export function interleave<T>(batches: T[][], limit: number): T[] {
    const out: T[] = [];
    for (let row = 0; out.length < limit; row++) {
        let added = false;
        for (const batch of batches) {
            if (row < batch.length) {
                out.push(batch[row]);
                added = true;
                if (out.length === limit)
                    return out;
            }
        }
        if (!added)
            break;
    }
    return out;
}
