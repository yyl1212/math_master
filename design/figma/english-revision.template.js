// 待执行：交给目标文件的 use_figma，不直接在 Node 中运行。
// 先核对字体和原组件；新增独立英文修订版，不修改上一版原稿。
const catalogue = __CATALOGUE__;
const route = __ROUTE__;
const revision = 'English / 16 domains / v2';
if (figma.fileKey && figma.fileKey !== 'UDwHVfKZYNDTdvDx8V3dLH') throw Error('目标 Figma 文件不匹配');
const page = await figma.getNodeByIdAsync('0:1');
if (!page || page.type !== 'PAGE') throw Error('未找到目标页面');
await figma.setCurrentPageAsync(page);
const existing = page.children.filter(n => n.name.startsWith(revision));
if (existing.length) return {status:'already-present-review-required',existingNodeIds:existing.map(n=>n.id),createdNodeIds:[],mutatedNodeIds:[]};

const componentIds = {Button:'4:77',Badge:'4:92',NavItem:'4:97',KnowledgeCard:'4:132',StatCard:'4:98'};
const C = Object.fromEntries(await Promise.all(Object.entries(componentIds).map(async([key,id])=>[key,await figma.getNodeByIdAsync(id)])));
for (const [key,node] of Object.entries(C)) if (!node || !['COMPONENT','COMPONENT_SET'].includes(node.type)) throw Error('原组件缺失：'+key);
const variableIds = {paper:'4:6',surface:'4:8',ink:'4:10',muted:'4:12',brand:'4:14',soft:'4:16',border:'4:18',amber:'4:20',sand:'4:22',neutral:'4:24',white:'4:26',blue:'4:28',blueSoft:'4:30'};
const V = Object.fromEntries(await Promise.all(Object.entries(variableIds).map(async([key,id])=>[key,await figma.variables.getVariableByIdAsync('VariableID:'+id)])));
if (Object.values(V).some(v=>!v)) throw Error('原颜色变量缺失');
const available = await figma.listAvailableFontsAsync();
const fonts = ['Regular','Medium','Bold'].map(style=>({family:'Noto Sans SC',style}));
for (const f of fonts) if (!available.some(x=>x.fontName.family===f.family&&x.fontName.style===f.style)) throw Error('字体缺失：'+f.family+' '+f.style);
const fontMap = new Map(fonts.map(f=>[f.family+'/'+f.style,f]));
for (const component of Object.values(C)) for (const text of component.findAllWithCriteria({types:['TEXT']})) for (const segment of text.getStyledTextSegments(['fontName'])) fontMap.set(segment.fontName.family+'/'+segment.fontName.style,segment.fontName);
await Promise.all([...fontMap.values()].map(f=>figma.loadFontAsync(f)));

