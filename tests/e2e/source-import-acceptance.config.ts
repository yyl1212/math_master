import base from './playwright.config';
import {defineConfig} from '../../frontend/node_modules/@playwright/test/index.js';
export default defineConfig({...base,testMatch:'source-import-acceptance.ts'});
