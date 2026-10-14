import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api } from './api';

const GoEditor = lazy(() => import('./GoEditor'));

export type ProjectSummary = { id: string; title: string; rank: string; description: string };
type Project = ProjectSummary & {
  requirements: string[];
  milestones: { id: string; title: string; criteria: string[] }[];
  starter_files: Record<string, string>;
  public_examples: { args: string[]; stdout: string }[];
};
type Snapshot = { id: number; created_at: string };
type ProjectResult = {
  stdout?: string; stderr?: string; status?: string; exit_code?: number; passed?: boolean; manual_review_required?: boolean;
  checks?: { name: string; passed: boolean }[];
  milestones?: { id: string; automatic_passed: boolean; needs_review: boolean }[];
};
type Action = 'run' | 'test' | 'submit';

export default function ProjectWorkspace({ projectId, onBack }: { projectId: string; onBack: () => void }) {
  const [project, setProject] = useState<Project | null>(null);
  const [files, setFiles] = useState<Record<string, string>>({});
  const [activeFile, setActiveFile] = useState('main.go');
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [ready, setReady] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [saveState, setSaveState] = useState('Загружаем файлы проекта…');
  const [busy, setBusy] = useState<Action | 'save' | 'export' | 'restore' | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [result, setResult] = useState<ProjectResult | null>(null);
  const [resultAction, setResultAction] = useState<Action | null>(null);
  const [args, setArgs] = useState('[]');
  const [newFilename, setNewFilename] = useState('');
  const editVersion = useRef(0);

  const basePath = `/projects/${encodeURIComponent(projectId)}`;
  const refreshSnapshots = useCallback(async () => {
    const items = await api<Snapshot[]>(`${basePath}/snapshots`);
    setSnapshots(items);
  }, [basePath]);
  useEffect(() => {
    let current = true;
    Promise.all([api<Project>(basePath), api<{ files: Record<string, string> }>(`${basePath}/files`), api<Snapshot[]>(`${basePath}/snapshots`)])
      .then(([details, saved, history]) => {
        if (!current) return;
        const hasSavedFiles = Boolean(saved.files && Object.keys(saved.files).length);
        const initial = hasSavedFiles ? saved.files : details.starter_files;
        setProject(details); setFiles(initial); setActiveFile(Object.keys(initial)[0] || 'main.go'); setSnapshots(history); setDirty(false); setSaveState(hasSavedFiles ? 'Сохранённая версия загружена.' : 'Загружены начальные файлы проекта.'); setReady(true);
      })
      .catch(e => { if (current) { setError(`Не удалось открыть проект: ${errorMessage(e)}`); setSaveState(''); setReady(true); } });
    return () => { current = false; };
  }, [basePath]);

  const validationError = useMemo(() => validateProjectFiles(files), [files]);
  const saveFiles = useCallback(async () => {
    if (!Object.keys(files).length || validationError) {
      setSaveState(validationError || 'В проекте должен остаться хотя бы один файл.');
      return false;
    }
    setBusy('save'); setError(''); setSaveState('Сохраняем файлы…');
    const versionAtSave = editVersion.current;
    try {
      await api(`${basePath}/files`, { method: 'PUT', body: JSON.stringify({ files }) });
      const unchanged = editVersion.current === versionAtSave;
      if (unchanged) setDirty(false);
      setSaveState(unchanged ? 'Файлы сохранены.' : 'Эта версия сохранена; новые изменения ещё не сохранены.');
      try { await refreshSnapshots(); } catch (e) { setNotice(`Файлы сохранены, но история версий не обновилась: ${errorMessage(e)}`); }
      return unchanged;
    } catch (e) {
      setSaveState(`Не сохранено: ${errorMessage(e)}`);
      return false;
    } finally { setBusy(null); }
  }, [basePath, files, refreshSnapshots, validationError]);

  useEffect(() => {
    if (!ready || !dirty || busy !== null || saveState.startsWith('Не сохранено:') || validationError || !Object.keys(files).length) return;
    const timer = window.setTimeout(() => { void saveFiles(); }, 800);
    return () => window.clearTimeout(timer);
  }, [busy, dirty, files, ready, saveFiles, saveState, validationError]);

  useEffect(() => {
    const retryWhenOnline = () => { if (dirty && saveState.startsWith('Не сохранено:')) void saveFiles(); };
    window.addEventListener('online', retryWhenOnline);
    return () => window.removeEventListener('online', retryWhenOnline);
  }, [dirty, saveFiles, saveState]);

  const editFile = (contents: string) => { editVersion.current += 1; setFiles(current => ({ ...current, [activeFile]: contents })); setDirty(true); setSaveState('Есть несохранённые изменения.'); setResult(null); setResultAction(null); };
  const addFile = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const filename = newFilename.trim();
    if (!filename || !validProjectFilename(filename)) { setError('Имя файла должно быть в корне проекта и заканчиваться на .go, go.mod, .md или .json.'); return; }
    if (filename in files) { setError('Файл с таким именем уже есть в проекте.'); return; }
    if (Object.keys(files).length >= 16) { setError('В проекте может быть не больше 16 файлов.'); return; }
    const contents = filename.endsWith('.go') ? 'package main\n\n' : filename === 'go.mod' ? 'module cli-toolkit\n\ngo 1.27.2\n' : filename.endsWith('.json') ? '{}\n' : '';
    editVersion.current += 1; setFiles(current => ({ ...current, [filename]: contents })); setActiveFile(filename); setNewFilename(''); setDirty(true); setSaveState('Есть несохранённые изменения.'); setError(''); setNotice(`Файл ${filename} добавлен в черновик.`);
  };
  const deleteFile = () => {
    if (!(activeFile in files)) return;
    if (activeFile.endsWith('.go') && Object.keys(files).filter(name => name.endsWith('.go')).length <= 1) {
      setError('Нельзя удалить последний Go-файл проекта.'); return;
    }
    if (activeFile === 'README.md' || activeFile === 'go.mod') { setError(`${activeFile} обязателен для проекта и не может быть удалён.`); return; }
    const remaining = { ...files }; delete remaining[activeFile];
    editVersion.current += 1; setFiles(remaining); setActiveFile(Object.keys(remaining)[0] || ''); setDirty(true); setSaveState('Есть несохранённые изменения.'); setResult(null); setResultAction(null); setError(''); setNotice(`Файл ${activeFile} удалён из черновика.`);
  };
  const perform = async (action: Action) => {
    if (validationError) { setError(validationError); return; }
    let parsedArgs: string[] = [];
    if (action === 'run') {
      try {
        const parsed: unknown = JSON.parse(args);
        if (!Array.isArray(parsed) || parsed.some(value => typeof value !== 'string')) throw new Error('Ожидается JSON-массив строк.');
        parsedArgs = parsed;
      } catch (e) { setError(`Аргументы заданы неверно: ${errorMessage(e)}`); return; }
    }
    setBusy(action); setError(''); setNotice(''); setResult(null); setResultAction(action);
    try {
      const response = await api<ProjectResult>(`${basePath}/${action}`, { method: 'POST', body: JSON.stringify(action === 'run' ? { files, args: parsedArgs } : { files }) });
      setResult(response);
      if (action === 'submit' && response.passed) setNotice('Автоматические проверки пройдены. Качество README и история Git требуют отдельного ревью; проект и ранг не считаются полностью подтверждёнными.');
    } catch (e) { setError(`Действие не выполнено: ${errorMessage(e)}`); }
    finally { setBusy(null); }
  };
  const restore = async (snapshot: Snapshot) => {
    if (dirty) return;
    setBusy('restore'); setError(''); setNotice('');
    try {
      const response = await api<{ files: Record<string, string> }>(`${basePath}/restore`, { method: 'POST', body: JSON.stringify({ snapshot_id: snapshot.id }) });
      editVersion.current += 1; setFiles(response.files); setActiveFile(Object.keys(response.files)[0] || ''); setDirty(false); setSaveState('Версия восстановлена.'); setResult(null); setResultAction(null); setNotice(`Восстановлена версия от ${formatDate(snapshot.created_at)}.`);
      try { await refreshSnapshots(); } catch (e) { setNotice(`Версия восстановлена, но история не обновилась: ${errorMessage(e)}`); }
    } catch (e) { setError(`Не удалось восстановить версию: ${errorMessage(e)}`); }
    finally { setBusy(null); }
  };
  const exportZip = async () => {
    if (validationError) { setError(validationError); return; }
    if (dirty && !await saveFiles()) return;
    setBusy('export'); setError(''); setNotice('');
    try {
      const response = await fetch(`/api${basePath}/export`, { credentials: 'same-origin' });
      if (!response.ok) throw new Error(await responseError(response));
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a'); link.href = url; link.download = `${projectId}.zip`; link.click(); window.setTimeout(() => URL.revokeObjectURL(url), 1000);
      setNotice('Архив проекта скачан.');
    } catch (e) { setError(`Не удалось экспортировать проект: ${errorMessage(e)}`); }
    finally { setBusy(null); }
  };

  if (!ready) return <main className="page"><p className="empty-state">Загружаем файлы, требования и историю проекта…</p></main>;
  if (!project) return <main className="page project-page"><button className="back-link" onClick={onBack}>‹ К проектам</button><p className="empty-state" role="alert">{error || 'Проект недоступен.'}</p></main>;
  return <main className="page project-page">
    <div className="project-toolbar"><button className="back-link" onClick={onBack}>‹ К проектам</button><span className="project-rank">{project.rank}</span><span className="project-toolbar-spacer" /><span className="project-save-status" role="status">{saveState.startsWith('Не сохранено:') ? saveState : dirty ? 'Есть несохранённые изменения' : saveState}</span><button className="outline-button" onClick={() => void saveFiles()} disabled={!dirty || busy !== null || Boolean(validationError)}>Сохранить</button></div>
    {(error || notice) && <div className={error ? 'project-message project-error' : 'project-message project-notice'} role={error ? 'alert' : 'status'}>{error || notice}<button onClick={() => { setError(''); setNotice(''); }} aria-label="Закрыть сообщение">×</button></div>}
    <div className="project-heading"><div><p className="eyebrow">ПРАКТИЧЕСКИЙ ПРОЕКТ</p><h1>{project.title}</h1><p>{project.description}</p></div><button className="outline-button" onClick={() => void exportZip()} disabled={busy !== null || Boolean(validationError)}>Скачать ZIP</button></div>
    <div className="project-meta-grid"><section className="project-info"><h2>Требования</h2><p className="prerequisite-note">Полный курс Go core для ранга Junior ещё не опубликован. Для этого проекта нужны основы командной строки и тестирования.</p><p className="project-limits-note">Используй только стандартную библиотеку Go. Лимиты: 16 файлов и 128 КиБ всего; 64 КиБ Go-кода; до 8 JSON-файлов и 64 КиБ данных. TestMain здесь не поддерживается: запускаются обычные тесты пакетов.</p><ul>{project.requirements.map(item => <li key={item}>{item}</li>)}</ul><details><summary>Этапы проекта</summary><div className="milestone-list">{project.milestones.map(milestone => <section key={milestone.id}><h3>{milestone.title}</h3><ul>{milestone.criteria.map(criterion => <li key={criterion}>{criterion}</li>)}</ul></section>)}</div></details><details className="examples-details"><summary>Публичные примеры</summary>{project.public_examples.map((example, index) => <div className="public-example" key={index}><p>Аргументы: <code>{JSON.stringify(example.args)}</code></p><pre tabIndex={0}>{example.stdout}</pre></div>)}</details><section className="snapshot-list"><h2>Сохранённые версии</h2>{dirty && <p>Сохрани изменения перед восстановлением версии.</p>}{snapshots.length ? snapshots.map(snapshot => <div className="snapshot-row" key={snapshot.id}><span>{formatDate(snapshot.created_at)}</span><button className="quiet-button" disabled={dirty || busy !== null} onClick={() => void restore(snapshot)}>Восстановить</button></div>) : <p>Сохранённых версий пока нет.</p>}</section></section>
      <section className="project-editor"><div className="project-editor-head"><h2>Файлы проекта</h2><span>{Object.keys(files).length} / 16</span></div><div className="project-workbench"><aside className="project-file-rail" aria-label="Файлы проекта"><ul>{Object.keys(files).map(filename => <li key={filename}><button className={filename === activeFile ? 'project-file active' : 'project-file'} aria-current={filename === activeFile ? 'page' : undefined} onClick={() => setActiveFile(filename)}>{filename}</button></li>)}</ul><form className="add-project-file" onSubmit={addFile}><label htmlFor="new-project-file">Добавить файл</label><input id="new-project-file" value={newFilename} onChange={event => setNewFilename(event.target.value)} placeholder="например, util.go" maxLength={64} disabled={isProjectLocked(busy)} /><button className="quiet-button" disabled={isProjectLocked(busy) || Object.keys(files).length >= 16}>Добавить</button></form><button className="delete-project-file" onClick={deleteFile} disabled={isProjectLocked(busy) || !activeFile}>Удалить выбранный файл</button></aside><div className="project-code-area"><div className="project-active-file">{activeFile || 'Файл не выбран'} <span>{activeFile.endsWith('.go') ? 'Go' : 'Текстовый файл'}</span></div>{activeFile && (activeFile.endsWith('.go') ? <Suspense fallback={<div className="editor-canvas editor-loading">Загружаем редактор…</div>}><GoEditor readOnly={isProjectLocked(busy)} value={files[activeFile] || ''} onChange={editFile} /></Suspense> : <textarea readOnly={isProjectLocked(busy)} className="project-text-editor" aria-label={`Содержимое файла ${activeFile}`} value={files[activeFile] || ''} onChange={event => editFile(event.target.value)} spellCheck={activeFile.endsWith('.md')} />)}</div></div>{validationError && <p className="project-validation" role="alert">{validationError}</p>}<div className="project-actions"><button className="outline-button" onClick={() => void perform('run')} disabled={busy !== null || Boolean(validationError)}>{busy === 'run' ? 'Запускаем…' : 'Запустить'}</button><button className="outline-button" onClick={() => void perform('test')} disabled={busy !== null || Boolean(validationError)}>{busy === 'test' ? 'Запускаем тесты…' : 'Запустить тесты'}</button><button className="primary-button" onClick={() => void perform('submit')} disabled={busy !== null || Boolean(validationError)}>{busy === 'submit' ? 'Проверяем проект…' : 'Проверить проект'}</button></div><label className="args-label" htmlFor="project-args">Аргументы запуска (JSON-массив строк)</label><input id="project-args" className="project-args" value={args} onChange={event => setArgs(event.target.value)} placeholder='["--help"]' disabled={isProjectLocked(busy)} /><ProjectOutput result={result} action={resultAction} /></section>
    </div>
  </main>;
}