const createdNodeIds=[],mutatedNodeIds=[],links=[],screens={};
const originX = Math.max(...page.children.map(n=>n.x+n.width),0)+240;
const englishStates = {'已掌握':'Mastered','已学过':'Learned','学习中':'In progress','待巩固':'Needs review','未解锁':'Locked','已解锁':'Unlocked','筹备中':'Content planned','学习：未开始 · 内容可浏览':'Not started · Reading available','学习：未开始 · 可进入检测':'Not started · Checkpoint available','已解锁 · 可回顾与检测':'Unlocked · Review available'};
function track(n) { createdNodeIds.push(n.id); return n; }
function paint(key) { return figma.variables.setBoundVariableForPaint({type:'SOLID',color:{r:0,g:0,b:0}},'color',V[key]); }
function box(parent,name,width,direction='VERTICAL',padding=0,gap=16,bg=null) {
  const f=track(figma.createAutoLayout(direction));f.name=name;f.resize(width,60);f.layoutSizingHorizontal='FIXED';f.layoutSizingVertical='HUG';f.fills=bg?[paint(bg)]:[];f.itemSpacing=gap;f.paddingTop=f.paddingBottom=f.paddingLeft=f.paddingRight=padding;f.clipsContent=false;if(bg)f.cornerRadius=16;if(parent)parent.appendChild(f);return f;
}
function text(parent,value,width,size=16,color='ink',weight='Regular') {
  const t=track(figma.createText());t.name=String(value).slice(0,60);t.fontName={family:'Noto Sans SC',style:weight};t.fontSize=size;t.lineHeight={unit:'PERCENT',value:145};t.fills=[paint(color)];t.characters=String(value);t.resize(width,28);t.textAutoResize='HEIGHT';parent.appendChild(t);return t;
}
function properties(instance,values) {
  const definitions=instance.componentProperties;const edits={};for(const [name,value] of Object.entries(values)){const key=Object.keys(definitions).find(k=>k.split('#')[0]===name);if(!key)throw Error('组件属性缺失：'+name);edits[key]=value;}instance.setProperties(edits);
}
function instance(parent,family,variant,values={},width) {
  const source=C[family].type==='COMPONENT_SET'?C[family].children.find(n=>n.name===variant):C[family];if(!source)throw Error('组件变体缺失：'+family+'/'+variant);
  const i=track(source.createInstance());parent.appendChild(i);createdNodeIds.push(...i.findAll(()=>true).map(n=>n.id));properties(i,values);
  for(const t of i.findAllWithCriteria({types:['TEXT']})){if(englishStates[t.characters])t.characters=englishStates[t.characters];t.textAutoResize='HEIGHT';}
  if(width){i.resize(width,i.height);i.layoutSizingVertical='HUG';}return i;
}
function button(parent,label,to,secondary=false) { const i=instance(parent,'Button','Variant='+(secondary?'Secondary':'Primary'),{Label:label});if(to)links.push({id:i.id,to});return i; }
function badge(parent,state) { return instance(parent,'Badge','State='+state); }
function heading(parent,kicker,title,description,w) { text(parent,kicker,w,12,'brand','Medium');text(parent,title,w,34,'ink','Bold');if(description)text(parent,description,w,16,'muted'); }
function screen(name,width,index,mobile=false) {
  const f=box(page,revision+' / '+name,width,'VERTICAL',0,0,'paper');f.cornerRadius=0;f.x=originX+(mobile?index*590:(index%3)*1680);f.y=mobile?7700:Math.floor(index/3)*2600+160;screens[name]=f;
  const nav=box(f,'Main navigation',width,'HORIZONTAL',mobile?24:64,mobile?24:32,'surface');nav.counterAxisAlignItems='CENTER';text(nav,'Math Master',mobile?214:240,20,'ink','Bold');
  if(mobile)text(nav,'Menu',56,14,'brand');else for(const [label,target] of [['Learning Hub','Learning Hub'],['Knowledge Map','Knowledge Map'],['Practice','Practice'],['My Progress','My Progress'],['Feedback','Feedback']]){const i=instance(nav,'NavItem','Active='+(name===target?'True':'False'),{Label:label});links.push({id:i.id,to:target});}
  const p=box(f,'Page content',width,'VERTICAL',mobile?24:64,24);const w=width-(mobile?48:128);text(p,'DESIGN PREVIEW · Sample content and learning records',w,12,'muted');return {f,p,w};
}
function knowledge(parent,n,width) {
  const variants={mastered:'Mastered',learned:'Learned',learning:'Learning',review:'Learned',unlearned:n.unlocked?'Unlocked':'Locked'};
  const i=instance(parent,'KnowledgeCard','State='+variants[n.learningState],{Title:n.title,Meta:n.type,Description:n.description},width);
  for(const t of i.findAllWithCriteria({types:['TEXT']})) if(t.characters.startsWith('Unlocked ·')||t.characters.startsWith('Not started ·')) t.characters=(n.unlocked?'Unlocked':'Locked')+' · '+({mastered:'Mastered',learned:'Learned',learning:'In progress',review:'Needs review',unlearned:'Not started'})[n.learningState]+'\n'+(n.lessonAvailable?'Sample lesson available':'Outline in this preview');
  if(n.lessonAvailable)links.push({id:i.id,to:'Knowledge Detail'});return i;
}

