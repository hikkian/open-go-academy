import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import ReactMarkdown from 'react-markdown';
import { api } from './api';
import './Tutor.css';

type Mode = 'explain' | 'simpler' | 'deeper' | 'example' | 'hint' | 'debug' | 'review' | 'quiz' | 'interview' | 'architecture' | 'study-plan';
type Usage = { day: string; used_tokens: number; reserved_tokens: number; requests: number; failed_requests: number; estimated_requests: number };
type TutorConfig = { enabled: boolean; cloud: boolean; provider_label: string; models: { economy: string; advanced: string }; daily_budget: number; max_output_tokens: number; usage: Usage; modes: string[]; privacy_notice: string };
type TutorReply = { text: string; model: string; input_tokens: number; output_tokens: number; estimated: boolean; cached: boolean; usage: Usage };

const modeInfo: Record<Mode, { label: string; description: string }> = {
  explain: { label: 'Объяснить тему', description: 'Объяснить тему целиком, начиная с базовых понятий и примеров.' },
  simpler: { label: 'Объяснить проще', description: 'Разобрать идею короткими шагами и простыми словами.' },
  deeper: { label: 'Объяснить подробнее', description: 'Добавить детали и связать тему с устройством Go.' },
  example: { label: 'Показать пример', description: 'Показать небольшой пример и объяснить каждую часть.' },
  hint: { label: 'Дать подсказку', description: 'Подвести к решению, не раскрывая готовый ответ.' },
  debug: { label: 'Помочь найти ошибку', description: 'Объяснить возможную причину ошибки и способ проверки.' },
  review: { label: 'Разобрать код', description: 'Обсудить читаемость, корректность и простые улучшения.' },
  quiz: { label: 'Задать вопрос', description: 'Проверить понимание одним вопросом и разобрать ответ.' },
  interview: { label: 'Потренировать собеседование', description: 'Задать учебный вопрос и обсудить ход рассуждений.' },
  architecture: { label: 'Обсудить устройство программы', description: 'Помочь продумать части программы и их связи.' },
  'study-plan': { label: 'Составить план занятий', description: 'Предложить следующий учебный шаг по твоему запросу.' },
};
const modeOrder: Mode[] = ['explain', 'simpler', 'deeper', 'example', 'hint', 'debug', 'review', 'quiz', 'interview', 'architecture', 'study-plan'];
const messageLimit = 2000;
const sourceLimit = 8 * 1024;

