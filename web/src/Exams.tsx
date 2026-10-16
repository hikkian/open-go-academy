import { lazy, Suspense, useCallback, useEffect, useRef, useState } from 'react';
import { api, ApiError, type ExamAttempt, type ExamSummary, type RankEvidenceRequirement, type PublicExam, type ExamSubmitResult } from './api';
const GoEditor = lazy(() => import('./GoEditor'));

const tokenKey = (id: string) => `go-academy-exam-token:${id}`;
type Draft = { answers: Record<string, string>; codes: Record<string, string> };

export default function Exams({ onRankAwarded }: { onRankAwarded: () => Promise<void> }) {
  const [exams, setExams] = useState<ExamSummary[]>([]);
  const [history, setHistory] = useState<ExamAttempt[]>([]);
  const [exam, setExam] = useState<PublicExam | null>(null);
  const [attempt, setAttempt] = useState<ExamAttempt | null>(null);
  const [draft, setDraft] = useState<Draft>({ answers: {}, codes: {} });
  const [result, setResult] = useState<ExamSubmitResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [saving, setSaving] = useState('');
  const [error, setError] = useState('');
  const [conflict, setConflict] = useState(false);
  const attemptRef = useRef<ExamAttempt | null>(null);
  const draftRef = useRef<Draft>({ answers: {}, codes: {} });
  const draftOwnerRef = useRef('');
  const draftGenerationRef = useRef(0);
  const savedGenerationRef = useRef(0);
  const saveInFlightRef = useRef<{ id: string; promise: Promise<ExamAttempt> } | null>(null);

  const setCurrentAttempt = (next: ExamAttempt | null) => { attemptRef.current = next; setAttempt(next); };
  const replaceDraft = (next: Draft, owner: string, markSaved: boolean) => {
    draftRef.current = next; draftOwnerRef.current = owner; draftGenerationRef.current += 1;
    if (markSaved) savedGenerationRef.current = draftGenerationRef.current;
    setDraft(next);
  };
  const editDraft = (change: (current: Draft) => Draft) => {
    const next = change(draftRef.current); draftRef.current = next; draftGenerationRef.current += 1; setDraft(next);
  };

  const refreshHistory = useCallback(async () => {
    const [catalog, attempts] = await Promise.all([api<ExamSummary[]>('/exams'), api<ExamAttempt[]>('/exams/attempts')]);
    setExams(catalog); setHistory(attempts);
  }, []);
  useEffect(() => { let live = true; setLoading(true); refreshHistory().catch(e => { if (live) setError(`Не удалось загрузить экзамены: ${message(e)}`); }).finally(() => { if (live) setLoading(false); }); return () => { live = false; }; }, [refreshHistory]);

  const loadAttempt = async (id: string, knownExam?: PublicExam) => {
    setBusy(true); setError(''); setResult(null);
    try {
      const record=await api<ExamAttempt>(`/exams/attempts/${encodeURIComponent(id)}`);
      const publicExam=knownExam || await api<PublicExam>(`/exams/${encodeURIComponent(record.exam_id)}?version=${encodeURIComponent(record.version)}`);
      setCurrentAttempt(record); setExam(publicExam);
      let answers: Record<string, string> = {}; let codes: Record<string, string> = {};
      try { answers = record.answers_json ? JSON.parse(record.answers_json) : {}; } catch { /* Keep a usable empty draft if old stored data is malformed. */ }
      try { codes = record.codes_json ? JSON.parse(record.codes_json) : {}; } catch { /* Keep a usable empty draft if old stored data is malformed. */ }
      replaceDraft({ answers, codes }, record.id, true); setConflict(false); setSaving('');
      if (record.status !== 'active') {
        const parsedChecks = parseChecks(record.checks_json);
        setResult({ attempt: record, score: record.score ?? 0, passed: Boolean(record.passed), checks: parsedChecks, rank_awarded: false, rank: publicExam.rank, trust_notice: '' });
      }
    } catch (e) { setError(`Не удалось открыть попытку: ${message(e)}`); }
    finally { setBusy(false); }
  };

  const start = async (id: string) => {
    setBusy(true); setError('');
    try {
      const publicExam = await api<PublicExam>(`/exams/${encodeURIComponent(id)}`);
      const created = await api<ExamAttempt & { attempt_token: string }>(`/exams/${encodeURIComponent(id)}/attempts`, { method: 'POST', body: '{}' });
      if (created.revision !== 0) throw new Error('Сервер вернул некорректную версию новой попытки.');
      localStorage.setItem(tokenKey(created.id), created.attempt_token);
      await loadAttempt(created.id, publicExam);
      await refreshHistory();
    } catch (e) { setError(`Не удалось начать экзамен: ${message(e)}`); }
    finally { setBusy(false); }
  };

  const persistDraft = useCallback(async (quiet = false): Promise<ExamAttempt | undefined> => {
    const target = attemptRef.current;
    if (!target || target.status !== 'active' || draftOwnerRef.current !== target.id) return;
    const token = localStorage.getItem(tokenKey(target.id));
    if (!token) { setSaving(''); if (!quiet) setError('Ключ этой попытки не найден в браузере. Продолжить её нельзя; начни новую попытку.'); return; }
    if (conflict) return;
    try {
      while (draftOwnerRef.current === target.id) {
        const current = attemptRef.current;
        if (!current || current.id !== target.id || current.status !== 'active') return;
        const generation = draftGenerationRef.current;
        if (savedGenerationRef.current === generation) return current;
        if (conflict) return;
        if (saveInFlightRef.current) {
          if (saveInFlightRef.current.id !== target.id) { await saveInFlightRef.current.promise.catch(() => undefined); continue; }
          await saveInFlightRef.current.promise;
          continue;
        }
        const snapshot = draftRef.current;
        const revision = current.revision;
        setSaving('Сохраняем ответы…');
        const promise = api<{ saved: true; attempt: ExamAttempt }>(`/exams/attempts/${encodeURIComponent(target.id)}/draft`, { method: 'PUT', body: JSON.stringify({ attempt_token: token, revision, answers: snapshot.answers, codes: snapshot.codes }) }).then(response => response.attempt);
        saveInFlightRef.current = { id: target.id, promise };
        let updated: ExamAttempt;
        try { updated = await promise; }
        finally { if (saveInFlightRef.current?.promise === promise) saveInFlightRef.current = null; }
        if (updated.revision !== revision + 1) throw new Error('Сервер вернул некорректную версию черновика.');
        if (attemptRef.current?.id === target.id) {
          setCurrentAttempt(updated);
          savedGenerationRef.current = generation;
          setSaving('Черновик сохранён');
          setConflict(false);
        }
      }
      return attemptRef.current?.id === target.id ? attemptRef.current : undefined;
    } catch (e) {
      if (attemptRef.current?.id === target.id) {
        if (e instanceof ApiError && e.status === 409) { setConflict(true); setSaving('Есть более новая версия на сервере'); setError('Черновик изменён в другой вкладке. Твои текущие ответы сохранены на экране. Загрузи серверную версию, чтобы продолжить.'); }
        else setSaving(`Не сохранено: ${message(e)}`);
      }
      return;
    }
  }, [conflict]);
  useEffect(() => {
    if (!attempt || attempt.status !== 'active' || draftOwnerRef.current !== attempt.id) return;
    const timer = window.setTimeout(() => { void persistDraft(true); }, 650);
    return () => window.clearTimeout(timer);
  }, [draft, attempt?.id, attempt?.status, persistDraft]);

  const reloadServerDraft = async () => {
    const target = attemptRef.current;
    if (!target) return;
    setBusy(true); setError('');
    try {
      const serverAttempt = await api<ExamAttempt>(`/exams/attempts/${encodeURIComponent(target.id)}`);
      if (attemptRef.current?.id !== target.id) return;
      let answers: Record<string, string> = {}; let codes: Record<string, string> = {};
      try { answers = serverAttempt.answers_json ? JSON.parse(serverAttempt.answers_json) : {}; } catch { /* Keep a usable empty draft if stored JSON is malformed. */ }
      try { codes = serverAttempt.codes_json ? JSON.parse(serverAttempt.codes_json) : {}; } catch { /* Keep a usable empty draft if stored JSON is malformed. */ }
      setCurrentAttempt(serverAttempt); replaceDraft({ answers, codes }, serverAttempt.id, true); setConflict(false); setSaving('Серверная версия загружена');
    } catch (e) { setError(`Не удалось загрузить серверную версию: ${message(e)}`); }
    finally { setBusy(false); }
  };

  const submit = async () => {
    const currentAttempt = attemptRef.current;
    if (!currentAttempt) return;
    const token = localStorage.getItem(tokenKey(currentAttempt.id));
    if (!token) { setError('Ключ попытки не найден в браузере. Отправить её нельзя; начни новую попытку.'); return; }
    setBusy(true); setError(''); setSaving('Проверяем ответы…');
    try {
      const flushed = await persistDraft();
      if (!flushed || conflict) return;
      const latest = attemptRef.current;
      if (!latest || latest.id !== currentAttempt.id) return;
      const finalDraft = draftRef.current;
      const submitted = await api<ExamSubmitResult>(`/exams/attempts/${encodeURIComponent(latest.id)}/submit`, { method: 'POST', body: JSON.stringify({ attempt_token: token, revision: flushed.revision, answers: finalDraft.answers, codes: finalDraft.codes }) });
      setCurrentAttempt(submitted.attempt); setResult(submitted); localStorage.removeItem(tokenKey(latest.id));
      await refreshHistory();
      if (submitted.rank_awarded) await onRankAwarded();
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) { setConflict(true); setSaving('Есть более новая версия на сервере'); setError('Попытку изменили в другой вкладке. Она не была отправлена с устаревшими ответами. Загрузи серверную версию, чтобы продолжить.'); }
      else setError(`Не удалось завершить экзамен: ${message(e)}`);
    }
    finally { setBusy(false); }
  };

  const cancel = async () => {
    if (!attempt || !window.confirm('Отменить эту попытку? Черновик будет удалён.')) return;
    setBusy(true); setError('');
    try {
      await api(`/exams/attempts/${encodeURIComponent(attempt.id)}/cancel`, { method: 'POST', body: '{}' });
      localStorage.removeItem(tokenKey(attempt.id)); setCurrentAttempt(null); setExam(null); setResult(null); replaceDraft({ answers: {}, codes: {} }, '', true); setConflict(false); await refreshHistory();
    } catch (e) { setError(`Не удалось отменить попытку: ${message(e)}`); }
    finally { setBusy(false); }
  };

  const tokenAvailable = attempt ? Boolean(localStorage.getItem(tokenKey(attempt.id))) : false;
  const editableAttempt = attempt?.status === 'active' && tokenAvailable && !busy && !conflict;
  if (loading) return <main className="page exam-page"><p className="empty-state">Загружаем экзамены…</p></main>;
  if (exam && attempt) return <main className="page exam-page">
    <div className="exam-toolbar"><button className="back-link" onClick={() => { setExam(null); setAttempt(null); setResult(null); setError(''); }}>‹ К экзаменам</button><span>{exam.rank} · версия {exam.version}</span><span className="exam-save" role="status">{attempt.status === 'active' ? saving : 'Попытка завершена'}</span></div>
    <div className="page-heading"><h1>{exam.title}</h1><p>{exam.description}</p><p className="exam-duration">Ориентировочно {exam.estimated_minutes} минут. Черновик автоматически сохраняется на сервере.</p></div>
    {error && <p className="exam-error" role="alert">{error}</p>}
    {attempt.status === 'active' && !tokenAvailable && <p className="rank-trust-note" role="status">Ключ попытки в этом браузере не найден. Ответы доступны для просмотра, но продолжить, сохранить или отправить попытку нельзя.</p>}
    {attempt.status === 'active' && conflict && <div className="exam-conflict" role="alert"><p>В другой вкладке сохранена более новая версия. Твои ответы на этой странице сохранены локально.</p><button className="outline-button" disabled={busy} onClick={() => void reloadServerDraft()}>Загрузить серверную версию</button></div>}
    {result ? <section className="exam-result" aria-live="polite"><p className="eyebrow">РЕЗУЛЬТАТ ПОПЫТКИ</p><h2>{result.attempt.status === 'completed' ? result.passed ? 'Критерии экзамена выполнены' : 'Экзамен пока не пройден' : result.attempt.status === 'cancelled' ? 'Попытка отменена' : 'Попытка прервана'}</h2>{result.attempt.status === 'completed' && <p className="exam-score">{result.score}%</p>}{result.rank_awarded && <p className="exam-award">Подтверждён ранг {result.rank}.</p>}{result.trust_notice && <p>{result.trust_notice}</p>}{result.checks.length > 0 && <ul className="exam-checks">{result.checks.map(check => <li key={check.id} className={check.passed ? 'check-pass' : 'check-fail'}><span aria-hidden="true">{check.passed ? '✓' : '×'}</span>{check.name}{check.detail && <small>{check.detail}</small>}</li>)}</ul>}</section> : <>
      {exam.questions.map((question, index) => <fieldset className="exam-question" key={question.id} disabled={!editableAttempt}><legend><span>{index + 1}</span>{question.prompt}</legend><div>{question.options.map(option => <label className="exam-option" key={option.id}><input type="radio" name={question.id} value={option.id} checked={draft.answers[question.id] === option.id} onChange={() => editDraft(current => ({ ...current, answers: { ...current.answers, [question.id]: option.id } }))} />{option.text}</label>)}</div></fieldset>)}
      <section className="exam-tasks"><h2>Практические задания</h2><p>Редактор сохраняет твой код. Запуск и проверка выполняются сервером при отправке экзамена.</p>{exam.tasks.map(task => <article className="exam-task" key={task.id}><div><span className="exam-task-kind">{task.kind === 'debugging' ? 'Отладка' : 'Написание кода'}</span><h3>{task.title}</h3><p>{task.description}</p></div>{task.public_examples.map((item, i) => <details key={i} className="exam-example"><summary>Публичный пример {i + 1}</summary><p>Ввод: <code>{item.input || 'пустой'}</code></p><p>Ожидаемый результат: <code>{item.output || 'пустой'}</code></p></details>)}<Suspense fallback={<div className="editor-canvas editor-loading">Загружаем редактор…</div>}><div className="exam-code"><GoEditor scrollLabel={`Прокрутка кода: ${task.title}`} value={draft.codes[task.id] ?? task.starter_code} readOnly={!editableAttempt} onChange={value => editDraft(current => ({ ...current, codes: { ...current.codes, [task.id]: value } }))} /></div></Suspense></article>)}</section>
      {attempt.status === 'active' && <div className="exam-actions">{tokenAvailable && !conflict && <button className="quiet-button" disabled={busy} onClick={() => void persistDraft()}>{saving.startsWith('Не сохранено:') ? 'Повторить сохранение' : 'Сохранить сейчас'}</button>}<span /> <button className="quiet-button" disabled={busy} onClick={() => void cancel()}>Отменить попытку</button>{tokenAvailable && <button className="primary-button" disabled={busy || conflict} onClick={() => void submit()}>{busy ? 'Проверяем…' : 'Завершить и проверить'}</button>}</div>}
      {attempt.status !== 'active' && <p className="rank-trust-note">Попытка завершена. Ответы доступны только для чтения.</p>}
    </>}
  </main>;

  return <main className="page exam-page"><div className="page-heading"><p className="eyebrow">ПРОВЕРКА ПО КРИТЕРИЯМ КУРСА</p><h1>Экзамены</h1><p>Экзамен проверяет знания и практические навыки. Сам факт участия не подтверждает ранг.</p></div>
    {error && <p className="exam-error" role="alert">{error}</p>}
    {busy && <p className="empty-state">Открываем попытку…</p>}
    {exams.length ? exams.map(available=><section className="exam-catalog-item" key={`${available.id}:${available.version}`}>
      <div className="exam-catalog-rank">{available.rank}</div><div><h2>{available.title}</h2><p>{available.description}</p><small>{available.estimated_minutes} минут · {available.criteria.length} критериев · версия {available.version}</small></div>
      <div className="exam-catalog-action">{available.eligible ? <button className="primary-button" disabled={busy} onClick={() => void start(available.id)}>Начать экзамен</button> : <button className="outline-button" disabled title="Сначала выполни предварительные условия ниже">Пока недоступен</button>}</div>
      {!available.eligible && available.missing_prerequisites.length > 0 && <details className="exam-prereq"><summary>Сначала выполни предварительные условия ({available.missing_prerequisites.length})</summary><ul>{available.missing_prerequisites.map(id=><li key={id}>{prerequisiteTitle(id)}</li>)}</ul></details>}
      {available.rank_evidence && <details className="exam-prereq"><summary>Проектные и курсовые доказательства {available.rank_evidence.policy_version ? `(политика ${available.rank_evidence.policy_version})` : ''}</summary>
        {available.rank_evidence.status === 'unavailable' ? <p>{rankEvidenceReason(available.rank_evidence.reason)}</p> : <>
          <ul>{(available.rank_evidence.requirements || []).map(item=><li key={`${item.kind}:${item.requirement_id}`}><strong>{item.status === 'verified' ? 'Подтверждено' : 'Пока не подтверждено'}:</strong> {rankRequirementTitle(item)}{item.reason ? ` · ${rankEvidenceReason(item.reason, available.rank)}` : ''}</li>)}</ul>
          {(available.rank_evidence.requirements || []).some(item => item.kind === 'manual_review') && <p className="rank-trust-note">Ручное ревью относится к конкретной отправленной версии проекта и её прошедшей автоматической оценке. Для Senior (P5) нужны решения двух разных разрешённых ключей проверяющих. Отпечаток ключа подтверждает владение ключом, но сам по себе не подтверждает личность проверяющего или независимость проверок.</p>}
        </>}
        {available.rank_evidence.policy_hash && <small>Версия criteria закреплена сервером: <code>{available.rank_evidence.policy_hash}</code></small>}
      </details>}
    </section>) : <p className="empty-state">Опубликованных экзаменов пока нет.</p>}
    <section className="exam-history"><h2>История попыток</h2>{history.length ? <ul>{history.map(item => { const canResume = item.status === 'active' && localStorage.getItem(tokenKey(item.id)); return <li key={item.id}><div><strong>{exams.find(exam=>exam.id===item.exam_id)?.title || item.exam_id}</strong><span>{attemptStatus(item.status)}{item.status === 'completed' && item.score !== undefined ? ` · ${item.score}%` : ''}</span><small>{item.created_at ? new Date(item.created_at).toLocaleString('ru-RU') : ''}</small></div><button className="outline-button" onClick={() => void loadAttempt(item.id)}>{canResume ? 'Продолжить' : item.status === 'active' ? 'Посмотреть' : 'Результат'}</button></li>; })}</ul> : <p>Попыток пока нет. Когда будешь готов, начни экзамен выше.</p>}</section>
    <p className="rank-trust-note">Каждый экзамен проверяет опубликованные критерии своей версии. Более высокий ранг требует подтверждённых упражнений, предыдущего ранга и автоматической проверки проекта. Архитектурное и профессиональное ревью оцениваются отдельно. Это локальная учебная проверка без прокторинга и проверки личности.</p>
  </main>;
}

