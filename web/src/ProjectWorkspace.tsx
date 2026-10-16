import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { api, getProjectReviewHistory, getProjectReviewPacket, submitProjectReview, type ProjectReviewAttestation, type ProjectReviewHistoryItem, type ProjectReviewResult } from './api';

const GoEditor = lazy(() => import('./GoEditor'));

export type ProjectSummary = { id: string; title: string; rank: string; description: string };
type Project = ProjectSummary & {
  execution_available?: boolean;
  requirements: string[];
  milestones: { id: string; title: string; criteria: string[] }[];
  starter_files: Record<string, string>;
  public_examples: { args: string[]; stdout: string }[];
};
type Snapshot = { id: number; created_at: string };
type ProjectResult = {
  stdout?: string; stderr?: string; status?: string; exit_code?: number; passed?: boolean; manual_review_required?: boolean;
  manual_review_labels?: string[];
  project_evidence?: {
    submission_id: string; evaluation_id: string; project_version: string; criteria_hash: string;
    artifact_hash: string; executed_source_hash: string; checker_version: string; checker_digest: string;
    runtime_policy_hash: string; rubric_version: string; rubric_hash: string; checks_hash: string;
    evaluation_status: string; exit_code: number;
  };
  checks?: { name: string; passed: boolean }[];
  milestones?: { id: string; automatic_passed: boolean; needs_review: boolean }[];
};
type Action = 'run' | 'test' | 'submit';
type ReviewState = { error?: string; success?: ProjectReviewResult };
type ReviewEvidence = { submission_id: string; evaluation_id: string; project_version: string; artifact_hash: string; evaluation_status: string };

