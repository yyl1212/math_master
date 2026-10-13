import {SafeMarkdown} from '@/features/reading/safe-markdown';import styles from '@/styles/notes.module.css';
export function NotePreview({body}:{body:string}){return <div className={styles.preview}><SafeMarkdown source={body} assets={[]}/></div>}
