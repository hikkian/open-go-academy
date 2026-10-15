import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react';
import { ApiError, api, type LearningSnapshot } from './api';
import './Passport.css';

type Achievement = { id: string; title: string; description: string; earned_at: string };
type PassportProject = { id: string; title: string; automatic_passed: boolean; manual_review_required: boolean };
type ActivityDay = { date: string; verified_count: number; cumulative_count: number };
type Award = { attempt_id: string; rank: string; exam_id: string; version: string };
type Credential = {
  id: string; rank: string; subject_name: string; created_at: string; published: boolean;
  revoked_at: string | null; signature_valid: boolean; verify_url: string; criteria_version: string;
  attempt_id?: string;
};
type PassportData = {
  profile: { name: string };
  learning: LearningSnapshot;
  achievements: Achievement[];
  projects: PassportProject[];
  activity: ActivityDay[];
  credentials: Credential[];
  awards: Award[];
  public: boolean;
  issuer: { name: string; key_id: string; public_key: string; trust_notice: string };
};
type PrivacyDraft = { published: boolean; consent: boolean };
type VerifyStatus = 'loading' | 'notfound' | 'unavailable' | 'signatureinvalid' | 'revoked' | 'valid';
type VerifyResponse = {
  status?: string; signature_valid?: boolean; revoked?: boolean; issuer_trust?: string;
  notice?: string; key_id?: string; public_key?: string;
  payload?: Partial<{
    format: string; id: string; issuer_id: string; issuer_name: string; subject_name: string; rank: string;
    exam_id: string; criteria_version: string; criteria_hash: string; attempt_id: string; issued_at: string;
    verify_url: string; competencies: string[];
  }>;
};

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Не удалось выполнить запрос.';
}
function dateLabel(value?: string | null): string {
  if (!value) return 'Дата не указана';
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? 'Некорректная дата' : new Intl.DateTimeFormat('ru-RU', { dateStyle: 'long' }).format(date);
}
function safeFilePart(value: string): string { return value.replace(/[^a-f0-9-]/gi, '').slice(0, 80) || 'credential'; }