// 单个新增 DomainCard 组件复用现有颜色变量；不重建已有组件库。
const shelf=box(page,revision+' / DomainCard component',400,'VERTICAL',0,24);shelf.x=originX;shelf.y=11000;
const domainComponent=track(figma.createComponent());shelf.appendChild(domainComponent);domainComponent.name='DomainCard / English';domainComponent.description='学习板块入口。分组范围与内容建设状态不代表个人解锁状态。';domainComponent.layoutMode='VERTICAL';domainComponent.resize(306,280);domainComponent.layoutSizingVertical='HUG';domainComponent.paddingTop=domainComponent.paddingBottom=domainComponent.paddingLeft=domainComponent.paddingRight=24;domainComponent.itemSpacing=12;domainComponent.fills=[paint('surface')];domainComponent.strokes=[paint('border')];domainComponent.cornerRadius=16;
const domainProperties={};
for(const [key,value,size,color,weight] of [['Number','DOMAIN 01',12,'muted','Medium'],['Title','Foundations & Logic',19,'ink','Bold'],['Chinese','数学基础与逻辑',12,'muted','Regular'],['Topics','Sets · Propositions · Proofs · Mathematical logic',14,'muted','Regular'],['Status','Content planned',12,'brand','Medium']]){const t=text(domainComponent,value,258,size,color,weight);const prop=domainComponent.addComponentProperty(key,'TEXT',value);t.componentPropertyReferences={characters:prop};domainProperties[key]=prop;}
function domainCard(parent,d,width) {
  const i=track(domainComponent.createInstance());parent.appendChild(i);createdNodeIds.push(...i.findAll(()=>true).map(n=>n.id));properties(i,{Number:'DOMAIN '+String(d.order).padStart(2,'0'),Title:d.name,Chinese:d.nameZh,Topics:d.topics.join(' · '),Status:d.contentStatus==='preview'?'Preview route':'Content planned'});i.resize(width,i.height);i.layoutSizingVertical='HUG';for(const t of i.findAllWithCriteria({types:['TEXT']})){t.resize(width-48,t.height);t.textAutoResize='HEIGHT';}if(d.id==='elementary-mathematics')links.push({id:i.id,to:'Current Route'});return i;
}
function mapContent(p,w,mobile=false) {
  heading(p,'KNOWLEDGE MAP','Find your place in mathematics.','Explore 16 learning domains, then choose a path.',w);
  const note=box(p,'Grouping explanation',w,'VERTICAL',20,12,'soft');text(note,'16 study-friendly groups — not an official count of mathematical fields.',w-40,14,'brand');
  button(p,'My current route','Current Route',true);
  if(mobile){text(p,'Jump to a domain · All 16 domains',w,14,'muted');for(const d of catalogue.domains)domainCard(p,d,w);}else{const area=box(p,'Directory and domain grid',w,'HORIZONTAL',0,24);const sidebar=box(area,'16 learning domains',252,'VERTICAL',20,16,'surface');text(sidebar,'Learning domains / 16',212,17,'ink','Bold');for(const d of catalogue.domains)text(sidebar,String(d.order).padStart(2,'0')+'  '+d.name,212,13,'muted');const grid=box(area,'All domains',w-276,'VERTICAL',0,24);const cardWidth=(w-324)/3;for(let i=0;i<16;i+=3){const row=box(grid,'Domains '+(i+1)+'–'+Math.min(i+3,16),w-276,'HORIZONTAL',0,24);for(const d of catalogue.domains.slice(i,i+3))domainCard(row,d,cardWidth);}}
}
function lessonContent(p,w,mobile=false) {
  heading(p,'ELEMENTARY MATHEMATICS / FRACTIONS','Equivalent fractions','Different expressions. The same value.',w);const status=box(p,'Independent personal states',w,'HORIZONTAL');badge(status,'Unlocked');badge(status,'Learned');
  const theorem=box(p,'Statement and conditions',w,'VERTICAL',24,16,'soft');text(theorem,'a / b = (a × k) / (b × k)',w-48,mobile?24:32,'brand','Bold');text(theorem,'Conditions: b ≠ 0 and k ≠ 0. Multiplying or dividing the numerator and denominator by the same nonzero number preserves the value.',w-48,16,'ink');
  text(p,'One idea. More than one way to see it.',w,24,'ink','Bold');text(p,'Visual understanding · Symbolic reasoning · Worked examples',w,14,'brand');
  const diagram=box(p,'Original equal-part diagram',w,'VERTICAL',24,16,'surface');const dw=w-48;
  for(const parts of [2,4]){const row=box(diagram,'Equal parts / '+parts,dw,'HORIZONTAL',0,4);for(let i=0;i<parts;i++){const r=track(figma.createRectangle());row.appendChild(r);r.resize((dw-(parts-1)*4)/parts,56);r.fills=[paint(i<parts/2?'brand':'neutral')];r.cornerRadius=4;}text(diagram,parts===2?'1 of 2 equal parts · 1/2':'2 of 4 equal parts · 2/4',dw,14,'muted');}
  text(p,'The same whole, divided differently. Both diagrams shade half of an equal-sized whole.',w,16,'muted');text(p,'Why it works',w,24,'ink','Bold');text(p,'Multiplying both terms by k multiplies the fraction by k/k = 1. The value stays unchanged. Division uses the same reasoning with 1/k.',w,16,'muted');text(p,'Worked examples',w,24,'ink','Bold');text(p,'6/8 = (6 ÷ 2)/(8 ÷ 2) = 3/4\n10/15 = (10 ÷ 5)/(15 ÷ 5) = 2/3',w,18,'brand');text(p,'Do not add the same number to both terms: 1/2 ≠ 2/3. Never divide by zero.',w,16,'muted');button(p,'Start the checkpoint','Practice');button(p,'Report a content issue','Feedback',true);text(p,'Original sample lesson · v1.0 · Design preview',w,12,'muted');
}

