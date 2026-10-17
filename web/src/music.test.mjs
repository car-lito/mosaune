import test from 'node:test';
import assert from 'node:assert/strict';
import { emptyQueue, makeQueue, appendQueue, shuffleQueue, moveItem, nextInQueue, savedQueue, restoreQueue } from './music.ts';

test('queue follows playlist order including duplicate tracks, stops or repeats as requested', () => {
  const queue = makeQueue(['a', 'b', 'a']);
  const available = new Set(['a', 'b']);
  assert.equal(nextInQueue(queue, available)?.key, queue.items[1].key);
  queue.current = queue.items[2].key;
  assert.equal(nextInQueue(queue, available), undefined);
  assert.equal(nextInQueue({ ...queue, repeat: 'all' }, available)?.key, queue.items[0].key);
  assert.equal(nextInQueue({ ...queue, repeat: 'one' }, available, 1, true)?.key, queue.current);
  queue.current = queue.items[0].key;
  assert.equal(nextInQueue({ ...queue, repeat: 'one' }, available)?.key, queue.items[1].key, 'manual skip ignores repeat one');
  assert.equal(nextInQueue({ ...queue, repeat: 'all' }, available, -1)?.key, queue.items[2].key);
  assert.equal(nextInQueue({ ...queue, repeat: 'all' }, new Set()), undefined, 'no loop on missing files');
});
test('missing files are skipped and a single available track can repeat', () => {
  const queue = makeQueue(['gone', 'live', 'gone']);
  assert.equal(nextInQueue(queue, new Set(['live']))?.id, 'live');
  queue.current = queue.items[1].key;
  assert.equal(nextInQueue(queue, new Set(['live'])), undefined);
  assert.equal(nextInQueue({ ...queue, repeat: 'all' }, new Set(['live']))?.key, queue.current);
  assert.equal(nextInQueue(emptyQueue(), new Set()), undefined);
});
test('shuffle only affects upcoming tracks and reordering keeps playing identity', () => {
  const queue = makeQueue(['a', 'b', 'c', 'd', 'e'], 1);
  const shuffled = shuffleQueue(queue, () => 0);
  assert.deepEqual(shuffled.items.slice(0, 2), queue.items.slice(0, 2));
  assert.notDeepEqual(shuffled.items.slice(2), queue.items.slice(2));
  assert.deepEqual(shuffled.items.map(i => i.key).sort(), queue.items.map(i => i.key).sort());
  const reordered = { ...queue, items: moveItem(queue.items, 1, 3) };
  assert.equal(nextInQueue(reordered, new Set(['a','b','c','d','e']))?.id, 'e');
  assert.equal(queue.items[1].id, 'b', 'original is immutable');
});
test('restoration preserves queue settings and missing library IDs, excludes temporary files', () => {
  const queue = makeQueue(['disk', 'temporary', 'disconnected'], 1, 'all');
  const restored = restoreQueue(savedQueue(queue, [{ id: 'disk', kind: 'music', libraryId: 'library' }, { id: 'temporary', kind: 'music' }]));
  assert.deepEqual(restored.items.map(i => i.id), ['disk', 'disconnected']);
  assert.equal(restored.current, null);
  assert.equal(restored.repeat, 'all');
  const duplicateKeys = JSON.stringify({ ...queue, items: [queue.items[0], queue.items[0]] });
  for (const raw of ['null', '{', '{"items":[null],"repeat":"off"}', duplicateKeys, JSON.stringify({ ...queue, repeat: 'bad' })]) assert.deepEqual(restoreQueue(raw), emptyQueue());
});
test('queue capacity rejects entire append without discarding existing tracks', () => {
  const queue = makeQueue(Array(199).fill('a'));
  assert.throws(() => appendQueue(queue, ['b','c']), /200/);
  assert.equal(queue.items.length, 199);
  assert.equal(appendQueue(queue, ['b']).items.length, 200);
  assert.throws(() => makeQueue(Array(201).fill('a')), /200/);
});