export default function Tutor({ lessonId, source }: { lessonId?: string; source?: string }) {
  const [config, setConfig] = useState<TutorConfig | null>(null);
  const [mode, setMode] = useState<Mode>('simpler');
  const [message, setMessage] = useState('');
  const [consent, setConsent] = useState(false);
  const [shareCode, setShareCode] = useState(false);
  const [reply, setReply] = useState<TutorReply | null>(null);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const [configError, setConfigError] = useState('');

  const loadConfig = async () => {
    setLoading(true); setConfigError('');
    try {
      const loaded = await api<TutorConfig>('/tutor');
      setConfig(loaded);
      const firstSupported = modeOrder.find(item => loaded.modes.includes(item));
      if (firstSupported) setMode(current => loaded.modes.includes(current) ? current : firstSupported);
    }
    catch (e) { setConfigError(`Не удалось загрузить настройки наставника: ${errorText(e)}`); }
    finally { setLoading(false); }
  };
  useEffect(() => { void loadConfig(); }, []);

  const allowedModes = useMemo(() => modeOrder.filter(item => config?.modes.includes(item)), [config?.modes]);
  const sourceBytes = useMemo(() => source === undefined ? 0 : new TextEncoder().encode(source).length, [source]);
  const canShareSource = source !== undefined && sourceBytes <= sourceLimit;
  const remainingTokens = config ? Math.max(0, config.daily_budget - config.usage.used_tokens - config.usage.reserved_tokens) : 0;

  const send = async (event: FormEvent) => {
    event.preventDefault();
    if (!config?.enabled || sending || !consent || !message.trim() || message.length > messageLimit || (shareCode && !canShareSource)) return;
    setSending(true); setError(''); setReply(null);
    const payload = { mode, message: message.trim(), ...(lessonId ? { lesson_id: lessonId } : {}), ...(shareCode && source !== undefined ? { source } : {}), consent: true as const, share_code: shareCode };
    try {
      const result = await api<TutorReply>('/tutor', { method: 'POST', body: JSON.stringify(payload) });
      setReply(result);
      setConfig(current => current ? { ...current, usage: result.usage } : current);
      setConsent(false);
    } catch (e) {
      const failure = errorText(e);
      let refreshFailure = '';
      try {
        const refreshed = await api<TutorConfig>('/tutor');
        setConfig(current => current ? { ...current, usage: refreshed.usage } : current);
      } catch (refreshError) { refreshFailure = ` Статус расхода тоже не удалось обновить: ${errorText(refreshError)}.`; }
      setError(`Не удалось получить ответ: ${failure}.${refreshFailure} Данные запроса сохранены; повторная отправка будет только по твоему действию.`);
    }
    finally { setSending(false); }
  };

  if (loading) return <main className="tutor-page"><p className="tutor-muted" role="status">Проверяем настройки ИИ-наставника…</p></main>;
  if (configError || !config) return <main className="tutor-page"><header className="tutor-heading"><p className="tutor-kicker">ДОПОЛНИТЕЛЬНАЯ ПОМОЩЬ</p><h1>ИИ-наставник</h1></header><p className="tutor-error" role="alert">{configError || 'Настройки наставника недоступны.'}</p><button className="tutor-button secondary" onClick={() => void loadConfig()}>Повторить загрузку</button></main>;

  return <main className="tutor-page">
    <header className="tutor-heading"><p className="tutor-kicker">ДОПОЛНИТЕЛЬНАЯ ПОМОЩЬ</p><h1>ИИ-наставник</h1><p>Задай вопрос по Go или попроси помочь с учебной задачей. Наставник может ошибаться; проверяй объяснения на практике.</p></header>
    <section className="tutor-status" aria-label="Состояние наставника">
      <div><span className={config.enabled ? 'tutor-indicator on' : 'tutor-indicator'} />{config.enabled ? 'Наставник включён' : 'Наставник отключён'}</div>
      <p>Назначение запроса: <strong>{config.cloud ? 'облачный внешний ИИ-провайдер' : 'локальное окружение учебного сервиса'}</strong>{config.provider_label ? ` (${config.provider_label})` : ''}.</p>
      <p>Доступные модели: <code>{config.models.economy}</code> и <code>{config.models.advanced}</code>. Конкретную модель для ответа покажет сервер.</p>
      <p>{config.privacy_notice}</p>
    </section>
    <section className="tutor-budget" aria-label="Лимит запросов">
      <div><strong>Дневной лимит (UTC, {config.usage.day})</strong><span>{number(config.usage.used_tokens)} использовано из {number(config.daily_budget)} токенов</span></div>
      <progress max={Math.max(1, config.daily_budget)} value={Math.min(config.daily_budget, config.usage.used_tokens + config.usage.reserved_tokens)} aria-label="Использование дневного бюджета" />
      <p>{number(config.usage.reserved_tokens)} токенов зарезервировано · {number(config.usage.estimated_requests)} запросов с оценочным расходом · {number(config.usage.requests)} отправлено · {number(config.usage.failed_requests)} не завершилось. На запрос резервируется запас до четырёх вызовов провайдера. Ошибка или тайм-аут списывает резерв целиком; ручная повторная отправка создаёт новый резерв. Это счётчик токенов, а не цена в деньгах. Свободно примерно {number(remainingTokens)} токенов; лимит проверяется до обращения к провайдеру.</p>
    </section>
    {!config.enabled ? <section className="tutor-disabled"><h2>Наставник сейчас недоступен</h2><p>ИИ-запросы отключены в настройках этого учебного сервиса. Все материалы курса, редактор и проверки заданий продолжают работать без ИИ.</p><p>Для локальной настройки администратор сервиса может изучить комментарии в <code>.env.example</code> и задать параметры на сервере. Ключи провайдера хранятся только на сервере; не вставляй их в браузер и не добавляй в репозиторий.</p></section> : <form className="tutor-form" onSubmit={send}>
      <label className="tutor-label" htmlFor="tutor-mode">Чем помочь</label>
      <select id="tutor-mode" value={mode} onChange={event => { setMode(event.target.value as Mode); setConsent(false); }} disabled={sending || allowedModes.length === 0}>{allowedModes.map(item => <option key={item} value={item}>{modeInfo[item].label}</option>)}</select>
      <p className="tutor-mode-help">{modeInfo[mode].description}</p>
      {allowedModes.length === 0 && <p className="tutor-unavailable" role="status">Сервер не включил режимы наставника. Проверь настройки позже.</p>}
      <label className="tutor-label" htmlFor="tutor-message">Твой вопрос</label>
      <textarea id="tutor-message" rows={5} maxLength={messageLimit} value={message} onChange={event => { setMessage(event.target.value); setConsent(false); }} placeholder="Например: объясни, почему функция возвращает значение типа int" disabled={sending || allowedModes.length === 0} required />
      <div className="tutor-count" aria-live="polite">{message.length} / {messageLimit} символов</div>
      {lessonId && <p className="tutor-context-note">В запросе будет указан урок <code>{lessonId}</code>. Сервер может добавить его опубликованный текст как контекст.</p>}
      {source !== undefined && <label className="tutor-consent tutor-code-choice"><input type="checkbox" checked={shareCode} onChange={event => { setShareCode(event.target.checked); setConsent(false); }} disabled={sending || !canShareSource} /><span><strong>Передать код редактора</strong><small>{canShareSource ? `${number(sourceBytes)} байт будут отправлены провайдеру вместе с вопросом.` : `Код занимает ${number(sourceBytes)} байт и превышает лимит 8 КиБ; отправка кода недоступна.`}</small></span></label>}
      <details className="tutor-preview"><summary>Посмотреть, что будет отправлено</summary><dl><dt>Режим</dt><dd>{modeInfo[mode]?.label || mode}</dd><dt>Место обработки</dt><dd>{config.cloud ? 'Облачный внешний ИИ-провайдер' : 'Локальное окружение учебного сервиса'}{config.provider_label ? ` · ${config.provider_label}` : ''}</dd><dt>Сообщение</dt><dd className="tutor-preview-text">{message.trim() || 'Пока не заполнено'}</dd>{lessonId && <><dt>Идентификатор урока</dt><dd><code>{lessonId}</code>; сервер может добавить опубликованный текст урока.</dd></>}{source !== undefined && <><dt>Код</dt><dd>{shareCode ? <pre>{source}</pre> : 'Не будет отправлен.'}</dd></>}</dl></details>
      <label className="tutor-consent"><input type="checkbox" checked={consent} onChange={event => setConsent(event.target.checked)} disabled={sending} /><span><strong>Я прочитал сведения о передаваемых данных и согласен отправить этот запрос.</strong><small>Запрос может расходовать дневной лимит и учитываться у выбранного провайдера согласно его правилам.</small></span></label>
      {error && <p className="tutor-error" role="alert">{error}</p>}
      <div className="tutor-submit-row"><span>Максимальный ответ: {number(config.max_output_tokens)} токенов. ИИ не запускает код и не подтверждает учебный ранг.</span><button className="tutor-button" disabled={sending || !consent || !message.trim() || message.length > messageLimit || allowedModes.length === 0 || (shareCode && !canShareSource)}>{sending ? 'Ждём ответ…' : 'Отправить запрос'}</button></div>
    </form>}
    {reply && <section className="tutor-reply" aria-live="polite"><div className="tutor-reply-heading"><h2>Ответ наставника</h2><span>{reply.model}</span></div><div className="tutor-markdown"><ReactMarkdown components={{ img: ({ alt }) => <span className="tutor-image-suppressed">{alt || 'Внешнее изображение скрыто.'}</span>, a: ({ href, children }) => <a href={href} target="_blank" rel="noopener noreferrer">{children}</a> }}>{reply.text}</ReactMarkdown></div><p className="tutor-reply-meta">{number(reply.input_tokens)} входных · {number(reply.output_tokens)} выходных токенов{reply.estimated ? ' · оценка расхода' : ''}{reply.cached ? ' · ответ из кэша' : ''}. Остаток бюджета уточнён сервером.</p><p className="tutor-rank-note">Используй редактор и проверки курса, чтобы проверить код. Ответ ИИ сам по себе не запускает программу, не меняет прогресс и не подтверждает ранг.</p></section>}
  </main>;
}

function number(value: number) { return new Intl.NumberFormat('ru-RU').format(value); }
function errorText(error: unknown) { return error instanceof Error ? error.message : 'неизвестная ошибка'; }
