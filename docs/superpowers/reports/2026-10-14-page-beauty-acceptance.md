# 第三阶段页面美感验收

## 结果与适用范围

采用已确认的安静学术阅读风格；范围以方案A01—A25和26个旧入口为准。仅提交MR，不合并或部署。数据库和全部测试数据来自本阶段专属合成库，不使用生产备份、账号或私人笔记。

## 可重现条件

- 起点master bf8e02c；分支codex/page-beauty。Node24.17.0、Go1.27.1（CGO_ENABLED=0）、PG17.11、锁定Playwright1.63.0。
- Chromium当前三文件；Firefox/WebKit各桌面1280×900及手机390×844核心冒烟，额外320检查。
- 当前矩阵中英文320/390/1280，另有768/1440、阅读1099/1100与比对800/801、长标题/长公式/空态/管理员展开态。
- 原49条浏览器命令按原workflow顺序和前端模式串行运行，每条≤540秒；原Playwright480秒、单worker、零重试。Go测试5分钟限时。
- 原始T1基线不重测；只补身份元数据，删除新增role后与T1深比较完全相等。改后证据经过采集路径及页面就绪复核，使用同一轮PNG/JSON；截图遮挡所有input/textarea。

## 自动验证

|项目|结果|
|---|---|
|前端全单元|649/649（147文件）|
|类型检查/生产构建|通过|
|API类型生成|通过，生成文件零差异|
|npm生产依赖审计|0漏洞|
|兼容与验证工具|103/103（T7原102项加最终CI上下文回归）|
|内容导入工具|22/22|
|Go vet与原夹具/测试工具|通过；夹具命令首次缺本机测试库变量，补显式合成库后131.9秒通过|
|固定旧前端构建|原工具生成68文件；原fixture/manifest未改|
|当前Chromium矩阵|118/118（公开48、学习42、管理28）|
|Firefox/WebKit核心|Mac20/20、Ubuntu24 arm64 20/20；四项目各5例|
|原49条浏览器命令|49/49，386次用例；最慢124.43秒，含原重复命令；源码8b145d2，最终共享样式修复后另重跑当前118、跨浏览器Mac20+Linux20、受影响原入口36|
|全分支独立审查|一次新鲜上下文gpt-6-astra审查；Critical 0、Important 1已修复验证、Minor 0|
|最终修复后原入口回归|当前36/36（form-controls + site-streamline + ui-language-streamline）；旧前端catalogue + reading 16/16|
|原生200%仅文字放大|14/14；真实textZoom2/fullZoom1/DPR1，320/390/1280|

此前两处标题修复的原入口补验为26项（site18+language8），最终共享导航修复后扩大到36项（另加原form-controls10项）。首轮25通过/1失败：历史筛选长流程抵达账户页时耗尽30秒用例预算，失败快照中Sign out已出现；独立原场景两视口2/2在12.9秒通过，随后完整原26项通过。未对该问题改产品、原测试或任何预算；保留首轮失败，不能据此声称已定位一个产品缺陷。

最终补跑两轮被本机合盖休眠中断：首次学习组32通过/1失败/9未跑，次轮公开组41通过/1失败/4未跑，均不计通过。电源事件与异常用时对齐；受影响登录/比对场景醒后原预算2/2在7.1秒通过。此前先完成相同标题修复字节的公开46、学习42及管理28；最终共享导航修复后重新完整串行采集118项，不复用此前画面作为最终结果。没有调整超时或重试。最终凭据明确各组日志与源码范围。

## 资源预算

同T1：所有.next/static CSS去重后gzip9求和；五代表路由初始HTML中的Client脚本去重后gzip9求和。预算CSS增量≤10240B，JS每路由增量≤8192B；工具包含新增真实大CSS/JS拒绝、漏路由、越界资产和基线篡改负例。最终CSS为14224→17062B，增量2838B；首页/阅读/笔记/复习/管理JS增量分别104/923/923/819/665B，均通过。准确资产明细见[改后摘要](../../operations/evidence/page-beauty/after.json)。

