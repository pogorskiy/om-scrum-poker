import { render } from 'preact';
import './tokens.css';
import './global.css';
import { App } from './app';

render(<App />, document.getElementById('app')!);
