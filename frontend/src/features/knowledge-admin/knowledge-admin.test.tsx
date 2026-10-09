import {it,expect,vi} from 'vitest';import {render,screen} from '@testing-library/react';import {UploadKnowledge} from './upload';
vi.mock('@/lib/i18n/provider',()=>({useUiI18n:()=>({t:(key:string)=>key})}));
it('offers direct upload and saving with no review or version controls',()=>{render(<UploadKnowledge onComplete={()=>{}}/>);expect(screen.getByRole('button',{name:'managed.uploadPublish'})).toBeDisabled();expect(screen.queryByText(/review|version/i)).toBeNull();expect(document.querySelector('input[type=file][multiple]')).not.toBeNull()});