function ProjectOutput({ result, action }: { result: ProjectResult | null; action: Action | null }) {
  if (!result) return <section className="project-output"><h2>Вывод и проверки</h2><p>Запусти приложение или его тесты, чтобы увидеть реальный результат.</p></section>;
  return <section className="project-output"><h2>{action === 'submit' ? 'Автоматическая оценка проекта' : action === 'test' ? 'Результат запуска ваших тестов' : 'Результат запуска приложения'}</h2>{action === 'test' && <p>Эти тесты проверяют код проекта и сами по себе не подтверждают ранг.</p>}{result.status && <p>Состояние: {result.status}; код завершения: {result.exit_code ?? '—'}</p>}{action === 'submit' && result.passed !== undefined && <p>{result.passed ? 'Автоматические проверки пройдены.' : 'Некоторые автоматические проверки не пройдены.'} Результат не присваивает учебный ранг.{result.manual_review_required && ' Качество README и история Git требуют отдельного ревью; проект не считается полностью подтверждённым.'}</p>}{result.milestones?.map(milestone => <p className="project-check" key={milestone.id}><span>{milestone.automatic_passed ? '✓' : '·'}</span>{milestoneName(milestone.id)}: {milestone.automatic_passed ? 'автоматические проверки пройдены' : 'автоматические проверки не пройдены'}{milestone.needs_review && ' · требуется отдельное ревью'}</p>)}{result.checks?.map(check => <p className="project-check" key={check.name}><span>{check.passed ? '✓' : '×'}</span>{check.name}</p>)}{result.stdout && <><h3>Стандартный вывод</h3><pre tabIndex={0}>{result.stdout}</pre></>}{result.stderr && <><h3>Ошибки и диагностика</h3><pre tabIndex={0}>{result.stderr}</pre></>}</section>;
}

