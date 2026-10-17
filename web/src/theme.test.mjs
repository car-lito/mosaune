import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import postcss from 'postcss';
const src=path.dirname(fileURLToPath(import.meta.url)),root=path.resolve(src,'../..');
const contract=JSON.parse(fs.readFileSync(path.join(root,'theme/contract.json'),'utf8'));
const theme=JSON.parse(fs.readFileSync(path.join(root,'theme/default.json'),'utf8'));
test('every component style consumes declared visual tokens, including scoped Svelte styles',()=>{
 const known=new Set(Object.keys(contract.tokens).map(k=>'--m-'+k.replaceAll('.','-')));
 assert.equal(known.size,Object.keys(contract.tokens).length,'CSS variable name collision');
 let declarations=0;
 for(const file of fs.readdirSync(src)){
  if(!/\.(css|svelte)$/.test(file)||file==='theme-defaults.css')continue;
  let raw=fs.readFileSync(path.join(src,file),'utf8');if(file.endsWith('.svelte'))raw=[...raw.matchAll(/<style[^>]*>([\s\S]*?)<\/style>/g)].map(m=>m[1]).join('\n');
  postcss.parse(raw).walkDecls(d=>{
   for(const m of d.value.matchAll(/var\((--[a-zA-Z0-9-]+)/g)){assert.ok(known.has(m[1]),`${file}: undeclared ${m[1]}`);declarations++}
   if(/^(color|background|background-color|border|border-color|border-top|border-bottom|border-left|border-right|outline|box-shadow|text-shadow|accent-color|scrollbar-color|fill|stroke)$/.test(d.prop))assert.doesNotMatch(d.value,/#[a-f0-9]{3,8}\b|\b(?:white|black|red|green|blue)\b/i,`${file}: hardcoded ${d.prop}`);
   if(/^(font-size|font-family|border-radius|font-weight|letter-spacing|line-height)$/.test(d.prop))assert.ok(d.value.includes('var(')||['inherit','0'].includes(d.value),`${file}: hardcoded ${d.prop} ${d.value}`);
   if(d.prop==='font')assert.equal(d.value,'inherit',`${file}: font shorthand bypasses theme`);
  });
 }
 assert.ok(declarations>900);
 for(const key of known)assert.ok(Object.keys(theme.tokens).some(k=>'--m-'+k.replaceAll('.','-')===key));
});
test('section is explicit, while applying the theme never excludes settings',()=>{
 const app=fs.readFileSync(path.join(src,'App.svelte'),'utf8');
 assert.match(app,/data-extension-theme=\{extensionAppearance.stamp\}/);
 assert.match(app,/data-ui-section=\{view\}/);
 assert.doesNotMatch(app,/data-extension-theme=\{view/);
});
