import {it,expect} from 'vitest';
import {render,screen} from '@testing-library/react';
import {QualificationLink} from './qualification-link';
import {qualification} from '@/lib/learning/test-fixtures';
import {id} from '@/lib/correction/test-fixtures';
it('links a current corrected qualification separately from its original assessment',()=>{render(<QualificationLink qualification={{...qualification(),correctionId:id}}/>);expect(screen.getByRole('link',{name:'View corrected qualification'})).toHaveAttribute('href','/corrections/'+id);expect(screen.getByRole('link',{name:'Original assessment'})).toHaveAttribute('href','/assessments/'+id+'/result')});
it('keeps original qualifications without an invented correction link',()=>{render(<QualificationLink qualification={qualification()}/>);expect(screen.queryByRole('link',{name:'View corrected qualification'})).toBeNull();expect(screen.getByRole('link',{name:'Original assessment'})).toBeVisible()});