最终同一目录生成241份PNG/JSON，每份检查真实身份、唯一h1和页面宽度；均无页面横溢。此前A24在切语言瞬间分次采样出现标题空数组，新增断言实际失败后改为原5秒预算内等待原子DOM快照，最终完整118矩阵重跑通过。T1基线保持，未用旧图覆盖新结果。

## 原生缩放与人工视觉复核

隔离Chromium的tabs.setZoom/getZoom实际读数：200%时innerWidth640/DPR2，400%时innerWidth320/DPR4，外窗1280；未使用CSS zoom模拟。11项记录包括首页/正文不同滚动位置、长公式左右箭头、12次Tab焦点、原生select在forced-colors下appearance:auto，以及reduced-motion下动画none/滚动auto。最终脚本通过并视觉复核，见[记录](../../operations/evidence/page-beauty/manual-browser.json)、[400%正文](../../operations/evidence/page-beauty/manual-reading-4x.png)和[高对比](../../operations/evidence/page-beauty/manual-forced-colors-reduced-motion.png)。原生缩放下默认全页截图会裁切物理视口，因此最终采用CDP captureScreenshot完整1280物理像素、临时DOM遮挡输入后保存；不将裁切图片当作通过证据。高对比及reduced-motion通过浏览器媒体偏好模拟与截图复核；iOS真机不可用，未覆盖软键盘、选择和Safari真机，不将WebKit冒烟等同真机。

