import {useEffect, useMemo, useState} from 'react';
import {ChooseOutput, DiscoverProfiles, ExportProfiles} from '../wailsjs/go/main/App';
import {WindowSetSystemDefaultTheme} from '../wailsjs/runtime/runtime';
import {LanguageMode, messages, resolveLocale} from './i18n';
import './App.css';

type Profile = { id: string; browser: string; name: string; database: string; engine: string };
type Status = { kind: 'scanning' | 'found' | 'snapshots' | 'exported' | 'error'; count?: number; output?: string; message?: string };

function initialLanguage(): LanguageMode {
    const saved = localStorage.getItem('language');
    return saved === 'en-US' || saved === 'zh-CN' ? saved : 'auto';
}

export default function App() {
    const [language, setLanguage] = useState<LanguageMode>(initialLanguage);
    const [systemLanguageVersion, setSystemLanguageVersion] = useState(0);
    const locale = useMemo(() => resolveLocale(language), [language, systemLanguageVersion]);
    const text = messages(locale);
    const [profiles, setProfiles] = useState<Profile[]>([]);
    const [selected, setSelected] = useState<Set<string>>(new Set());
    const [output, setOutput] = useState('');
    const [password, setPassword] = useState('');
    const [confirmation, setConfirmation] = useState('');
    const [status, setStatus] = useState<Status>({kind: 'scanning'});
    const [busy, setBusy] = useState(false);
    const groups = useMemo(() => profiles.reduce((result, profile) => {
        const entries = result.get(profile.browser) ?? [];
        entries.push(profile);
        result.set(profile.browser, entries);
        return result;
    }, new Map<string, Profile[]>()), [profiles]);

    useEffect(() => {
        document.documentElement.lang = locale;
    }, [locale]);

    useEffect(() => {
        const update = () => setSystemLanguageVersion(value => value + 1);
        window.addEventListener('languagechange', update);
        WindowSetSystemDefaultTheme();
        void refresh();
        return () => window.removeEventListener('languagechange', update);
    }, []);

    async function refresh() {
        setStatus({kind: 'scanning'});
        try {
            const found = await DiscoverProfiles(locale) as Profile[];
            setProfiles(found);
            setSelected(current => new Set([...current].filter(id => found.some(profile => profile.id === id))));
            setStatus({kind: 'found', count: found.length});
        } catch (error) {
            setStatus({kind: 'error', message: String(error)});
        }
    }

    function changeLanguage(value: LanguageMode) {
        setLanguage(value);
        if (value === 'auto') localStorage.removeItem('language');
        else localStorage.setItem('language', value);
    }

    function toggle(id: string) {
        setSelected(current => {
            const next = new Set(current);
            next.has(id) ? next.delete(id) : next.add(id);
            return next;
        });
    }

    async function choose() {
        try {
            const path = await ChooseOutput(locale);
            if (path) setOutput(path);
        } catch (error) {
            setStatus({kind: 'error', message: String(error)});
        }
    }

    async function run() {
        if (password !== confirmation) return;
        setBusy(true);
        setStatus({kind: 'snapshots'});
        try {
            const result = await ExportProfiles([...selected], output, password, locale);
            setStatus({kind: 'exported', count: result.recordCount, output: result.output});
            setPassword('');
            setConfirmation('');
        } catch (error) {
            setStatus({kind: 'error', message: String(error)});
        } finally {
            setBusy(false);
        }
    }

    function statusText() {
        switch (status.kind) {
            case 'scanning': return text.scanning;
            case 'found': return text.found(status.count ?? 0);
            case 'snapshots': return text.snapshots;
            case 'exported': return text.exported(status.count ?? 0, status.output ?? '');
            case 'error': return text.error(status.message ?? '');
        }
    }

    function historyName(name: string) {
        if (name === 'Default') return text.defaultHistory;
        if (name === 'Manual') return text.manualHistory;
        const numbered = /^Profile\s+(.+)$/.exec(name);
        if (numbered) return `${text.history} ${numbered[1]}`;
        return `${text.history} · ${name}`;
    }

    const mismatch = password !== confirmation;
    const weak = password.length > 0 && password.length < 12;

    return <main>
        <header>
            <div>
                <p className="eyebrow">{text.appName}</p>
                <h1>{text.title}</h1>
                <p>{text.privacy}</p>
            </div>
            <div className="header-actions">
                <label className="language">
                    <span>{text.language}</span>
                    <select value={language} onChange={event => changeLanguage(event.target.value as LanguageMode)} disabled={busy}>
                        <option value="auto">{text.automatic}</option>
                        <option value="en-US">{text.english}</option>
                        <option value="zh-CN">{text.chinese}</option>
                    </select>
                </label>
                <button className="secondary" onClick={refresh} disabled={busy}>{text.rescan}</button>
            </div>
        </header>
        <section className="card">
            <h2>{text.profiles}</h2>
            {profiles.length === 0 ? <p className="empty">{text.noProfiles}</p> : [...groups].map(([browser, items]) =>
                <div className="group" key={browser}>
                    <h3>{browser}</h3>
                    {items.map(profile => <label className="profile" key={profile.id}>
                        <input type="checkbox" checked={selected.has(profile.id)} onChange={() => toggle(profile.id)}/>
                        <span><strong>{historyName(profile.name)}</strong><small>{profile.database}</small></span>
                    </label>)}
                </div>)}
        </section>
        <section className="card output">
            <div><h2>{text.output}</h2><p>{output || text.chooseOutput}</p></div>
            <button className="secondary" onClick={choose} disabled={busy}>{text.choose}</button>
        </section>
        <section className="card">
            <h2>{text.protection}</h2>
            <div className="password-grid">
                <label><span>{text.password}</span><input type="password" value={password} onChange={event => setPassword(event.target.value)} disabled={busy} autoComplete="new-password"/></label>
                <label><span>{text.confirmPassword}</span><input type="password" value={confirmation} onChange={event => setConfirmation(event.target.value)} disabled={busy} autoComplete="new-password"/></label>
            </div>
            <p className={`hint ${mismatch ? 'error' : weak ? 'warning' : ''}`}>
                {mismatch ? text.passwordMismatch : weak ? text.weakPassword : text.passwordHint}
            </p>
        </section>
        <footer>
            <p className="status" role="status" aria-live="polite">{statusText()}</p>
            <button className="primary" onClick={run} disabled={busy || selected.size === 0 || !output || mismatch}>
                {busy ? text.exporting : text.exportProfiles(selected.size)}
            </button>
        </footer>
    </main>;
}