function validProjectFilename(filename: string) {
  return filename === 'go.mod' || filename === 'README.md' || /^[A-Za-z][A-Za-z0-9_-]*\.(go|json)$/.test(filename);
}
function validateProjectFiles(files: Record<string, string>) {
  const names = Object.keys(files);
  if (names.length < 1) return 'В проекте должен остаться хотя бы один файл.';
  if (names.length > 16) return 'В проекте может быть не больше 16 файлов.';
  if (names.some(name => !validProjectFilename(name))) return 'Разрешены только корневые файлы .go, go.mod, README.md и .json.';
  const encoder = new TextEncoder();
  const totalBytes = Object.values(files).reduce((total, contents) => total + encoder.encode(contents).byteLength, 0);
  const goFiles = names.filter(name => name.endsWith('.go'));
  const goBytes = goFiles.reduce((total, name) => total + encoder.encode(files[name]).byteLength, 0);
  const jsonFiles = names.filter(name => name.endsWith('.json'));
  const jsonBytes = jsonFiles.reduce((total, name) => total + encoder.encode(files[name]).byteLength, 0);
  if (totalBytes > 128 * 1024) return 'Общий размер файлов не должен превышать 128 КиБ.';
  if (goBytes > 64 * 1024) return 'Общий размер Go-файлов не должен превышать 64 КиБ.';
  if (jsonFiles.length > 8) return 'В проекте может быть не больше 8 JSON-файлов.';
  if (jsonBytes > 64 * 1024) return 'Общий размер JSON-файлов не должен превышать 64 КиБ.';
  if (!goFiles.some(name => !name.endsWith('_test.go'))) return 'Нужен хотя бы один Go-файл с исходным кодом; одного *_test.go недостаточно.';
  if (!('README.md' in files)) return 'Добавь обязательный файл README.md.';
  if (!files['README.md'].trim()) return 'README.md должен содержать текст.';
  if (!('go.mod' in files)) return 'Добавь обязательный файл go.mod.';
  if (files['go.mod'].replace(/\r\n/g,'\n').split('\n').map(line=>line.trim().split(/\s+/).join(' ')).filter(Boolean).join('\n') !== 'module cli-toolkit\ngo 1.27.2') return 'В go.mod должны остаться только module cli-toolkit и go 1.27.2 без внешних зависимостей.';
  return '';
}
function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? value : new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}
function milestoneName(id: string) { return ({ cli: 'CLI', json: 'JSON', release: 'Сборка и выпуск' } as Record<string, string>)[id] || id; }
function errorMessage(error: unknown) { return error instanceof Error ? error.message : 'неизвестная ошибка'; }
function isProjectLocked(busy: Action | 'save' | 'export' | 'restore' | null) { return busy === 'run' || busy === 'test' || busy === 'submit' || busy === 'export' || busy === 'restore'; }
async function responseError(response: Response) {
  const body = await response.text().catch(() => '');
  try { const parsed: unknown = JSON.parse(body); if (parsed && typeof parsed === 'object' && 'error' in parsed && typeof parsed.error === 'string') return parsed.error; } catch { /* Plain text is already readable. */ }
  return body || `Сервер ответил ${response.status}`;
}
