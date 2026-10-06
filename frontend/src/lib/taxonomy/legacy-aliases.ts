// 人工可审查的导航范围，不替代任何知识点的冻结主题归属，不执行单目标重定向。
export const legacyDomainTopics:Record<string,readonly string[]>={
 "foundations-logic":["msc-03","msc-00"],
 "elementary-mathematics":["msc-00","msc-11","msc-15","msc-26","msc-51","msc-97"],
 "linear-algebra":["msc-15"],
 "abstract-algebra":["msc-06","msc-08","msc-12","msc-13","msc-16","msc-17","msc-18","msc-19","msc-20","msc-22"],
 "number-theory":["msc-11"],
 "mathematical-analysis":["msc-26","msc-28","msc-33","msc-40","msc-41","msc-42","msc-43","msc-44"],
 "complex-analysis":["msc-30","msc-31","msc-32"],
 "functional-analysis":["msc-46","msc-47"],
 "geometry":["msc-14","msc-51","msc-52","msc-53"],
 "topology":["msc-54","msc-55","msc-57","msc-58"],
 "discrete-combinatorics":["msc-05","msc-06","msc-68"],
 "probability-stochastic":["msc-60"],
 "statistics":["msc-62"],
 "differential-equations":["msc-34","msc-35","msc-37","msc-39","msc-45"],
 "numerical-optimization":["msc-41","msc-49","msc-65","msc-90"],
 "applied-interdisciplinary":["msc-68","msc-70","msc-74","msc-76","msc-78","msc-80","msc-81","msc-82","msc-83","msc-85","msc-86","msc-90","msc-91","msc-92","msc-93","msc-94"]
};
export const resolveLegacyDomain=(id:string)=>Object.hasOwn(legacyDomainTopics,id)?legacyDomainTopics[id]:null;