function usePassport() {
  const [data, setData] = useState<PassportData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const reload = useCallback(() => setRevision(value => value + 1), []);
  useEffect(() => {
    let active = true;
    setLoading(true); setError('');
    api<PassportData>('/passport').then(value => { if (active) setData(value); })
      .catch(reason => { if (active) setError(errorMessage(reason)); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [revision]);
  return { data, loading, error, reload };
}

export default function Passport() {
  const { data, loading, error, reload } = usePassport();
  const [issueAttempt, setIssueAttempt] = useState('');
  const [pseudonym, setPseudonym] = useState('');
  const [issueConsent, setIssueConsent] = useState(false);
  const [globalConsent, setGlobalConsent] = useState(false);
  const [passportPublic, setPassportPublic] = useState(false);
  const [privacy, setPrivacy] = useState<Record<string, PrivacyDraft>>({});
  const [busy, setBusy] = useState('');
  const [notice, setNotice] = useState('');
  const [actionError, setActionError] = useState('');
  const [privateVerify, setPrivateVerify] = useState<Record<string, VerifyStatus | undefined>>({});

  useEffect(() => {
    if (data) { setPassportPublic(data.public); setPrivacy({}); }
  }, [data]);

  const issuedAttempts = useMemo(() => new Set(data?.credentials.map(item => item.attempt_id).filter((id): id is string => Boolean(id)) ?? []), [data]);
  const availableAwards = data?.awards.filter(award => award.attempt_id && !issuedAttempts.has(award.attempt_id)) ?? [];
  const maxDailyVerified = Math.max(0, ...(data?.activity.map(day => day.verified_count) ?? []));

  const runAction = async (key: string, success: string, action: () => Promise<unknown>) => {
    setBusy(key); setActionError(''); setNotice('');
    try { await action(); setNotice(success); reload(); }
    catch (reason) { setActionError(errorMessage(reason)); }
    finally { setBusy(''); }
  };

  const issue = (event: FormEvent) => {
    event.preventDefault();
    if (!issueAttempt || pseudonym.trim().length === 0 || !issueConsent) return;
    void runAction('issue', 'Credential выпущен и сохранён в вашем профиле.', () => api('/credentials', {
      method: 'POST', body: JSON.stringify({ attempt_id: issueAttempt, name: pseudonym.trim(), consent: true }),
    }));
  };

  const saveCredentialPrivacy = (credential: Credential) => {
    const draft = privacy[credential.id] ?? { published: credential.published, consent: false };
    if (draft.published && !draft.consent) return;
    void runAction(`privacy:${credential.id}`, 'Настройки credential обновлены.', () => api(`/credentials/${encodeURIComponent(credential.id)}/privacy`, {
      method: 'PUT', body: JSON.stringify({ published: draft.published, consent: draft.published && draft.consent }),
    }));
  };

  const savePassportPrivacy = (event: FormEvent) => {
    event.preventDefault();
    if (passportPublic && !globalConsent) return;
    void runAction('passport-privacy', 'Настройки публичного паспорта обновлены.', () => api('/passport/privacy', {
      method: 'PUT', body: JSON.stringify({ published: passportPublic, consent: passportPublic && globalConsent }),
    }));
  };

  const revoke = (credential: Credential) => {
    if (credential.revoked_at) return;
    const approved = window.confirm(`Отозвать credential ${credential.id}? После отзыва он будет помечен недействительным.`);
    if (!approved) return;
    void runAction(`revoke:${credential.id}`, 'Credential отозван. Ссылка проверки отразит этот статус.', () => api(`/credentials/${encodeURIComponent(credential.id)}/revoke`, { method: 'POST', body: '{}' }));
  };

  const downloadSnapshot = () => {
    if (!data) return;
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a'); link.href = url; link.download = 'open-go-academy-private-passport.json'; link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 1000);
  };

  const checkPrivateCredential = async (id: string) => {
    setPrivateVerify(current => ({ ...current, [id]: 'loading' }));
    try {
      const result = await api<VerifyResponse>(`/credentials/${encodeURIComponent(id)}/verify`);
      setPrivateVerify(current => ({ ...current, [id]: verificationStatus(result, id) }));
    } catch (reason) {
      setPrivateVerify(current => ({ ...current, [id]: reason instanceof ApiError && reason.status === 404 ? 'notfound' : 'unavailable' }));
    }
  };

  if (loading && !data) return <main className="page passport-page"><p role="status">Загружаем паспорт…</p></main>;
  if (!data) return <main className="page passport-page"><div className="page-heading"><p className="eyebrow">ДОСТИЖЕНИЯ</p><h1>Паспорт разработчика</h1></div><div className="passport-message" role="alert">{error || 'Данные паспорта недоступны.'}</div><button className="outline-button" onClick={reload}>Повторить</button></main>;

  return <main className="page passport-page">
    <header className="page-heading passport-heading">
      <div><p className="eyebrow">ЛИЧНЫЙ РАЗДЕЛ · ПО УМОЛЧАНИЮ ПРИВАТНО</p><h1>Паспорт разработчика</h1><p>Учебные достижения, проверенные компетенции и локальные credentials.</p></div>
      <button className="outline-button" type="button" onClick={downloadSnapshot}>Скачать снимок JSON</button>
    </header>
    <aside className="passport-trust" role="note"><strong>Локальный учебный credential</strong><span>{data.issuer.trust_notice || 'Подпись подтверждает данные этой установки Go Academy, но не личность, стаж, должность или внешнюю аккредитацию.'}</span></aside>
    {(error || actionError) && <p className="passport-message" role="alert">{actionError || error}</p>}
    {notice && <p className="passport-notice" role="status">{notice}</p>}

    <section className="passport-overview" aria-labelledby="passport-overview-title">
      <div className="passport-profile-card"><span className="passport-avatar" aria-hidden="true">{data.profile.name.trim().slice(0, 1).toLocaleUpperCase('ru-RU') || 'G'}</span><div><p className="eyebrow">ЛОКАЛЬНЫЙ ПРОФИЛЬ</p><h2 id="passport-overview-title">{data.profile.name}</h2><p>Учебный ранг: <strong>{data.learning.current_rank}</strong> · {data.learning.verified ? 'критерии ранга подтверждены' : 'ранг ещё не подтверждён'}</p></div></div>
      <dl className="passport-metrics"><div><dt>Навыки</dt><dd>{data.learning.skills.length}</dd></div><div><dt>Достижения</dt><dd>{data.achievements.length}</dd></div><div><dt>Credentials</dt><dd>{data.credentials.length}</dd></div></dl>
    </section>

    <section className="passport-section" aria-labelledby="passport-projects-title"><div className="passport-section-head"><div><p className="eyebrow">ДОСТУПНЫЙ КАТАЛОГ</p><h2 id="passport-projects-title">Обзор проектов</h2></div></div>
      {data.projects.length ? <ul className="passport-project-list">{data.projects.map(project => <li key={project.id}><div><p className="eyebrow">{project.id.toUpperCase()}</p><h3>{project.title}</h3><p>{project.automatic_passed ? 'Автоматические проверки пройдены.' : 'Автоматическое прохождение пока не подтверждено.'}</p><small>{project.manual_review_required ? 'Ручное ревью обязательно; автопроверка его не заменяет.' : 'Ручное ревью не требуется.'}</small></div><span className={`passport-state ${project.automatic_passed ? 'valid' : ''}`}>{project.automatic_passed ? 'Автопроверки пройдены' : 'Не подтверждено'}</span></li>)}</ul> : <p className="passport-empty">В доступном каталоге пока нет проектов.</p>}
    </section>

    <section className="passport-section" aria-labelledby="passport-skills-title"><div className="passport-section-head"><div><p className="eyebrow">ДАННЫЕ УЧЕБНОГО СЕРВЕРА</p><h2 id="passport-skills-title">Навыки и ранги</h2></div></div>
      <ol className="passport-ranks">{data.learning.ranks.map((rank, index) => <li key={rank.name} className={rank.name === data.learning.current_rank ? 'current' : ''}><span>{String(index + 1).padStart(2, '0')}</span><div><strong>{rank.name}</strong><small>{rank.name === data.learning.current_rank ? 'Текущий ранг' : rank.assessment_available ? 'Проверка доступна' : 'Критерии курса'}</small></div></li>)}</ol>
      {data.learning.skills.length ? <ul className="passport-skill-list">{data.learning.skills.map(skill => <li key={skill.id}><span><strong>{skill.title}</strong><small>{skill.rank} · подтверждающих свидетельств: {skill.evidence.length}</small></span><span className="passport-skill-level" aria-label={`Уровень ${skill.level} из 4`}>{skill.level}<small>/4</small></span></li>)}</ul> : <p className="passport-empty">Сервер пока не зарегистрировал навыки.</p>}
    </section>

    <section className="passport-section" aria-labelledby="passport-achievements-title"><div className="passport-section-head"><div><p className="eyebrow">ЗАРАБОТАНО</p><h2 id="passport-achievements-title">Достижения</h2></div></div>
      {data.achievements.length ? <ul className="passport-achievements">{data.achievements.map(item => <li key={item.id}><span className="passport-achievement-mark" aria-hidden="true">✓</span><div><h3>{item.title}</h3><p>{item.description}</p><small>{dateLabel(item.earned_at)}</small></div></li>)}</ul> : <p className="passport-empty">Достижения появятся здесь после выполнения критериев курса.</p>}
    </section>

    <section className="passport-section" aria-labelledby="passport-activity-title"><div className="passport-section-head"><div><p className="eyebrow">ПОДТВЕРЖДЁННЫЕ СОБЫТИЯ</p><h2 id="passport-activity-title">Активность</h2></div></div>
      {data.activity.length ? <ol className="passport-activity-list" aria-label="Проверенные завершения по дням">{data.activity.map(day => <li key={day.date}><time dateTime={day.date}>{dateLabel(day.date)}</time><span className="passport-activity-track" aria-hidden="true"><span style={{ width: `${Math.max(5, day.verified_count / maxDailyVerified * 100)}%` }} /></span><span className="passport-activity-count">{day.verified_count} {day.verified_count === 1 ? 'подтверждение' : 'подтверждения'}</span><small>Накопительно: {day.cumulative_count}</small></li>)}</ol> : <p className="passport-empty">Пока нет подтверждённых событий. Неудачные попытки, XP и черновики здесь не учитываются.</p>}
    </section>

    <section className="passport-section" aria-labelledby="passport-credentials-title"><div className="passport-section-head"><div><p className="eyebrow">СВЯЗАНЫ С ПРОЙДЕННЫМИ ЭКЗАМЕНАМИ</p><h2 id="passport-credentials-title">Credentials</h2></div></div>
      {availableAwards.length > 0 ? <form className="passport-issue" onSubmit={issue}>
        <h3>Выпустить учебный credential</h3><p>Имя можно заменить псевдонимом. Публикация — отдельное действие и требует отдельного согласия ниже.</p>
        <label htmlFor="credential-award">Подтверждённый экзамен</label><select id="credential-award" value={issueAttempt} onChange={event => setIssueAttempt(event.target.value)} required><option value="">Выберите результат экзамена</option>{availableAwards.map(award => <option key={award.attempt_id} value={award.attempt_id}>{award.rank} · {award.exam_id} {award.version}</option>)}</select>
        <label htmlFor="credential-name">Имя в credential или псевдоним</label><input id="credential-name" value={pseudonym} onChange={event => setPseudonym(event.target.value)} maxLength={64} autoComplete="off" placeholder="Например, GoLearner" required />
        <label className="passport-consent"><input type="checkbox" checked={issueConsent} onChange={event => setIssueConsent(event.target.checked)} /><span>Я прошу выпустить локальный учебный credential с указанным именем и понимаю, что он не подтверждает личность или внешнюю квалификацию.</span></label>
        <button className="primary-button" type="submit" disabled={busy === 'issue' || !issueAttempt || !pseudonym.trim() || !issueConsent}>{busy === 'issue' ? 'Выпускаем…' : 'Выпустить credential'}</button>
      </form> : <p className="passport-empty">Пока нет доступного неиспользованного результата экзамена для выпуска credential.</p>}

      {data.credentials.length ? <div className="passport-credentials">{data.credentials.map(credential => {
        const draft = privacy[credential.id] ?? { published: credential.published, consent: false };
        const verification = privateVerify[credential.id];
        return <article className="passport-credential" key={credential.id}>
          <div className="passport-credential-top"><div><span className="passport-rank-badge">{credential.rank}</span><h3>{credential.revoked_at ? 'Отозванный credential' : credential.subject_name}</h3><p>Выдан {dateLabel(credential.created_at)} · критерии {credential.criteria_version}</p></div><span className={`passport-state ${credential.revoked_at ? 'revoked' : credential.signature_valid ? 'valid' : 'invalid'}`}>{credential.revoked_at ? 'Отозван' : credential.signature_valid ? 'Подпись проверена' : 'Подпись не подтверждена'}</span></div>
          <p className="passport-id"><span>ID</span><code>{credential.id}</code></p>
          <div className="passport-downloads" aria-label="Скачать credential">{(['json', 'pdf', 'png'] as const).map(format => <a key={format} href={`/api/credentials/${encodeURIComponent(credential.id)}/${format}`} download={`${safeFilePart(credential.id)}.${format}`} aria-label={`Скачать credential в формате ${format.toUpperCase()}`}>{format.toUpperCase()}</a>)}</div>
          <form className="passport-privacy" onSubmit={event => { event.preventDefault(); saveCredentialPrivacy(credential); }}>
        <label><input type="checkbox" checked={draft.published} disabled={Boolean(credential.revoked_at && !credential.published)} onChange={event => setPrivacy(current => ({ ...current, [credential.id]: { published: event.target.checked, consent: false } }))} /><span>Опубликовать credential</span></label>
            <label><input type="checkbox" checked={draft.consent} onChange={event => setPrivacy(current => ({ ...current, [credential.id]: { ...draft, consent: event.target.checked } }))} /><span>Я разрешаю публично показывать имя, ранг, дату, версию критериев, эмитента и ID.</span></label>
            <div className="passport-privacy-actions"><button className="outline-button" type="submit" disabled={busy === `privacy:${credential.id}` || (draft.published && !draft.consent) || Boolean(credential.revoked_at && (!credential.published || draft.published))}>{busy === `privacy:${credential.id}` ? 'Сохраняем…' : 'Сохранить видимость'}</button>
              {!credential.revoked_at && <button className="passport-revoke" type="button" onClick={() => revoke(credential)} disabled={busy === `revoke:${credential.id}`}>{busy === `revoke:${credential.id}` ? 'Отзываем…' : 'Отозвать credential'}</button>}</div>
          </form>
          <div className="passport-verify-actions">{credential.published && credential.verify_url ? <a className="text-button" href={credential.verify_url}>Открыть публичную проверку</a> : <button className="text-button" type="button" onClick={() => void checkPrivateCredential(credential.id)} disabled={verification === 'loading'}>{verification === 'loading' ? 'Проверяем…' : 'Проверить подпись в профиле'}</button>}
            </div>
          {verification && verification !== 'loading' && <p className={`passport-inline-status ${verification}`} role="status">{verifyLabel(verification)}</p>}
        </article>;
      })}</div> : null}
    </section>

    <section className="passport-section passport-publish-section" aria-labelledby="passport-public-title"><div className="passport-section-head"><div><p className="eyebrow">НЕОБЯЗАТЕЛЬНО</p><h2 id="passport-public-title">Публичный паспорт</h2></div><span className={`passport-state ${data.public ? 'valid' : ''}`}>{data.public ? 'Опубликован' : 'Приватный'}</span></div>
      <p>Публично будут показаны только имя, учебный ранг, навыки и credentials, которые вы отдельно опубликовали.</p>
      <form className="passport-privacy" onSubmit={savePassportPrivacy}>
        <label><input type="checkbox" checked={passportPublic} onChange={event => { setPassportPublic(event.target.checked); setGlobalConsent(false); }} /><span>Опубликовать мой паспорт</span></label>
        <label><input type="checkbox" checked={globalConsent} onChange={event => setGlobalConsent(event.target.checked)} /><span>Я согласен публично показывать имя, ранг и карту навыков.</span></label>
        <button className="outline-button" type="submit" disabled={busy === 'passport-privacy' || (passportPublic && !globalConsent)}>{busy === 'passport-privacy' ? 'Сохраняем…' : 'Сохранить настройки паспорта'}</button>
      </form>
      <a className="passport-public-link" href="/passport">Открыть страницу публичного паспорта <span aria-hidden="true">↗</span></a>
      <p className="passport-issuer"><strong>Эмитент:</strong> {data.issuer.name} · локальный ключ <code>{data.issuer.key_id}</code></p>
    </section>
    <p className="passport-local-note">Экспорт JSON содержит снимок личного раздела. Подписанный credential подтверждает целостность данных этой локальной установки; он не подтверждает личность, годы опыта, должность или аккредитацию.</p>
  </main>;
}

function verificationStatus(result: VerifyResponse, expectedID: string): VerifyStatus {
  if (result.signature_valid === false || result.status === 'signature_invalid' || result.status === 'invalid') return 'signatureinvalid';
  if (result.signature_valid !== true) return 'unavailable';
  const payload = result.payload;
  const completePayload = payload?.format === 'oga-local-v1' && payload.id === expectedID &&
    Boolean(payload.issuer_id && payload.issuer_name && payload.subject_name && payload.rank && payload.exam_id &&
      payload.criteria_version && /^[a-f0-9]{64}$/.test(payload.criteria_hash || '') &&
      /^[a-f0-9]{32}$/.test(payload.attempt_id || '') && payload.issued_at && payload.verify_url &&
      Array.isArray(payload.competencies) && payload.competencies.length > 0) &&
    /^[a-f0-9]{64}$/.test(result.key_id || '') && /^[A-Za-z0-9_-]{43}$/.test(result.public_key || '') &&
    result.issuer_trust === 'local-self-issued' && Boolean(result.notice);
  if (!completePayload) return 'unavailable';
  if (result.status === 'revoked' || result.revoked === true) return 'revoked';
  if (result.status === 'active') return 'valid';
  return 'unavailable';
}
function verifyLabel(status: VerifyStatus): string {
  switch (status) {
    case 'notfound': return 'Credential не найден.';
    case 'unavailable': return 'Не удалось подтвердить подпись.';
    case 'signatureinvalid': return 'Подпись недействительна: данные могли измениться.';
    case 'revoked': return 'Credential отозван эмитентом.';
    case 'valid': return 'Подпись действительна в этой установке Go Academy.';
    case 'loading': return 'Проверяем credential…';
  }
}

export function CredentialVerification({ credentialId }: { credentialId: string }) {
  const [status, setStatus] = useState<VerifyStatus>('loading');
  const [result, setResult] = useState<VerifyResponse | null>(null);
  useEffect(() => {
    let active = true;
    setStatus('loading'); setResult(null);
    api<VerifyResponse>(`/verify/${encodeURIComponent(credentialId)}`).then(value => {
      if (active) { setResult(value); setStatus(verificationStatus(value, credentialId)); }
    }).catch(error => {
      if (active) setStatus(error instanceof ApiError && error.status === 404 ? 'notfound' : 'unavailable');
    });
    return () => { active = false; };
  }, [credentialId]);
  const credential = status === 'valid' || status === 'revoked' ? result?.payload : undefined;
  const hideSubject = status === 'revoked';
  return <main className="page passport-page passport-verification">
    <div className="page-heading"><p className="eyebrow">ПУБЛИЧНАЯ ПРОВЕРКА</p><h1>Учебный credential</h1><p>Проверка подписи и статуса отзыва по данным локального эмитента.</p></div>
    <section className={`passport-verify-card ${status}`} aria-live="polite">
      <span className="passport-verify-icon" aria-hidden="true">{status === 'valid' ? '✓' : status === 'loading' ? '…' : '!'}</span>
      <h2>{status === 'loading' ? 'Проверяем подпись…' : verifyLabel(status)}</h2>
      {credential && <dl><div><dt>ID</dt><dd><code>{credential.id || credentialId}</code></dd></div>{!hideSubject && credential.subject_name && <div><dt>Имя</dt><dd>{credential.subject_name}</dd></div>}{credential.rank && <div><dt>Учебный ранг</dt><dd>{credential.rank}</dd></div>}{credential.exam_id && <div><dt>Экзамен</dt><dd>{credential.exam_id}</dd></div>}{credential.criteria_version && <div><dt>Версия критериев</dt><dd>{credential.criteria_version}</dd></div>}{credential.issued_at && <div><dt>Выдан</dt><dd>{dateLabel(credential.issued_at)}</dd></div>}{credential.criteria_hash && <div><dt>Хэш критериев</dt><dd><code>{credential.criteria_hash}</code></dd></div>}</dl>}
      <p className="passport-trust-notice">{result?.notice || 'Credential локально подписан установкой Go Academy. Проверка целостности подписи не подтверждает личность, стаж, должность или внешнюю аккредитацию.'}</p>
      <a className="outline-button" href="/">Вернуться в Go Academy</a>
    </section>
  </main>;
}

type PublicSkill = string | { id?: string; title?: string; rank?: string; level?: number };
type PublicData = {
  public?: boolean; profile?: { name?: string }; name?: string; rank?: string; current_rank?: string;
  learning?: { current_rank?: string; skills?: PublicSkill[] };
  skills?: PublicSkill[];
  credentials?: Array<Pick<Credential, 'id' | 'rank' | 'subject_name' | 'created_at' | 'verify_url' | 'criteria_version'> & { criteria_hash?: string; issuer_name?: string; revoked_at?: string | null; published?: boolean }>;
  issuer?: { name?: string };
};

export function PublicPassport() {
  const [data, setData] = useState<PublicData | null>(null);
  const [status, setStatus] = useState<'loading' | 'ready' | 'private' | 'unavailable'>('loading');
  useEffect(() => {
    let active = true;
    api<PublicData>('/passport/public').then(value => { if (active) { setData(value); setStatus(value.public === false ? 'private' : 'ready'); } })
      .catch(error => { if (active) setStatus(error instanceof ApiError && error.status === 404 ? 'private' : 'unavailable'); });
    return () => { active = false; };
  }, []);
  const skills = data?.learning?.skills ?? data?.skills ?? [];
  const publicCredentials = (data?.credentials ?? []).filter(item => item.published === true && !item.revoked_at);
  const issuerName = data?.issuer?.name || 'локальная установка Go Academy';
  const rank = data?.learning?.current_rank ?? data?.current_rank ?? data?.rank;
  const name = data?.profile?.name ?? data?.name;
  return <main className="page passport-page passport-public-page">
    <div className="page-heading"><p className="eyebrow">ПУБЛИЧНЫЙ ПРОФИЛЬ</p><h1>{status === 'ready' ? (name || 'Паспорт разработчика') : 'Паспорт разработчика'}</h1><p>Публично опубликованные учебные достижения.</p></div>
    {status === 'loading' && <p role="status">Загружаем паспорт…</p>}
    {status === 'private' && <p className="passport-empty" role="status">Этот паспорт приватный или больше не опубликован.</p>}
    {status === 'unavailable' && <p className="passport-message" role="alert">Не удалось загрузить публичный паспорт.</p>}
    {status === 'ready' && <>
      <aside className="passport-trust" role="note">Публичные данные показаны с согласия владельца. Учебный ранг и подпись credential не удостоверяют личность или внешнюю квалификацию.</aside>
      <section className="passport-public-summary"><p className="eyebrow">УЧЕБНЫЙ РАНГ</p><h2>{rank || 'Не указан'}</h2><p>{skills.length} навыков опубликовано</p></section>
      <section className="passport-section"><h2>Навыки</h2>{skills.length ? <ul className="passport-public-skills">{skills.map((skill, index) => typeof skill === 'string' ? <li key={`${skill}-${index}`}>{skill}</li> : <li key={skill.id || `${skill.title}-${index}`}><span>{skill.title || skill.id || 'Навык'}</span>{typeof skill.level === 'number' && <small>{skill.level} / 4</small>}</li>)}</ul> : <p className="passport-empty">Публичные навыки не указаны.</p>}</section>
      <section className="passport-section"><h2>Опубликованные credentials</h2>{publicCredentials.length ? <ul className="passport-public-credentials">{publicCredentials.map(item => <li key={item.id}><div><strong>{item.revoked_at ? `${item.rank} · отозван` : item.rank}</strong>{!item.revoked_at && <span>{item.subject_name}</span>}<small>{dateLabel(item.created_at)} · критерии {item.criteria_version}</small><small>Эмитент: {item.issuer_name || issuerName}</small></div><a href={item.verify_url || `/verify/${encodeURIComponent(item.id)}`}>Проверить</a></li>)}</ul> : <p className="passport-empty">Публичных credentials пока нет.</p>}</section>
    </>}
    <p className="passport-trust-notice">Credentials выпускаются этой установкой как подтверждение учебных критериев. Они не подтверждают личность, стаж, должность или аккредитацию.</p>
  </main>;
}