{
  const {p,w}=screen('Learning Hub',1440,0);heading(p,'YOUR NEXT SMALL STEP','Make every idea part of your thinking.','Begin with a short review of equivalent fractions.',w);button(p,'Start today’s review','Knowledge Detail');button(p,'Explore the knowledge map','Knowledge Map',true);
  const row=box(p,'Sample progress',w,'HORIZONTAL',0,24);for(const [label,value,note] of [['Knowledge points learned','12 / 30','Wider Elementary learning path'],['Mastered','8','Supported by checkpoints'],['Ready for review','3','Revisit and strengthen'],['This week','4 days','86 minutes']])instance(row,'StatCard',null,{Label:label,Value:value,Note:note},(w-72)/4);
  text(p,'Continue your learning path',w,24,'ink','Bold');text(p,'Numbers & operations → Fractions & decimals → Ratios & percentages',w,18,'muted');const cards=box(p,'Suggested knowledge points',w,'HORIZONTAL',0,24);for(const id of ['equivalent-fractions','decimal-place-value','simplifying-fractions'])knowledge(cards,route.nodes.find(n=>n.id===id),(w-48)/3);text(p,'16 domains. Many ways to grow.',w,24,'ink','Bold');button(p,'Browse every learning domain','Knowledge Map',true);
}
{ const {p,w}=screen('Knowledge Map',1440,1);mapContent(p,w); }
{
  const {p,w}=screen('Current Route',1440,2);heading(p,'ELEMENTARY MATHEMATICS / ARITHMETIC',route.title,'A sample route through a much larger subject.',w);text(p,'Learning prerequisites are listed on each card. Related domains are not automatic prerequisites.',w,14,'muted');
  const depth=new Map();function level(id){if(depth.has(id))return depth.get(id);const n=route.nodes.find(x=>x.id===id);const value=n.prerequisites.length?Math.max(...n.prerequisites.map(level))+1:0;depth.set(id,value);return value;}route.nodes.forEach(n=>level(n.id));
  for(let i=0;i<=Math.max(...depth.values());i++){const row=box(p,'Prerequisite level '+(i+1),w,'HORIZONTAL',0,24);const group=route.nodes.filter(n=>depth.get(n.id)===i);for(const n of group){const col=box(row,n.title,(w-48)/3);knowledge(col,n,(w-48)/3);text(col,'Before this point: '+(n.prerequisites.length?n.prerequisites.map(id=>route.nodes.find(x=>x.id===id).title).join(' · '):'No prerequisites'),(w-48)/3,13,'muted');}}
  text(p,'Unlock and learning status stay separate. Locked published lessons remain readable. Most points here show only a preview outline.',w,14,'muted');button(p,'Back to all domains','Knowledge Map',true);
}
{const {p,w}=screen('Knowledge Detail',1440,3);lessonContent(p,w);}
{const {p,w}=screen('Practice',1440,4);heading(p,'KNOWLEDGE CHECKPOINT / EQUIVALENT FRACTIONS','Let an idea stand up to a question.','5 questions. At least 4 correct to pass.',w);text(p,'01 Answered · 02 Answered · 03 Current · 04 Not answered · 05 Not answered',w,16,'muted');const q=box(p,'Sample question',w,'VERTICAL',32,24,'surface');text(q,'QUESTION 3 OF 5',w-64,12,'brand');text(q,'Write 6/8 in lowest terms.',w-64,28,'ink','Bold');text(q,'Your answer: 3/4',w-64,24,'brand');text(q,'Design preview: answers are not submitted, stored or graded.',w-64,14,'muted');button(q,'Back to the lesson','Knowledge Detail',true);}
{const {p,w}=screen('My Progress',1440,5);heading(p,'MY PROGRESS / ELEMENTARY MATHEMATICS','Every step has a place in your story.','Progress belongs to a learning path, supported by learning and checkpoint records.',w);const row=box(p,'Sample progress',w,'HORIZONTAL',0,24);for(const [label,value,note] of [['Learned','12 / 30','Wider Elementary learning path'],['Mastered','8','Learning and checkpoint complete'],['Review','3','Consolidate this week']])instance(row,'StatCard',null,{Label:label,Value:value,Note:note},(w-48)/3);text(p,'Four weeks of steady growth',w,24,'ink','Bold');text(p,'Week 1: 1 point · Week 2: 2 points · Week 3: 4 points · Week 4: 5 points',w,18,'brand');text(p,'Your knowledge record',w,24,'ink','Bold');for(const id of ['arithmetic','equivalent-fractions','simplifying-fractions'])knowledge(p,route.nodes.find(n=>n.id===id),w);text(p,'Sample learning records. Research frontiers are not measured as a percentage of all mathematics.',w,14,'muted');}
{const {p,w}=screen('Feedback',1440,6);heading(p,'FEEDBACK','Help make mathematics more reliable.','Report an error, suggest a clearer explanation or share an idea.',w);const f=box(p,'Sample feedback',w,'VERTICAL',32,24,'surface');text(f,'Content error · Explanation · Feature idea',w-64,18,'brand');text(f,'Related content: Equivalent fractions · Sample version 1.0',w-64,16,'muted');text(f,'A short description',w-64,16,'ink','Bold');text(f,'Add a step-by-step view of splitting the whole.',w-64,18,'muted');text(f,'Received → Reviewed → Corrected → Published',w-64,18,'brand');text(f,'Design preview: this page does not send or save feedback.',w-64,14,'muted');button(f,'Back to the lesson','Knowledge Detail',true);}
{const {p,w}=screen('Mobile Knowledge Map',390,0,true);mapContent(p,w,true);}
{const {p,w}=screen('Mobile Knowledge Detail',390,1,true);lessonContent(p,w,true);}

for(const link of links){const n=await figma.getNodeByIdAsync(link.id);const destination=screens[link.to];await n.setReactionsAsync([{trigger:{type:'ON_CLICK'},actions:[{type:'NODE',destinationId:destination.id,navigation:'NAVIGATE',transition:null,resetScrollPosition:true}]}]);mutatedNodeIds.push(n.id);}
figma.viewport.scrollAndZoomIntoView([screens['Knowledge Map']]);
return {status:'created-needs-visual-review',createdNodeIds:[...new Set(createdNodeIds)],mutatedNodeIds:[...new Set(mutatedNodeIds)],screens:Object.fromEntries(Object.entries(screens).map(([name,n])=>[name,{id:n.id,width:n.width,height:n.height}])),domainCount:catalogue.domains.length,routePointCount:route.nodes.length,prototypeLinks:links.length};
