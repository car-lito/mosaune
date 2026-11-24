import test from 'node:test';
import assert from 'node:assert/strict';
import {providerStatus,providerRoute,providerPackageRoute} from './provider-management.ts';

test('status distinguishes missing, paused, denied and unavailable providers',()=>{
 const base={enabled:true,available:true};
 for(const [patch,status] of [
  [{},'Actif'],[{enabled:false},'Désactivé'],[{available:false},'Indisponible'],
  [{runtime:{installed:false}},'Non installé'],[{runtime:{safeMode:true}},'Mode de secours'],
  [{runtime:{enabled:false}},'Module en pause'],[{runtime:{serviceAccess:false}},'Accès en ligne retiré'],
 ])assert.equal(providerStatus({...base,...patch}),status);
 assert.equal(providerStatus({...base,enabled:false,runtime:{enabled:false}}),'Désactivé');
});
test('all provider actions address the selected adapter without interpreting path characters',()=>{
 assert.equal(providerRoute('another-source','state'),'metadata/providers/another-source/state');
 assert.equal(providerRoute('another-source','runtime'),'metadata/providers/another-source/runtime');
 assert.equal(providerPackageRoute('another-source','install'),'provider-packages/another-source/install');
 assert.equal(providerPackageRoute('a/b?c','rollback'),'provider-packages/a%2Fb%3Fc/rollback');
 assert.equal(providerPackageRoute('another-source'),'provider-packages/another-source');
});