原生缩放方法参考[Chrome tabs.setZoom](https://developer.chrome.com/docs/extensions/reference/api/tabs#method-setZoom)和[Playwright隔离扩展会话](https://playwright.dev/docs/chrome-extensions)。仅加载本次原创本机测试扩展，不接触个人浏览器资料，也不写入产品。

## 200%仅文字放大补验与修复

独立审查指出整页缩放不能代替文字放大，评为Important并接受。用隔离Firefox155.0的原生`textZoom=2`、`fullZoom=1`运行14项；没有在页面注入字体覆盖。每项实际body字体16→32、DPR1、真实页面视口320/390/1280保持、文档宽度不超视口；macOS外窗最小500时仅在浏览器chrome层限定内容区域320，页面实际innerWidth仍为320。记录见[原生文字放大](../../operations/evidence/page-beauty/text200.json)。

补验实际发现首页英文h1宽344、阅读两段英文h2宽337的横溢。两个永久浏览器用例均先在桌面/手机项目见RED，再分别只为`.hero h1`和`.readingBody h2`加`overflow-wrap:anywhere`；原文本、公式及业务状态不变。目标4/4转GREEN，原生14/14、最终当前118/118、Firefox/WebKit Mac20/20及Linux20/20及受影响原入口36/36再次通过，修复后649单元/类型/固定旧构建通过。本次只有一个最终修复闭环，按Native要求不派二次审查。

覆盖双语窄页头、登录错误、全部管理长表单字段、验证对话框Tab循环/取消恢复焦点/理由保留、阅读正文与笔记长公式左右箭头。焦点检查实际active、可见/未遮挡、横向不裁；放大后超过一屏的多行编辑器要求焦点轮廓上缘和至少44px编辑区可见，逐字段滚动检查，不要求整个长文本域同时塞进一屏。[阅读320](../../operations/evidence/page-beauty/text200-reading-320.png)、[对话框320](../../operations/evidence/page-beauty/text200-dialog-320.png)、[公式键盘](../../operations/evidence/page-beauty/text200-formula-keyboard.png)已人工复核；洋红色为截图输入遮挡。

原生控制使用[Mozilla Marionette](https://firefox-source-docs.mozilla.org/remote/marionette/Intro.html)与[官方Python客户端](https://firefox-source-docs.mozilla.org/python/marionette_driver.html)，仅本机隔离浏览器测试；产品依赖和锁文件未变。永久自动用例只将目标标题字号加倍以防止同一换行缺陷回归，原生全页文字放大证据单独保留，不混淆两种验收。

## 新视觉workflow启动校验

首次远端run38572203933在创建job前失败；actionlint1.7.7精确指出第19行job级env不能使用runner。新增实际workflow回归先RED，再将采集目录改为github.workspace/page-beauty-capture、上传读取同一env路径；6项CI用例和独立语法校验GREEN，最终全工具103/103。原workflow/预算/重试均未改。上下文限制依据[GitHub官方规则](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts#context-availability)。临时检查器校验官方发布SHA256后仅用于scratch，不增加产品依赖。

## Linux较宽字体的页头兼容补验

准确5abdf92的远端视觉CI（run38574354115）实际运行后公开44通过/2失败：390英文管理员页头194.375px>184px。本机以Verdana及通用monospace均精确复现同一高度；逐元素诊断定位导航三项被flex:1强制均分114px，较宽文字让导航行高58.875px，账户工具区58.5px，合计194.375px。

新增永久较宽回退字体行为用例在两个项目实际RED；只为原手机`.site-header nav a`追加`flex:1 1 auto`，按原文字自然宽度分配，未改变字体、文字、44px最小目标或184px上限。目标8/8 GREEN；再次完成原生文字14项、Chrome11项、118当前全矩阵、Firefox/WebKit Mac20与Linux20项，以及含原控件的当前36项、旧前端16项原回归。全部画面和资源使用该最终生产构建重新采集。该CI缺陷与前述文字放大、workflow上下文修复均在最终同一修复验证闭环内，不派第二次审查。

## Linux WebKit原生主题选框补验

准确f1074af的CI公开/学习/管理矩阵已过，但Ubuntu24 amd64 WebKit的管理列表320检查出现页面342，跨引擎14过/2失败（job115901328115）；该版不计全绿。本机宽字体下WebKit可复现345：三个选框边框都在41—279，第三个长提示仍让label内部宽304并传播到页面。裁剪隐形文件input及设置零最小网格轨道均无效，排除这两项假设；只约束主题select的内部文字后页面恢复320。

新增永久窄屏宽字体用例先在两个WebKit项目RED（345>321），最小修复为`.topic-picker-grid select:not([multiple])`的收起态内容约束与省略显示，四项目GREEN。保留完整原生option文本、选中值、原生弹出、焦点、48px尺寸及全部事件；长选中标签需打开下拉查看完整内容，见第11项实施裁定。没有裁剪标题/正文/导航或隐藏页面横溢。

完整Mac20/20、[官方Ubuntu24 arm64浏览器20/20](../../operations/evidence/page-beauty/linux-browser.json)，最终当前118、原36+旧16、649单元/类型/构建/103工具及原生文字14/整页偏好键盘11再次通过。原生文字放大的长表单已逐一聚焦所有可见可用input/select/textarea，disabled/hidden单独计数。使用[官方远程浏览器方法](https://playwright.dev/docs/docker#remote-connection)，版本与仓库1.63匹配，原配置/预算保持；浏览器运行在Ubuntu24 arm64，本机应用与合成库保持，不能冒充GitHub amd64，后者由最终准确HEAD CI确认。Mac与Linux采集目录独立，完整Mac241画面按最后修复重采。

## 兼容与发布边界

后端、API、迁移、生成类型、业务状态机/客户端、依赖锁、原workflow/配置、固定夹具及部署均字节保持。匿名笔记最新10条、会员全部分享、管理员审批、撤回再修改、私人隔离、四学习状态、五复习桶、两个历史游标、揭示与自评条件使用原实现和原安全回归。

本地完整原回归与最终修复回归的源码范围分别记录在[原49命令凭据](../../operations/evidence/page-beauty/original-regressions.json)和[最终凭据](../../operations/evidence/page-beauty/final-regressions.json)；禁改区核对见[范围摘要](../../operations/evidence/page-beauty/scope-review.json)。准确最新提交CI以MR检查页面为准，创建后在当前会话核对全部检查和准确SHA；本报告不以旧提交CI代替本分支。视觉测试只覆盖列出的语料和浏览器，不声称完整WCAG认证。


主题选择器高对比补验：Chromium在320宽与monospace字体下模拟forced-colors，原生appearance:auto、background-image:none、48px尺寸及焦点均保持；Space打开/Escape取消后97-XX选值保持，65项原生option保留，页面320无横溢。记录仅保存选项数量/摘要及公开的当前标签，见[补验摘要](../../operations/evidence/page-beauty/topic-select-forced-colors.json)，不冒充Windows真机。