export default function ProjectWorkspace({ projectId, onBack }: { projectId: string; onBack: () => void }) {
  const [project, setProject] = useState<Project | null>(null);
  const [files, setFiles] = useState<Record<string, string>>({});
  const [activeFile, setActiveFile] = useState('main.go');
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [ready, setReady] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [saveState, setSaveState] = useState('Загружаем файлы проекта…');
  const [busy, setBusy] = useState<Action | 'save' | 'export' | 'restore' | 'review' | null>(null);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [result, setResult] = useState<ProjectResult | null>(null);
  const [resultAction, setResultAction] = useState<Action | null>(null);
  const [reviewState, setReviewState] = useState<ReviewState>({});
  const [reviewHistory, setReviewHistory] = useState<ProjectReviewHistoryItem[]>([]);
  const [reviewEvidence, setReviewEvidence] = useState<ReviewEvidence | null>(null);
  const [args, setArgs] = useState('[]');
  const [newFilename, setNewFilename] = useState('');
  const editVersion = useRef(0);

  const basePath = `/projects/${encodeURIComponent(projectId)}`;
  const reviewPointerKey = `open-go-academy:${projectId}:last-review-evidence`;
  const refreshReviewHistory = useCallback(async (submissionId: string) => {
    const response = await getProjectReviewHistory(projectId, submissionId);
    setReviewHistory(response.reviews);
  }, [projectId]);
  useEffect(() => {
    let current = true;
    setReviewEvidence(null); setReviewHistory([]);
    const rawPointer = window.localStorage.getItem(reviewPointerKey);
    if (!rawPointer) return () => { current = false; };
    let pointer: unknown;
    try { pointer = JSON.parse(rawPointer); } catch { window.localStorage.removeItem(reviewPointerKey); return () => { current = false; }; }
    if (!pointer || typeof pointer !== 'object' || !('submission_id' in pointer) || !('evaluation_id' in pointer) || typeof pointer.submission_id !== 'string' || typeof pointer.evaluation_id !== 'string' || !pointer.submission_id || !pointer.evaluation_id) {
      window.localStorage.removeItem(reviewPointerKey); return () => { current = false; };
    }
    const { submission_id: submissionId, evaluation_id: evaluationId } = pointer;
    getProjectReviewPacket(projectId, submissionId, evaluationId).then(async packet => {
      if (packet.project_id !== projectId || packet.submission_id !== submissionId || packet.evaluation_id !== evaluationId || packet.evaluation_status !== 'automatic_pass') {
        throw new Error('review pointer does not match a passing server packet');
      }
      if (!current) return;
      setReviewEvidence({ submission_id: packet.submission_id, evaluation_id: packet.evaluation_id, project_version: packet.project_version, artifact_hash: packet.artifact_hash, evaluation_status: packet.evaluation_status });
      const history = await getProjectReviewHistory(projectId, submissionId);
      if (current) setReviewHistory(history.reviews);
    }).catch(() => {
      if (!current) return;
      setReviewEvidence(null); setReviewHistory([]); window.localStorage.removeItem(reviewPointerKey);
    });
    return () => { current = false; };
  }, [projectId, reviewPointerKey]);
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

  const isP2 = projectId === 'p2';
  const isP3 = projectId === 'p3';
  const isP4 = projectId === 'p4';
  const isP5 = projectId === 'p5';
  const usesProjectRunner = isP2 || isP3 || isP4 || isP5;
  const hasGoSum = isP2 || isP3;
  const maxFiles = isP4 ? 24 : usesProjectRunner ? 48 : 16;
  const lockedModule = usesProjectRunner && (activeFile === 'go.mod' || hasGoSum && activeFile === 'go.sum');
  const validationError = useMemo(() => validateProjectFiles(files, projectId, project?.starter_files), [files, projectId, project]);
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
    if (!filename || !validProjectFilename(filename, projectId)) { setError(usesProjectRunner ? 'Используй безопасное имя из латинских букв, допустимое расширение и вложенность до трёх частей.' : 'Имя файла должно быть в корне проекта и заканчиваться на .go или .json.'); return; }
    if (filename in files) { setError('Файл с таким именем уже есть в проекте.'); return; }
    if (Object.keys(files).length >= maxFiles) { setError(`В проекте может быть не больше ${maxFiles} файлов.`); return; }
    const contents = filename.endsWith('.go') ? 'package main\n\n' : filename === 'go.mod' ? (project?.starter_files['go.mod'] || 'module cli-toolkit\n\ngo 1.27.2\n') : filename.endsWith('.json') ? '{}\n' : '';
    editVersion.current += 1; setFiles(current => ({ ...current, [filename]: contents })); setActiveFile(filename); setNewFilename(''); setDirty(true); setSaveState('Есть несохранённые изменения.'); setError(''); setNotice(`Файл ${filename} добавлен в черновик.`);
  };
  const deleteFile = () => {
    if (!(activeFile in files)) return;
    if (activeFile.endsWith('.go') && Object.keys(files).filter(name => name.endsWith('.go')).length <= 1) {
      setError('Нельзя удалить последний Go-файл проекта.'); return;
    }
    if (activeFile === 'README.md' || activeFile === 'go.mod' || (hasGoSum && activeFile === 'go.sum')) { setError(`${activeFile} обязателен для проекта и не может быть удалён.`); return; }
    const remaining = { ...files }; delete remaining[activeFile];
    editVersion.current += 1; setFiles(remaining); setActiveFile(Object.keys(remaining)[0] || ''); setDirty(true); setSaveState('Есть несохранённые изменения.'); setResult(null); setResultAction(null); setError(''); setNotice(`Файл ${activeFile} удалён из черновика.`);
  };
  const perform = async (action: Action) => {
    if (validationError) { setError(validationError); return; }
    let parsedArgs: string[] = [];
    if (action === 'run' && !usesProjectRunner) {
      try {
        const parsed: unknown = JSON.parse(args);
        if (!Array.isArray(parsed) || parsed.some(value => typeof value !== 'string')) throw new Error('Ожидается JSON-массив строк.');
        parsedArgs = parsed;
      } catch (e) { setError(`Аргументы заданы неверно: ${errorMessage(e)}`); return; }
    }
    setBusy(action); setError(''); setNotice(''); setResult(null); setResultAction(action); setReviewState({});
    try {
      const response = await api<ProjectResult>(`${basePath}/${action}`, { method: 'POST', body: JSON.stringify(action === 'run' ? { files, args: parsedArgs } : { files }) });
      setResult(response);
      if (action === 'submit' && response.passed) {
        setNotice('Автоматические проверки пройдены. Ранг не присваивается; критерии отдельного ревью перечислены ниже.');
        if (response.project_evidence) {
          const evidence = response.project_evidence;
          setReviewEvidence(evidence);
          window.localStorage.setItem(reviewPointerKey, JSON.stringify({ submission_id: evidence.submission_id, evaluation_id: evidence.evaluation_id }));
        }
      }
    } catch (e) { setError(`Действие не выполнено: ${errorMessage(e)}`); }
    finally { setBusy(null); }
  };
  const downloadReviewPacket = async (evidence: ReviewEvidence) => {
    setBusy('review'); setReviewState({});
    try {
      const packet = await getProjectReviewPacket(projectId, evidence.submission_id, evidence.evaluation_id);
      if (packet.project_id !== projectId || packet.submission_id !== evidence.submission_id || packet.evaluation_id !== evidence.evaluation_id || packet.evaluation_status !== 'automatic_pass') throw new Error('Сервер вернул пакет, который не совпадает с сохранённым passing evidence.');
      const blob = new Blob([`${JSON.stringify(packet, null, 2)}\n`], { type: 'application/json;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${projectId}-${evidence.submission_id}-${evidence.evaluation_id}-review-packet.json`;
      link.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
      setNotice('Пакет для ручного ревью скачан. Передай его reviewer вместе с подписанным решением.');
    } catch (e) { setReviewState({ error: `Не удалось получить review packet: ${errorMessage(e)}` }); }
    finally { setBusy(null); }
  };
  const uploadProjectReview = async (evidence: ReviewEvidence, file: File) => {
    setBusy('review'); setReviewState({});
    try {
      const parsed: unknown = JSON.parse(await file.text());
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) throw new Error('Нужен JSON-объект подписанного решения.');
      const response = await submitProjectReview(projectId, evidence.submission_id, parsed as ProjectReviewAttestation);
      setReviewState({ success: response });
      window.localStorage.setItem(reviewPointerKey, JSON.stringify({ submission_id: evidence.submission_id, evaluation_id: evidence.evaluation_id }));
      try { await refreshReviewHistory(evidence.submission_id); }
      catch { /* The signed POST is durable; history can be refreshed after reload. */ }
    } catch (e) { setReviewState(current => ({ ...current, error: `Решение не принято: ${errorMessage(e)}` })); }
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
    <div className="project-meta-grid"><section className="project-info"><h2>Требования</h2><p className="prerequisite-note">{isP3 ? 'Перед TaskFlow изучи HTTP, SQL, транзакции, tenant-доступ и работу с pgx. Автоматическая проверка проекта сама по себе не присваивает Middle.' : isP2 ? 'Перед проектом изучи HTTP, SQL, аутентификацию и работу с pgx. Проверка не присваивает ранг.' : isP4 ? 'Для ScaleLab нужны P3, benchmark, profiling и чтение основных SLO-метрик. Автоматические тесты не подтверждают измерительную методику или Middle+ ранг.' : isP5 ? 'Для Capstone нужны Middle/Middle+, проектирование систем и эксплуатационные основы. Senior требует независимой аутентифицированной защиты; AI, автопроверка и самооценка её не заменяют.' : 'Для этого проекта нужны основы Go core, командной строки и тестирования из курса Junior.'}</p><p className="project-limits-note">{isP2 || isP3 ? 'Go 1.27.2, pgx и PostgreSQL 18.6. До 48 файлов, 512 КиБ всего и 256 КиБ Go-кода; отдельный файл до 128 КиБ. go.mod и go.sum закреплены. Проверка использует временную изолированную БД и не открывает сервер на компьютере. Dockerfile и Compose нужны только для локального экспорта.' : usesProjectRunner ? `Go 1.27.2 и только стандартная библиотека. Лимиты: ${maxFiles} файлов, 512 КиБ всего, 256 КиБ Go-кода, 128 КиБ на файл. go.mod закреплён сервером; проектные проверки выполняются в изолированном локальном runner.` : 'Используй только стандартную библиотеку Go. Лимиты: 16 файлов и 128 КиБ всего; 64 КиБ Go-кода; до 8 JSON-файлов и 64 КиБ данных. TestMain здесь не поддерживается: запускаются обычные тесты пакетов.'}</p><ul>{project.requirements.map(item => <li key={item}>{item}</li>)}</ul><details><summary>Этапы проекта</summary><div className="milestone-list">{project.milestones.map(milestone => <section key={milestone.id}><h3>{milestone.title}</h3><ul>{milestone.criteria.map(criterion => <li key={criterion}>{criterion}</li>)}</ul></section>)}</div></details><details className="examples-details"><summary>Публичные примеры</summary>{(project.public_examples || []).map((example, index) => <div className="public-example" key={index}><p>Аргументы: <code>{JSON.stringify(example.args)}</code></p><pre tabIndex={0}>{example.stdout}</pre></div>)}</details><section className="snapshot-list"><h2>Сохранённые версии</h2>{dirty && <p>Сохрани изменения перед восстановлением версии.</p>}{snapshots.length ? snapshots.map(snapshot => <div className="snapshot-row" key={snapshot.id}><span>{formatDate(snapshot.created_at)}</span><button className="quiet-button" disabled={dirty || busy !== null} onClick={() => void restore(snapshot)}>Восстановить</button></div>) : <p>Сохранённых версий пока нет.</p>}</section></section>
      <section className="project-editor"><div className="project-editor-head"><h2>Файлы проекта</h2><span>{Object.keys(files).length} / {maxFiles}</span></div><div className="project-workbench"><aside className="project-file-rail" aria-label="Файлы проекта"><ul>{Object.keys(files).map(filename => <li key={filename}><button className={filename === activeFile ? 'project-file active' : 'project-file'} aria-current={filename === activeFile ? 'page' : undefined} onClick={() => setActiveFile(filename)}>{filename}</button></li>)}</ul><form className="add-project-file" onSubmit={addFile}><label htmlFor="new-project-file">Добавить файл</label><input id="new-project-file" value={newFilename} onChange={event => setNewFilename(event.target.value)} placeholder="например, util.go" maxLength={64} disabled={isProjectLocked(busy)} /><button className="quiet-button" disabled={isProjectLocked(busy) || Object.keys(files).length >= maxFiles}>Добавить</button></form><button className="delete-project-file" onClick={deleteFile} disabled={isProjectLocked(busy) || !activeFile || lockedModule}>Удалить выбранный файл</button></aside><div className="project-code-area"><div className="project-active-file">{activeFile || 'Файл не выбран'} <span>{activeFile.endsWith('.go') ? 'Go' : 'Текстовый файл'}</span></div>{activeFile && (activeFile.endsWith('.go') ? <Suspense fallback={<div className="editor-canvas editor-loading">Загружаем редактор…</div>}><GoEditor readOnly={isProjectLocked(busy) || lockedModule} value={files[activeFile] || ''} onChange={editFile} /></Suspense> : <textarea readOnly={isProjectLocked(busy) || lockedModule} className="project-text-editor" aria-label={`Содержимое файла ${activeFile}`} value={files[activeFile] || ''} onChange={event => editFile(event.target.value)} spellCheck={activeFile.endsWith('.md')} />)}</div></div>{validationError && <p className="project-validation" role="alert">{validationError}</p>}<div className="project-actions"><button className="outline-button" onClick={() => void perform('run')} disabled={busy !== null || Boolean(validationError) || project.execution_available === false}>{busy === 'run' ? 'Запускаем…' : 'Запустить'}</button><button className="outline-button" onClick={() => void perform('test')} disabled={busy !== null || Boolean(validationError) || project.execution_available === false}>{busy === 'test' ? 'Запускаем тесты…' : 'Запустить тесты'}</button><button className="primary-button" onClick={() => void perform('submit')} disabled={busy !== null || Boolean(validationError) || project.execution_available === false}>{busy === 'submit' ? 'Проверяем проект…' : 'Проверить проект'}</button></div>{project.execution_available === false && <p role="status">Исполнитель недоступен. Можно редактировать, сохранять и экспортировать проект; запусти {usesProjectRunner ? 'make project-runner' : 'make runner'} для проверки.</p>}{!usesProjectRunner && <><label className="args-label" htmlFor="project-args">Аргументы запуска (JSON-массив строк)</label><input id="project-args" className="project-args" value={args} onChange={event => setArgs(event.target.value)} placeholder='["--help"]' disabled={isProjectLocked(busy)} /></>}<ProjectOutput result={result} action={resultAction} reviewState={reviewState} reviewEvidence={reviewEvidence} reviewHistory={reviewHistory} reviewBusy={busy === 'review'} onDownloadReview={downloadReviewPacket} onUploadReview={uploadProjectReview} /></section>
    </div>
  </main>;
}

function ProjectOutput({ result, action, reviewState, reviewEvidence, reviewHistory, reviewBusy, onDownloadReview, onUploadReview }: {
  result: ProjectResult | null; action: Action | null; reviewState: ReviewState; reviewEvidence: ReviewEvidence | null;
  reviewHistory: ProjectReviewHistoryItem[]; reviewBusy: boolean;
  onDownloadReview: (evidence: ReviewEvidence) => void;
  onUploadReview: (evidence: ReviewEvidence, file: File) => void;
}) {
  const evidence = (action === 'submit' && result?.passed ? result.project_evidence : null) ?? reviewEvidence;
  const reviewPanel = evidence ? <ManualReviewPanel evidence={evidence} reviewState={reviewState} reviewBusy={reviewBusy} onDownloadReview={onDownloadReview} onUploadReview={onUploadReview} /> : null;
  if (!result) return <section className="project-output"><h2>Вывод и проверки</h2><p>Запусти приложение или его тесты, чтобы увидеть реальный результат.</p>{reviewPanel}<ReviewHistoryList reviews={reviewHistory} /></section>;
  return <section className="project-output">
    <h2>{action === 'submit' ? 'Автоматическая оценка проекта' : action === 'test' ? 'Результат запуска ваших тестов' : 'Результат запуска приложения'}</h2>
    {action === 'test' && <p>Эти тесты проверяют код проекта и сами по себе не подтверждают ранг.</p>}
    {result.status && <p>Состояние: {result.status}; код завершения: {result.exit_code ?? '—'}</p>}
    {action === 'submit' && result.passed !== undefined && <p>{result.passed ? 'Автоматические проверки пройдены.' : 'Некоторые автоматические проверки не пройдены.'} Результат не присваивает учебный ранг.{result.manual_review_required && <> Для полного подтверждения требуется отдельное ревью: {result.manual_review_labels?.length ? result.manual_review_labels.join('; ') : 'качества README и истории Git'}.</>}</p>}
    {reviewPanel}
    <ReviewHistoryList reviews={reviewHistory} />
    {action === 'submit' && result.project_evidence && <details className="project-evidence-details"><summary>Сохранённое доказательство этой версии проекта</summary><dl><dt>Пакет</dt><dd><code>{result.project_evidence.submission_id}</code> · версия <code>{result.project_evidence.project_version}</code></dd><dt>Проверка</dt><dd><code>{result.project_evidence.evaluation_id}</code> · <code>{result.project_evidence.evaluation_status}</code>, код {result.project_evidence.exit_code}</dd><dt>Checker</dt><dd><code>{result.project_evidence.checker_version}</code></dd><dt>Хеш артефакта</dt><dd><code>{result.project_evidence.artifact_hash}</code></dd><dt>Хеш исполняемых файлов</dt><dd><code>{result.project_evidence.executed_source_hash}</code></dd><dt>Хеш проверок</dt><dd><code>{result.project_evidence.checks_hash}</code></dd><dt>Политика / rubric</dt><dd><code>{result.project_evidence.runtime_policy_hash}</code> · <code>{result.project_evidence.rubric_version}</code></dd></dl><p>Запись хранится отдельно от текущего workspace; автоматическое evidence не подтверждает ручное ревью и не присваивает ранг.</p></details>}
    {result.milestones?.map(milestone => <p className="project-check" key={milestone.id}><span>{milestone.automatic_passed ? '✓' : '·'}</span>{milestoneName(milestone.id)}: {milestone.automatic_passed ? 'автоматические проверки пройдены' : 'автоматические проверки не пройдены'}{milestone.needs_review && ' · требуется отдельное ревью'}</p>)}
    {result.checks?.map(check => <p className="project-check" key={check.name}><span>{check.passed ? '✓' : '×'}</span>{check.name}</p>)}
    {result.stdout && <><h3>Стандартный вывод</h3><pre tabIndex={0}>{result.stdout}</pre></>}
    {result.stderr && <><h3>Ошибки и диагностика</h3><pre tabIndex={0}>{result.stderr}</pre></>}
  </section>;
}

function ManualReviewPanel({ evidence, reviewState, reviewBusy, onDownloadReview, onUploadReview }: {
  evidence: ReviewEvidence; reviewState: ReviewState; reviewBusy: boolean;
  onDownloadReview: (evidence: ReviewEvidence) => void;
  onUploadReview: (evidence: ReviewEvidence, file: File) => void;
}) {
  return <section aria-labelledby="manual-review-heading"><h3 id="manual-review-heading">Подписанное ручное ревью</h3>
    <p>Проверенное сервером evidence: submission <code>{evidence.submission_id}</code>, evaluation <code>{evidence.evaluation_id}</code> (<code>{evidence.evaluation_status}</code>, версия <code>{evidence.project_version}</code>). Скачай пакет именно для этой пары, передай его reviewer, затем загрузи подписанное JSON-решение. В шаблоне выбери решение, укажи reviewer ID и комментарий до подписи. Сервер проверит allowlist оператора; если ключ reviewer не настроен, запрос будет отклонён.</p>
    <div className="project-actions"><button className="outline-button" onClick={() => onDownloadReview(evidence)} disabled={reviewBusy}>Скачать JSON review packet</button>
      <label>Загрузить подписанный JSON <input type="file" accept="application/json,.json" aria-label="Загрузить подписанное решение reviewer JSON" disabled={reviewBusy} onChange={event => { const file = event.currentTarget.files?.[0]; event.currentTarget.value = ''; if (file) onUploadReview(evidence, file); }} /></label></div>
    <p>Подпись подтверждает владение ключом из операторского allowlist. Личность и независимость reviewer зависят от внешней настройки и доверия оператору. Это решение само по себе не начисляет ранг.</p>
    {reviewState.error && <p role="alert">{reviewState.error}</p>}
    {reviewState.success && <div role="status"><p>Подписанное решение сохранено: <code>{reviewState.success.decision}</code>, key fingerprint <code>{reviewState.success.reviewer_key_fingerprint}</code>, review ID <code>{reviewState.success.review_id}</code>.</p><p>Fingerprint идентифицирует ключ, не человека; ранг автоматически не начислен.</p></div>}
  </section>;
}

function ReviewHistoryList({ reviews }: { reviews: ProjectReviewHistoryItem[] }) {
  if (!reviews.length) return null;
  return <details className="project-evidence-details"><summary>Сохранённые решения reviewer ({reviews.length})</summary><ul>{reviews.map(review => <li key={review.id}><strong>{review.decision}</strong> · reviewer key <code>{review.reviewer_id}</code> · fingerprint <code>{review.reviewer_key_fingerprint || 'legacy/unverified'}</code> · review ID <code>{review.id}</code><p>{review.comments || 'Комментарий не приложен.'}</p><small>{formatDate(review.created_at)} · подпись/attestation <code>{review.attestation_digest}</code></small></li>)}</ul><p>Fingerprint связывает решение с конкретным операторским ключом, но не подтверждает реальную личность или независимость reviewer и не начисляет ранг. История загружается с локального сервера после проверки указателя submission.</p></details>;
}

function validProjectFilename(filename: string, projectId = '') {
  const usesProjectRunner = ['p2', 'p3', 'p4', 'p5'].includes(projectId);
  if (usesProjectRunner) {
    const extensionAllowed = projectId === 'p2' || projectId === 'p3'
      ? /\.(go|sql|json|md|ya?ml)$/.test(filename)
      : projectId === 'p4' ? /\.(go|md)$/.test(filename) : /\.(go|md|json)$/.test(filename);
    const special = filename === 'go.mod' || ((projectId === 'p2' || projectId === 'p3') && ['go.sum', 'Dockerfile', '.env.example', '.gitignore'].includes(filename));
    return filename.split('/').length <= 3 && filename.split('/').every(part => part !== '.' && part !== '..' && /^[A-Za-z0-9._-]+$/.test(part)) && (extensionAllowed || special);
  }
  return filename === 'go.mod' || filename === 'README.md' || /^[A-Za-z][A-Za-z0-9_-]*\.(go|json)$/.test(filename);
}
function validateProjectFiles(files: Record<string, string>, projectId = '', starter?: Record<string,string>) {
  const usesProjectRunner = ['p2', 'p3', 'p4', 'p5'].includes(projectId);
  if (usesProjectRunner) {
    const names = Object.keys(files), bytes = (value: string) => new TextEncoder().encode(value).byteLength;
    const maxFiles = projectId === 'p4' ? 24 : 48;
    const hasGoSum = projectId === 'p2' || projectId === 'p3';
    if (!names.length || names.length > maxFiles) return `Допустимо от 1 до ${maxFiles} файлов.`;
    if (names.some(name => !validProjectFilename(name, projectId))) return 'Некорректное имя или расширение файла.';
    if (names.some(name => bytes(files[name]) > 128 * 1024)) return 'Файл превышает 128 КиБ.';
    if (names.reduce((sum,name) => sum + bytes(files[name]),0) > 512 * 1024) return 'Общий размер превышает 512 КиБ.';
    if (names.filter(name => name.endsWith('.go')).reduce((sum,name) => sum+bytes(files[name]),0) > 256*1024) return 'Go-код превышает 256 КиБ.';
    if (!names.some(name => name.endsWith('.go') && !name.endsWith('_test.go'))) return 'Добавь исходный Go-файл.';
    if (!files['README.md']?.trim()) return 'Добавь непустой README.md.';
    const requiredFiles = projectId === 'p4'
      ? ['main.go', 'aggregate.go', 'docs/project.md', 'docs/rubric.md']
      : projectId === 'p5'
	    ? ['main.go', 'queue.go', 'docs/project.md', 'docs/rubric.md', 'docs/threat-model.md', 'design/requirements.md', 'design/architecture-template.md', 'design/adr-template.md', 'design/operations-template.md', 'design/postmortem-template.md', 'design/threat-template.md']
        : [];
    if (requiredFiles.some(name => !(name in files))) return 'В проекте удалён обязательный исходный или rubric-файл.';
    if (starter && (files['go.mod'] !== starter['go.mod'] || hasGoSum && files['go.sum'] !== starter['go.sum'])) return hasGoSum ? 'go.mod и go.sum должны совпадать с закреплённой версией проекта.' : 'go.mod должен совпадать с закреплённой версией проекта.';
    return '';
  }
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
  if (!/^[\x20-\x7E\t\r\n]+$/.test(files['go.mod']) || files['go.mod'].replace(/\r\n/g,'\n').includes('\r')) return 'В go.mod разрешены обычные пробелы, табы и перевод строки.';
  if (files['go.mod'].replace(/\r\n/g,'\n').split('\n').map(line=>line.trim().split(/\s+/).join(' ')).filter(Boolean).join('\n') !== 'module cli-toolkit\ngo 1.27.2') return 'В go.mod должны остаться только module cli-toolkit и go 1.27.2 без внешних зависимостей.';
  return '';
}
function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? value : new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' }).format(date);
}
function milestoneName(id: string) { return ({ cli: 'CLI', json: 'JSON', release: 'Сборка и выпуск' } as Record<string, string>)[id] || id; }
function errorMessage(error: unknown) { return error instanceof Error ? error.message : 'неизвестная ошибка'; }
function isProjectLocked(busy: Action | 'save' | 'export' | 'restore' | 'review' | null) { return busy === 'run' || busy === 'test' || busy === 'submit' || busy === 'export' || busy === 'restore' || busy === 'review'; }
async function responseError(response: Response) {
  const body = await response.text().catch(() => '');
  try { const parsed: unknown = JSON.parse(body); if (parsed && typeof parsed === 'object' && 'error' in parsed && typeof parsed.error === 'string') return parsed.error; } catch { /* Plain text is already readable. */ }
  return body || `Сервер ответил ${response.status}`;
}
