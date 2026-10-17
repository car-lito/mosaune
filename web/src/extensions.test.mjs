import test from 'node:test';
import assert from 'node:assert/strict';
import {appearanceLink} from './extensions.ts';
test('stylesheet URL is host-owned and disappears with the effective theme',()=>{
 const stamp='a'.repeat(64);
 assert.equal(appearanceLink({effective:'atelier.ambre',stamp}),'/api/v1/extensions/theme.css?stamp='+stamp);
 assert.equal(appearanceLink({effective:'',stamp}),'');
 assert.equal(appearanceLink({effective:'atelier.ambre',stamp:'" onload="alert(1)'}),'');
});