function parseChecks(value?: string) { try { const parsed = value ? JSON.parse(value) : []; return Array.isArray(parsed) ? parsed as ExamSubmitResult['checks'] : []; } catch { return []; } }
function prerequisiteTitle(id: string) { if(id.startsWith('rank:'))return `Подтверждённый учебный ранг ${id.slice(5)}`; if(id==='p1:automatic')return 'Автоматическая проверка P1 CLI Toolkit'; if(id==='p2:automatic')return 'Изолированная проверка P2 LinkHub'; if(id==='rank_evidence_trust_policy')return 'Текущие настройки ключей проверяющих и профилей project runner'; const task=/^(zero|junior|juniorplus)-(\d+)-(easy|medium|hard)$/.exec(id);if(task)return `Урок ${task[1]==='zero'?'Zero':task[1]==='junior'?'Junior':'Junior+'} ${task[2]}: ${({easy:'базовое задание',medium:'среднее задание',hard:'сложное задание'} as Record<string,string>)[task[3]]}`; return ({ hello:'приветствие Go', repair:'исправить программу', 'two-lines':'вывести две строки' } as Record<string, string>)[id] || id; }
function rankRequirementTitle(item: RankEvidenceRequirement) { if(item.kind==='prior_rank')return `ранг ${item.requirement_id.replace(/^rank:/,'')}`;if(item.kind==='project')return `проект ${item.requirement_id.replace(/^project:/,'')}`;if(item.kind==='manual_review')return `ручная проверка проекта ${item.requirement_id.replace(/^review:/,'')}`;return `упражнение ${item.requirement_id}`; }
function rankEvidenceReason(reason?: string, rank?: string) { if(!reason)return 'требование ещё не настроено';if(reason==='rank_evidence_policy_unconfigured')return 'для этого экзамена серверная политика доказательств ещё не настроена';if(reason==='review_trust_policy_mismatch')return 'настройки ключей проверяющих или профиль project runner изменились; оператору нужно проверить и обновить политику';if(reason==='course_evidence_missing')return 'нужно пройти и отправить указанное упражнение';if(reason==='prior_rank_missing')return 'предыдущий ранг ещё не подтверждён';if(reason.startsWith('project_evidence_missing:'))return `нет проходной оценки проекта ${reason.split(':').slice(1).join(':')}`;if(reason==='manual_review_quorum_missing')return rank==='Senior'?'не хватает решений двух разных разрешённых ключей проверяющих для точной версии проекта':'не хватает решения разрешённого проверяющего для точной версии проекта';return reason; }
function attemptStatus(status: string) { return ({ active: 'В процессе', completed: 'Завершён', cancelled: 'Отменён', interrupted: 'Прерван', grading: 'Проверяется' } as Record<string, string>)[status] || status; }
function message(error: unknown) { return error instanceof Error ? error.message : 'неизвестная ошибка'; }
