import {Children,type ReactNode} from 'react';
import styles from '@/styles/page-ui.module.css';
const hasContent=(node:ReactNode)=>Children.toArray(node).some(child=>child!=='');
/** Layout only: callers retain ownership of language, links and data. */
export function PageHeading({title,description,meta,actions}:{title:ReactNode;description?:ReactNode;meta?:ReactNode;actions?:ReactNode}){
 return <header className={styles.heading}><h1>{title}</h1>{hasContent(description)&&<div className={styles.description}>{description}</div>}{hasContent(meta)&&<div className={styles.meta}>{meta}</div>}{hasContent(actions)&&<div className={styles.actions}>{actions}</div>}</header>;
}
