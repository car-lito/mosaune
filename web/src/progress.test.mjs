import { test } from 'node:test';
import assert from 'node:assert/strict';
import { queueProgress, waitForProgress } from './progress.ts';

test('serialize and coalesce updates, including a restart of the same video', async () => {
  const original = globalThis.fetch;
  const requests = [];
  const responses = [];
  const saved = [];
  globalThis.fetch = (url, options) => new Promise(resolve => { requests.push({url, ...options}); responses.push(resolve); });
  try {
    const update = position => ({body: JSON.stringify({position}), token:'test', result: ok => saved.push(ok)});
    queueProgress('movie', update(40));
    queueProgress('movie', update(42));
    queueProgress('movie', update(0));
    let drained = false;
    const pending = waitForProgress('movie').then(() => { drained = true; });
    assert.equal(requests.length,1);
    responses.shift()({ok:true}); await new Promise(resolve => setImmediate(resolve));
    assert.equal(requests.length,2);
    assert.equal(drained,false);
    assert.equal(JSON.parse(requests[1].body).position,0);
    responses.shift()({ok:false}); await new Promise(resolve => setImmediate(resolve));
    assert.deepEqual(saved,[true,false]);
    await pending;
    assert.equal(drained,true);
    await waitForProgress('already-idle');
    queueProgress('movie', update(5)); assert.equal(requests.length,3);
    responses.shift()({ok:true}); await new Promise(resolve => setImmediate(resolve));
  } finally { globalThis.fetch = original; }
});
