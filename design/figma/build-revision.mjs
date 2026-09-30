import {readFile, writeFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import {resolve} from 'node:path';
import vm from 'node:vm';

const output = process.argv[2];
if (!output) throw new Error('用法：node design/figma/build-revision.mjs /tmp/math-master-english-revision.js');
const directory = fileURLToPath(new URL('.', import.meta.url));
const read = async path => JSON.parse(await readFile(resolve(directory,path),'utf8'));
const [catalogue, route, template] = await Promise.all([
  read('../data/knowledge-domains.json'),
  read('../data/elementary-route.json'),
  readFile(resolve(directory,'english-revision.template.js'),'utf8'),
]);
if (catalogue.domains.length !== 16) throw new Error('板块数量必须为 16');
const source = template.replace('__CATALOGUE__',JSON.stringify(catalogue)).replace('__ROUTE__',JSON.stringify(route));
// use_figma 会提供异步作用域；这里只检查语法，不执行画布操作。
new vm.Script(`(async function(){\n${source}\n})`);
await writeFile(resolve(output),source,'utf8');
process.stdout.write(`已生成待执行脚本：${resolve(output)}\n语法检查通过；尚未同步至 Figma。\n`);
