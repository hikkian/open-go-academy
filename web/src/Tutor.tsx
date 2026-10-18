import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import ReactMarkdown from 'react-markdown';
import { api } from './api';
import './Tutor.css';

type Mode = 'explain' | 'simpler' | 'deeper' | 'example' | 'hint' | 'debug' | 'review' | 'quiz' | 'interview' | 'architecture' | 'study-plan';
type TutorHistoryMessage = { role: 'assistant' | 'user'; content: string };
type Usage = { day: string; used_tokens: number; reserved_tokens: number; requests: number; failed_requests: number; estimated_requests: number };
type LocalProvider = { endpoint: string; model: string; advanced_model?: string };
type CloudProvider = { endpoint: string; model: string; advanced_model?: string };
type TutorConfig = { enabled: boolean; provider_revision: string; cloud?: boolean; provider_location?: 'local' | 'cloud' | 'unknown'; provider_label: string; local_provider_configurable?: boolean; local_provider?: LocalProvider | null; cloud_provider_configurable?: boolean; cloud_provider_configured?: boolean; cloud_provider?: CloudProvider | null; cloud_privacy_notice?: string; models: { economy: string; advanced: string }; daily_budget: number; max_output_tokens: number; usage: Usage; modes: string[]; privacy_notice: string };
type TutorReply = { text: string; model: string; input_tokens: number; output_tokens: number; estimated: boolean; cached: boolean; usage: Usage; context_lessons?: { id: string; title: string }[] };
type TutorDiagnostics = { status?: string; exit_code?: number; stdout?: string; stderr?: string; checks?: { total: number; passed: number; failed: number } };

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

const diagnosticsLimit = 4 * 1024;
const historyLimit = 12 * 1024;
const historyMessageLimit = 8;

export default function Tutor({ lessonId, source, lastResult }: { lessonId?: string; source?: string; lastResult?: TutorDiagnostics | null }) {
  const [config, setConfig] = useState<TutorConfig | null>(null);
  const [mode, setMode] = useState<Mode>('simpler');
  const [message, setMessage] = useState('');
  const [consent, setConsent] = useState(false);
  const [shareCode, setShareCode] = useState(false);
  const [shareDiagnostics, setShareDiagnostics] = useState(false);
  const [shareProgress, setShareProgress] = useState(false);
  const [reply, setReply] = useState<TutorReply | null>(null);
  const [history, setHistory] = useState<TutorHistoryMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const [configError, setConfigError] = useState('');
  const [localEndpoint, setLocalEndpoint] = useState('');
  const [localModel, setLocalModel] = useState('');
  const [localAdvancedModel, setLocalAdvancedModel] = useState('');
  const [localBusy, setLocalBusy] = useState(false);
  const [localStatus, setLocalStatus] = useState('');
  const [localError, setLocalError] = useState('');
  const [cloudEndpoint, setCloudEndpoint] = useState('https://api.openai.com/v1');
  const [cloudModel, setCloudModel] = useState('');
  const [cloudAdvancedModel, setCloudAdvancedModel] = useState('');
  const [cloudApiKey, setCloudApiKey] = useState('');
  const [cloudBusy, setCloudBusy] = useState(false);
  const [cloudStatus, setCloudStatus] = useState('');
  const [cloudError, setCloudError] = useState('');

  const loadConfig = async () => {
    setLoading(true); setConfigError('');
    try {
      const loaded = await api<TutorConfig>('/tutor');
      if (config && providerFingerprint(config) !== providerFingerprint(loaded)) setConsent(false);
      setConfig(loaded);
      setLocalEndpoint(loaded.local_provider?.endpoint ?? '');
      setLocalModel(loaded.local_provider?.model ?? '');
      setLocalAdvancedModel(loaded.local_provider?.advanced_model ?? '');
      setCloudEndpoint(loaded.cloud_provider?.endpoint ?? 'https://api.openai.com/v1');
      setCloudModel(loaded.cloud_provider?.model ?? '');
      setCloudAdvancedModel(loaded.cloud_provider?.advanced_model ?? '');
      setCloudApiKey('');
      const firstSupported = modeOrder.find(item => loaded.modes.includes(item));
      if (firstSupported) setMode(current => loaded.modes.includes(current) ? current : firstSupported);
    }
    catch (e) { setConfigError(`Не удалось загрузить настройки наставника: ${errorText(e)}`); }
    finally { setLoading(false); }
  };
  useEffect(() => { void loadConfig(); }, []);
  useEffect(() => { setHistory([]); setReply(null); }, [lessonId]);

  const allowedModes = useMemo(() => modeOrder.filter(item => config?.modes.includes(item)), [config?.modes]);
  const sourceBytes = useMemo(() => source === undefined ? 0 : new TextEncoder().encode(source).length, [source]);
  const canShareSource = source !== undefined && sourceBytes <= sourceLimit;
  const diagnosticsJSON = useMemo(() => lastResult ? JSON.stringify(lastResult, null, 2) : '', [lastResult]);
  const diagnosticsBytes = useMemo(() => diagnosticsJSON ? new TextEncoder().encode(diagnosticsJSON).length : 0, [diagnosticsJSON]);
  const canShareDiagnostics = Boolean(lastResult) && diagnosticsBytes <= diagnosticsLimit;
  const providerLocation = config?.provider_location === 'local' ? 'локальный настроенный языковой провайдер' : config?.provider_location === 'cloud' ? 'облачный настроенный языковой провайдер' : 'место обработки не указано сервером';
  const remainingTokens = config ? Math.max(0, config.daily_budget - config.usage.used_tokens - config.usage.reserved_tokens) : 0;
  const localEndpointError = localEndpoint.trim() ? loopbackEndpointError(localEndpoint.trim()) : 'Укажи адрес запущенного локального провайдера.';
  const localModelError = localModel.trim().length === 0 ? 'Укажи имя модели.' : localModel.trim().length > 128 ? 'Имя модели должно быть не длиннее 128 символов.' : '';
  const cloudEndpointError = cloudEndpoint.trim() ? cloudEndpointErrorFor(cloudEndpoint.trim()) : 'Укажи HTTPS endpoint провайдера.';
  const cloudModelError = cloudModel.trim().length === 0 ? 'Укажи имя модели.' : cloudModel.trim().length > 128 ? 'Имя модели должно быть не длиннее 128 символов.' : '';
  const cloudKeyError = cloudApiKey.length === 0 ? 'Введи API-ключ; сохранённый ключ не показывается в браузере.' : cloudApiKey.length > 4096 || /[\r\n\0]/.test(cloudApiKey) ? 'Ключ превышает допустимый формат или длину.' : '';

  useEffect(() => {
    if (!canShareDiagnostics && shareDiagnostics) { setShareDiagnostics(false); setConsent(false); }
  }, [canShareDiagnostics, shareDiagnostics]);

  const send = async (event: FormEvent) => {
    event.preventDefault();
    if (!config?.enabled || sending || localBusy || cloudBusy || !consent || !message.trim() || message.length > messageLimit || (shareCode && !canShareSource) || (shareDiagnostics && !canShareDiagnostics)) return;
    setSending(true); setError(''); setReply(null);
    const conversational = mode === 'quiz' || mode === 'interview';
    const requestHistory = conversational ? boundedHistory(history) : [];
    const payload = { mode, message: message.trim(), provider_revision: config.provider_revision, ...(lessonId ? { lesson_id: lessonId } : {}), ...(shareCode && source !== undefined ? { source } : {}), consent: true as const, share_code: shareCode, ...(shareDiagnostics && lastResult ? { diagnostics: lastResult } : {}), share_diagnostics: shareDiagnostics, share_progress: shareProgress, ...(requestHistory.length ? { history: requestHistory } : {}) };
    try {
      const result = await api<TutorReply>('/tutor', { method: 'POST', body: JSON.stringify(payload) });
      setReply(result);
      if (conversational) setHistory(current => boundedHistory([...current, { role: 'user', content: message.trim() }, { role: 'assistant', content: result.text }]));
      setConfig(current => current ? { ...current, usage: result.usage } : current);
      setConsent(false);
    } catch (e) {
      const failure = errorText(e);
      let refreshFailure = '';
      try {
        const refreshed = await api<TutorConfig>('/tutor');
        if (isApiStatus(e, 409) || providerFingerprint(config) !== providerFingerprint(refreshed)) {
          setConfig(refreshed);
          setLocalEndpoint(refreshed.local_provider?.endpoint ?? ''); setLocalModel(refreshed.local_provider?.model ?? ''); setLocalAdvancedModel(refreshed.local_provider?.advanced_model ?? '');
          setCloudEndpoint(refreshed.cloud_provider?.endpoint ?? 'https://api.openai.com/v1'); setCloudModel(refreshed.cloud_provider?.model ?? ''); setCloudAdvancedModel(refreshed.cloud_provider?.advanced_model ?? ''); setCloudApiKey('');
          setConsent(false);
          setError(`${failure} Настройка обновлена; проверь новое место обработки и согласись заново.`);
          return;
        }
        setConfig(current => current ? { ...current, usage: refreshed.usage } : current);
      } catch (refreshError) { refreshFailure = ` Статус расхода тоже не удалось обновить: ${errorText(refreshError)}.`; }
      setError(`Не удалось получить ответ: ${failure}.${refreshFailure} Данные запроса сохранены; повторная отправка будет только по твоему действию.`);
    }
    finally { setSending(false); }
  };

  const saveLocalProvider = async (event: FormEvent) => {
    event.preventDefault();
    if (!config?.local_provider_configurable || localBusy || localEndpointError || localModelError) return;
    setConsent(false); setLocalBusy(true); setLocalError(''); setLocalStatus('');
    try {
      await api<unknown>('/tutor/local-provider', { method: 'PUT', body: JSON.stringify({ endpoint: localEndpoint.trim(), model: localModel.trim(), ...(localAdvancedModel.trim() ? { advanced_model: localAdvancedModel.trim() } : {}) }) });
    } catch (e) { setLocalError(`Не удалось сохранить настройки: ${errorText(e)}`); setLocalBusy(false); return; }
    try {
      const refreshed = await api<TutorConfig>('/tutor');
      if (config && providerFingerprint(config) !== providerFingerprint(refreshed)) setConsent(false);
      setConfig(refreshed);
      setLocalEndpoint(refreshed.local_provider?.endpoint ?? ''); setLocalModel(refreshed.local_provider?.model ?? ''); setLocalAdvancedModel(refreshed.local_provider?.advanced_model ?? '');
      setLocalStatus('Настройки сохранены. Провайдер не запускался и модель не загружалась.');
    } catch (e) { setLocalError(`Настройки сохранены, но обновить их состояние не удалось: ${errorText(e)}`); }
    finally { setLocalBusy(false); }
  };

  const removeLocalProvider = async () => {
    if (!config?.local_provider_configurable || !config.local_provider || localBusy) return;
    setConsent(false); setLocalBusy(true); setLocalError(''); setLocalStatus('');
    try {
      await api<unknown>('/tutor/local-provider', { method: 'DELETE' });
    } catch (e) { setLocalError(`Не удалось удалить локальную настройку: ${errorText(e)}`); setLocalBusy(false); return; }
    try {
      const refreshed = await api<TutorConfig>('/tutor');
      if (config && providerFingerprint(config) !== providerFingerprint(refreshed)) setConsent(false);
      setConfig(refreshed);
      setLocalEndpoint(refreshed.local_provider?.endpoint ?? ''); setLocalModel(refreshed.local_provider?.model ?? ''); setLocalAdvancedModel(refreshed.local_provider?.advanced_model ?? '');
      setLocalStatus('Локальная настройка удалена. Наставник снова использует серверную конфигурацию.');
    } catch (e) { setLocalError(`Настройка удалена, но обновить состояние провайдера не удалось: ${errorText(e)}`); }
    finally { setLocalBusy(false); }
  };

  const saveCloudProvider = async (event: FormEvent) => {
    event.preventDefault();
    if (!config?.cloud_provider_configurable || cloudBusy || cloudEndpointError || cloudModelError || cloudKeyError) return;
    setConsent(false); setCloudBusy(true); setCloudError(''); setCloudStatus('');
    try {
      await api<unknown>('/tutor/cloud-provider', { method: 'PUT', body: JSON.stringify({ endpoint: cloudEndpoint.trim(), model: cloudModel.trim(), ...(cloudAdvancedModel.trim() ? { advanced_model: cloudAdvancedModel.trim() } : {}), api_key: cloudApiKey }) });
    } catch (e) { setCloudError(`Не удалось сохранить облачную настройку: ${errorText(e)}`); setCloudBusy(false); return; }
    setCloudApiKey('');
    try {
      const refreshed = await api<TutorConfig>('/tutor');
      if (config && providerFingerprint(config) !== providerFingerprint(refreshed)) setConsent(false);
      setConfig(refreshed);
      setCloudEndpoint(refreshed.cloud_provider?.endpoint ?? cloudEndpoint.trim());
      setCloudModel(refreshed.cloud_provider?.model ?? cloudModel.trim());
      setCloudAdvancedModel(refreshed.cloud_provider?.advanced_model ?? cloudAdvancedModel.trim());
      setLocalEndpoint(refreshed.local_provider?.endpoint ?? ''); setLocalModel(refreshed.local_provider?.model ?? ''); setLocalAdvancedModel(refreshed.local_provider?.advanced_model ?? '');
      setCloudStatus('Настройки сохранены. Ключ скрыт; подключение и модель не проверялись вызовом. Запрос отправится только после отдельного согласия.');
    } catch (e) { setCloudError(`Настройки сохранены, но обновить их состояние не удалось: ${errorText(e)}`); }
    finally { setCloudBusy(false); }
  };

  const removeCloudProvider = async () => {
    if (!config?.cloud_provider_configurable || !config.cloud_provider_configured || cloudBusy) return;
    setConsent(false); setCloudBusy(true); setCloudError(''); setCloudStatus('');
    try { await api<unknown>('/tutor/cloud-provider', { method: 'DELETE' }); }
    catch (e) { setCloudError(`Не удалось удалить облачную настройку: ${errorText(e)}`); setCloudBusy(false); return; }
    setCloudApiKey('');
    try {
      const refreshed = await api<TutorConfig>('/tutor');
      if (config && providerFingerprint(config) !== providerFingerprint(refreshed)) setConsent(false);
      setConfig(refreshed);
      setCloudEndpoint(refreshed.cloud_provider?.endpoint ?? 'https://api.openai.com/v1'); setCloudModel(refreshed.cloud_provider?.model ?? ''); setCloudAdvancedModel(refreshed.cloud_provider?.advanced_model ?? '');
      setLocalEndpoint(refreshed.local_provider?.endpoint ?? ''); setLocalModel(refreshed.local_provider?.model ?? ''); setLocalAdvancedModel(refreshed.local_provider?.advanced_model ?? '');
      setCloudStatus('Облачная настройка удалена; сервер вернулся к провайдеру из своей конфигурации.');
    } catch (e) { setCloudError(`Настройка удалена, но обновить её состояние не удалось: ${errorText(e)}`); }
    finally { setCloudBusy(false); }
  };

  if (loading) return <main className="tutor-page"><p className="tutor-muted" role="status">Проверяем настройки ИИ-наставника…</p></main>;
  if (configError || !config) return <main className="tutor-page"><header className="tutor-heading"><p className="tutor-kicker">ДОПОЛНИТЕЛЬНАЯ ПОМОЩЬ</p><h1>ИИ-наставник</h1></header><p className="tutor-error" role="alert">{configError || 'Настройки наставника недоступны.'}</p><button className="tutor-button secondary" onClick={() => void loadConfig()}>Повторить загрузку</button></main>;

  return <main className="tutor-page">
    <header className="tutor-heading"><p className="tutor-kicker">ДОПОЛНИТЕЛЬНАЯ ПОМОЩЬ</p><h1>ИИ-наставник</h1><p>Задай вопрос по Go или попроси помочь с учебной задачей. Наставник может ошибаться; проверяй объяснения на практике.</p></header>
    <section className="tutor-status" aria-label="Состояние наставника">
      <div><span className={config.enabled ? 'tutor-indicator on' : 'tutor-indicator'} />{config.enabled ? 'Наставник включён' : 'Наставник отключён'}</div>
      <p>Обработка: <strong>{providerLocation}</strong>{config.provider_label ? ` (${config.provider_label})` : ''}. Используется внешний или локально настроенный языковой провайдер. Академия не обучает модель на твоих сообщениях; провайдер может обрабатывать запрос по собственной политике данных.</p>
      <p>Наставник помогает разбирать материал, но не оценивает компетенцию и не меняет учебный ранг. Основной курс, редактор, проекты и проверки работают без ИИ.</p>
      {config.models.economy || config.models.advanced ? <p>Доступные модели: {[config.models.economy, config.models.advanced].filter(Boolean).map(model => <code key={model}>{model}</code>)}. Конкретную модель для ответа покажет сервер.</p> : <p>Модель пока не настроена. Подключи локальный провайдер или облачную модель в настройках ниже.</p>}
      <p>{config.privacy_notice}</p>
    </section>
    {config.local_provider_configurable && <section className="tutor-budget" aria-labelledby="local-provider-title">
      <h2 id="local-provider-title">Свой локальный ИИ-провайдер</h2>
      <p>Укажи адрес провайдера, который уже запущен на машине, где работает API Open Go Academy. Поддерживается OpenAI-compatible Chat Completions API. Сервер блокирует все адреса вне loopback; проверка в форме — только удобство. Принимаются HTTP-адреса localhost, 127.0.0.1 или ::1.</p>
      <p>Здесь нет поля для API-ключа. Сохранение только записывает настройки: оно не проверяет подключение вызовом модели, не запускает сервис и не скачивает веса.</p>
      {config.local_provider && <p role="status">Сейчас настроено: <code>{config.local_provider.model}</code> на <code>{config.local_provider.endpoint}</code>{config.local_provider.advanced_model ? ` · расширенная модель: ${config.local_provider.advanced_model}` : ''}</p>}
      <form className="tutor-form" onSubmit={saveLocalProvider}>
        <label className="tutor-label" htmlFor="local-provider-endpoint">Loopback endpoint</label>
        <input className="tutor-local-input" style={localInputStyle} id="local-provider-endpoint" type="url" value={localEndpoint} onChange={event => setLocalEndpoint(event.target.value)} placeholder="http://127.0.0.1:1234/v1" maxLength={512} disabled={localBusy} aria-describedby="local-endpoint-help" required />
        <small id="local-endpoint-help" className={localEndpointError ? 'tutor-local-help error' : 'tutor-local-help'}>{localEndpointError || 'Подходит только HTTP-адрес localhost, 127.0.0.1 или ::1.'}</small>
        <label className="tutor-label" htmlFor="local-provider-model">Имя основной модели</label>
        <input className="tutor-local-input" style={localInputStyle} id="local-provider-model" value={localModel} onChange={event => setLocalModel(event.target.value)} maxLength={128} placeholder="например, qwen2.5-coder" disabled={localBusy} required />
        {localModelError && <small className="tutor-local-help error">{localModelError}</small>}
        <label className="tutor-label" htmlFor="local-provider-advanced">Имя расширенной модели (необязательно)</label>
        <input className="tutor-local-input" style={localInputStyle} id="local-provider-advanced" value={localAdvancedModel} onChange={event => setLocalAdvancedModel(event.target.value)} maxLength={128} placeholder="например, qwen2.5-coder:14b" disabled={localBusy} />
        {localError && <p className="tutor-error" role="alert">{localError}</p>}
        {localStatus && <p className="tutor-local-status" role="status">{localStatus}</p>}
        <div className="tutor-submit-row"><span>Модель должна быть заранее доступна локальному провайдеру.</span><span className="tutor-local-actions"><button className="tutor-button" disabled={localBusy || Boolean(localEndpointError) || Boolean(localModelError)}>{localBusy ? 'Сохраняем…' : 'Сохранить настройки'}</button>{config.local_provider && <button type="button" className="tutor-button secondary" disabled={localBusy} onClick={() => void removeLocalProvider()}>{localBusy ? 'Подождите…' : 'Удалить настройку'}</button>}</span></div>
      </form>
    </section>}
    {config.cloud_provider_configurable && <section className="tutor-budget" aria-labelledby="cloud-provider-title">
      <h2 id="cloud-provider-title">Облачный ИИ-провайдер</h2>
      <p>Если локальная модель не подойдёт, здесь можно подключить OpenAI или другой OpenAI-compatible HTTPS endpoint. Настройка действует только для этого локального сервиса и переключает активного наставника с локальной модели. API-ключ нужен отдельно и вводится только в эту форму.</p>
      <p>{config.cloud_privacy_notice || 'Облачный провайдер получает только выбранный контекст и обрабатывает его по собственной политике данных. API тарифицируется отдельно; подписка Codex/ChatGPT не оплачивает использование API.'} Для OpenAI API-ключ создаётся в <a href="https://platform.openai.com/api-keys" target="_blank" rel="noopener noreferrer">настройках API-платформы</a>; ChatGPT-подписка не заменяет этот ключ.</p>
      {config.cloud_provider_configured && config.cloud_provider && <p role="status">Сейчас подключено: <code>{config.cloud_provider.model}</code> на <code>{config.cloud_provider.endpoint}</code>{config.cloud_provider.advanced_model ? ` · расширенная модель: ${config.cloud_provider.advanced_model}` : ''}. Сохранённый ключ скрыт.</p>}
      <form className="tutor-form" onSubmit={saveCloudProvider}>
        <label className="tutor-label" htmlFor="cloud-provider-endpoint">HTTPS endpoint</label>
        <input className="tutor-local-input" style={localInputStyle} id="cloud-provider-endpoint" type="url" value={cloudEndpoint} onChange={event => setCloudEndpoint(event.target.value)} placeholder="https://api.openai.com/v1" maxLength={2048} disabled={cloudBusy} aria-describedby="cloud-endpoint-help" required />
        <small id="cloud-endpoint-help" className={cloudEndpointError ? 'tutor-local-help error' : 'tutor-local-help'}>{cloudEndpointError || 'Только HTTPS и публичный хост; localhost и приватная сеть запрещены сервером.'}</small>
        <label className="tutor-label" htmlFor="cloud-provider-model">Имя основной модели</label>
        <input className="tutor-local-input" style={localInputStyle} id="cloud-provider-model" list="openai-model-suggestions" value={cloudModel} onChange={event => setCloudModel(event.target.value)} maxLength={128} placeholder="например, gpt-6.1-sol" disabled={cloudBusy} required />
        <datalist id="openai-model-suggestions"><option value="gpt-6.1-sol" /><option value="gpt-6-luna" /><option value="gpt-6-astra" /></datalist>
        <small className="tutor-local-help">Для OpenAI API: <code>gpt-6.1-sol</code> — баланс качества и цены, <code>gpt-6-luna</code> — более экономичный вариант, <code>gpt-6-astra</code> — сложные задачи. Доступность модели зависит от вашего API-аккаунта; <a href="https://developers.openai.com/api/docs/models" target="_blank" rel="noopener noreferrer">актуальный каталог OpenAI</a>.</small>
        {cloudModelError && <small className="tutor-local-help error">{cloudModelError}</small>}
        <label className="tutor-label" htmlFor="cloud-provider-advanced">Имя расширенной модели (необязательно)</label>
        <input className="tutor-local-input" style={localInputStyle} id="cloud-provider-advanced" list="openai-model-suggestions" value={cloudAdvancedModel} onChange={event => setCloudAdvancedModel(event.target.value)} maxLength={128} placeholder="имя модели для подробных режимов" disabled={cloudBusy} />
        <label className="tutor-label" htmlFor="cloud-provider-api-key">API-ключ</label>
        <small className="tutor-local-help">Ключ передаётся в Authorization только указанному HTTPS-хосту. Не используй ключ OpenAI на endpoint стороннего провайдера, которому не доверяешь.</small>
        <input className="tutor-local-input" style={localInputStyle} id="cloud-provider-api-key" type="password" autoComplete="new-password" value={cloudApiKey} onChange={event => setCloudApiKey(event.target.value)} maxLength={4096} placeholder={config.cloud_provider_configured ? 'Для замены введи новый ключ' : 'Вставь API-ключ провайдера'} disabled={cloudBusy} aria-describedby="cloud-key-help" required />
        <small id="cloud-key-help" className={cloudKeyError && cloudApiKey ? 'tutor-local-help error' : 'tutor-local-help'}>{cloudKeyError || 'Ключ отправляется только локальному API и сохраняется в серверном файле с правами 0600. Он не возвращается браузеру.'}</small>
        {cloudError && <p className="tutor-error" role="alert">{cloudError}</p>}
        {cloudStatus && <p className="tutor-local-status" role="status">{cloudStatus}</p>}
        <div className="tutor-submit-row"><span>Сервер проверяет URL на безопасный HTTPS публичный адрес, но не отправляет HTTP-запрос модели. Первый облачный запрос отправится только после отдельного согласия в чате и может тарифицироваться у провайдера.</span><span className="tutor-local-actions"><button className="tutor-button" disabled={cloudBusy || Boolean(cloudEndpointError) || Boolean(cloudModelError) || Boolean(cloudKeyError)}>{cloudBusy ? 'Сохраняем…' : 'Сохранить облачные настройки'}</button>{config.cloud_provider_configured && <button type="button" className="tutor-button secondary" disabled={cloudBusy} onClick={() => void removeCloudProvider()}>{cloudBusy ? 'Подождите…' : 'Удалить облачную настройку'}</button>}</span></div>
      </form>
    </section>}
    <section className="tutor-budget" aria-label="Лимит запросов">
      <div><strong>Дневной лимит (UTC, {config.usage.day})</strong><span>{number(config.usage.used_tokens)} использовано из {number(config.daily_budget)} токенов</span></div>
      <progress max={Math.max(1, config.daily_budget)} value={Math.min(config.daily_budget, config.usage.used_tokens + config.usage.reserved_tokens)} aria-label="Использование дневного бюджета" />
      <p>{number(config.usage.reserved_tokens)} токенов зарезервировано · {number(config.usage.estimated_requests)} запросов с оценочным расходом · {number(config.usage.requests)} отправлено · {number(config.usage.failed_requests)} не завершилось. На запрос резервируется запас до трёх вызовов провайдера. Ошибка или тайм-аут списывает резерв целиком; ручная повторная отправка создаёт новый резерв. Это счётчик токенов, а не цена в деньгах. Свободно примерно {number(remainingTokens)} токенов; лимит проверяется до обращения к провайдеру.</p>
    </section>
    {!config.enabled ? <section className="tutor-disabled"><h2>Наставник сейчас недоступен</h2><p>ИИ-запросы отключены в настройках этого учебного сервиса. Все материалы курса, редактор и проверки заданий продолжают работать без ИИ.</p><p>Подключи локальный провайдер без API-ключа или облачный с отдельным ключом. Настройка облака не отправляет запрос; первый вызов произойдёт только после согласия в чате.</p></section> : <form className="tutor-form" onSubmit={send}>
      <label className="tutor-label" htmlFor="tutor-mode">Чем помочь</label>
      <select id="tutor-mode" value={mode} onChange={event => { setMode(event.target.value as Mode); setConsent(false); }} disabled={sending || allowedModes.length === 0}>{allowedModes.map(item => <option key={item} value={item}>{modeInfo[item].label}</option>)}</select>
      <p className="tutor-mode-help">{modeInfo[mode].description}</p>
      {(mode === 'quiz' || mode === 'interview') && <div className="tutor-history-controls"><p>Вопросы и ответы этой вкладочной беседы можно передать наставнику вместе с каждым запросом после отдельного согласия ниже. История ограничена 8 сообщениями и 12 КиБ.</p><button type="button" className="tutor-button secondary" onClick={() => { setHistory([]); setReply(null); setConsent(false); }} disabled={sending}>Начать новый диалог</button></div>}
      {allowedModes.length === 0 && <p className="tutor-unavailable" role="status">Сервер не включил режимы наставника. Проверь настройки позже.</p>}
      <label className="tutor-label" htmlFor="tutor-message">Твой вопрос</label>
      <textarea id="tutor-message" rows={5} maxLength={messageLimit} value={message} onChange={event => { setMessage(event.target.value); setConsent(false); }} placeholder="Например: объясни, почему функция возвращает значение типа int" disabled={sending || allowedModes.length === 0} required />
      <div className="tutor-count" aria-live="polite">{message.length} / {messageLimit} символов</div>
      <p className="tutor-context-note">{lessonId ? <>К контексту добавится ограниченный публичный текст открытого урока <code>{lessonId}</code> (до 56 КиБ), точная версия курса Go, цели и словарь; сервер может найти ещё до двух релевантных опубликованных уроков.</> : 'Без открытого урока сервер может найти до трёх релевантных опубликованных уроков по вопросу.'} Черновики, приватные ответы и скрытые тесты не используются.</p>
      {source !== undefined && <label className="tutor-consent tutor-code-choice"><input type="checkbox" checked={shareCode} onChange={event => { setShareCode(event.target.checked); setConsent(false); }} disabled={sending || !canShareSource} /><span><strong>Передать код редактора</strong><small>{canShareSource ? `${number(sourceBytes)} байт будут отправлены провайдеру вместе с вопросом.` : `Код занимает ${number(sourceBytes)} байт и превышает лимит 8 КиБ; отправка кода недоступна.`}</small></span></label>}
      <label className="tutor-consent tutor-code-choice"><input type="checkbox" checked={shareDiagnostics} onChange={event => { setShareDiagnostics(event.target.checked); setConsent(false); }} disabled={sending || !canShareDiagnostics} /><span><strong>Передать вывод/диагностику последнего запуска</strong><small>{!lastResult ? 'Сначала запусти или проверь задание.' : canShareDiagnostics ? `Будут переданы только статус, код завершения, stdout, stderr и количество проверок (${number(diagnosticsBytes)} из ${number(diagnosticsLimit)} байт).` : `Диагностика занимает ${number(diagnosticsBytes)} байт и превышает лимит 4 КиБ.`}</small></span></label>
      {lastResult && <details className="tutor-preview"><summary>Предпросмотр диагностики</summary><pre>{diagnosticsJSON}</pre></details>}
      <label className="tutor-consent tutor-code-choice"><input type="checkbox" checked={shareProgress} onChange={event => { setShareProgress(event.target.checked); setConsent(false); }} disabled={sending} /><span><strong>Передать краткий прогресс и навыки</strong><small>Если включить, сервер сам сформирует краткий снимок учебного прогресса. Браузер не отправляет список уроков, оценок или навыков.</small></span></label>
      <details className="tutor-preview"><summary>Посмотреть, что будет отправлено</summary><dl><dt>Режим</dt><dd>{modeInfo[mode]?.label || mode}</dd><dt>Место обработки</dt><dd>{providerLocation}{config.provider_label ? ` · ${config.provider_label}` : ''}</dd><dt>Сообщение</dt><dd className="tutor-preview-text">{message.trim() || 'Пока не заполнено'}</dd>{(mode === 'quiz' || mode === 'interview') && <><dt>История беседы</dt><dd>{history.length ? <ol>{boundedHistory(history).map((item, i) => <li key={`${i}-${item.role}`}><strong>{item.role === 'assistant' ? 'Наставник' : 'Ты'}:</strong> {item.content}</li>)}</ol> : 'Предыдущих сообщений нет.'} {history.length ? 'История будет передана только с этим запросом после согласия ниже.' : 'История пока не будет отправлена.'}</dd></>}<dt>Контекст курса</dt><dd>{lessonId ? <>Публичный текст открытого урока <code>{lessonId}</code> (до 56 КиБ), точная версия курса Go, цели и словарь, плюс до двух релевантных опубликованных уроков.</> : 'До трёх релевантных опубликованных уроков по вопросу.'} Черновики, приватные ответы и скрытые тесты не используются.</dd>{source !== undefined && <><dt>Код редактора</dt><dd>{shareCode ? <pre>{source}</pre> : 'Не будет отправлен: отдельный opt-in выключен.'}</dd></>}{lastResult && <><dt>Диагностика</dt><dd>{shareDiagnostics ? 'Будет отправлен показанный выше краткий результат.' : 'Не будет отправлена: отдельный opt-in выключен.'}</dd></>}<dt>Прогресс</dt><dd>{shareProgress ? 'Браузер просит сервер добавить краткий снимок прогресса.' : 'Не будет добавлен: отдельный opt-in выключен.'}</dd></dl></details>
      <label className="tutor-consent"><input type="checkbox" checked={consent} onChange={event => setConsent(event.target.checked)} disabled={sending || localBusy || cloudBusy} /><span><strong>Я прочитал сведения о передаваемых данных и согласен отправить этот запрос.</strong><small>Согласие действует для версии текущего провайдера. Запрос может расходовать дневной лимит и учитываться у выбранного провайдера согласно его правилам.</small></span></label>
      {error && <p className="tutor-error" role="alert">{error}</p>}
      <div className="tutor-submit-row"><span>Максимальный ответ: {number(config.max_output_tokens)} токенов. ИИ не запускает код и не подтверждает учебный ранг.</span><button className="tutor-button" disabled={sending || localBusy || cloudBusy || !consent || !message.trim() || message.length > messageLimit || allowedModes.length === 0 || (shareCode && !canShareSource) || (shareDiagnostics && !canShareDiagnostics)}>{sending ? 'Ждём ответ…' : 'Отправить запрос'}</button></div>
    </form>}
    {reply && <section className="tutor-reply" aria-live="polite"><div className="tutor-reply-heading"><h2>Ответ наставника</h2><span>{reply.model}</span></div><div className="tutor-markdown"><ReactMarkdown components={{ img: ({ alt }) => <span className="tutor-image-suppressed">{alt || 'Внешнее изображение скрыто.'}</span>, a: ({ href, children }) => <a href={href} target="_blank" rel="noopener noreferrer">{children}</a> }}>{reply.text}</ReactMarkdown></div>{reply.context_lessons?.length ? <div className="tutor-context-note"><strong>Уроки, использованные для контекста</strong><ul>{reply.context_lessons.map(lesson => <li key={lesson.id}><code>{lesson.id}</code> — {lesson.title}</li>)}</ul></div> : null}<p className="tutor-reply-meta">{number(reply.input_tokens)} входных · {number(reply.output_tokens)} выходных токенов{reply.estimated ? ' · оценка расхода' : ''}{reply.cached ? ' · ответ из кэша' : ''}. Остаток бюджета уточнён сервером.</p><p className="tutor-rank-note">Используй редактор и проверки курса, чтобы проверить код. Ответ ИИ сам по себе не запускает программу, не меняет прогресс и не подтверждает ранг.</p></section>}
  </main>;
}

function number(value: number) { return new Intl.NumberFormat('ru-RU').format(value); }
function errorText(error: unknown) { return error instanceof Error ? error.message : 'неизвестная ошибка'; }
function isApiStatus(error: unknown, status: number) {
  return error !== null && typeof error === 'object' && 'status' in error && error.status === status;
}
function providerFingerprint(config: TutorConfig) {
  return [config.provider_revision, config.enabled, config.provider_location, config.provider_label, config.models.economy, config.models.advanced, config.local_provider?.endpoint, config.local_provider?.model, config.local_provider?.advanced_model, config.cloud_provider?.endpoint, config.cloud_provider?.model, config.cloud_provider?.advanced_model].join('|');
}
function boundedHistory(history: TutorHistoryMessage[]): TutorHistoryMessage[] {
  const pairs: [TutorHistoryMessage, TutorHistoryMessage][] = [];
  for (let i = 0; i + 1 < history.length; i++) {
    if (history[i].role === 'user' && history[i + 1].role === 'assistant') {
      pairs.push([
        { role: 'user', content: Array.from(history[i].content).slice(0, 2000).join('') },
        { role: 'assistant', content: Array.from(history[i + 1].content).slice(0, 2000).join('') },
      ]);
      i++;
    }
  }
  const selected: TutorHistoryMessage[] = [];
  for (const pair of pairs.slice(-historyMessageLimit / 2).reverse()) {
    let fitted = pair;
    const candidate = [...fitted, ...selected];
    if (new TextEncoder().encode(JSON.stringify(candidate)).length > historyLimit) {
      let low = 1, high = 2000, best = 0;
      while (low <= high) {
        const limit = Math.floor((low + high) / 2);
        const clipped = pair.map(item => ({ ...item, content: Array.from(item.content).slice(0, limit).join('') })) as [TutorHistoryMessage, TutorHistoryMessage];
        if (new TextEncoder().encode(JSON.stringify([...clipped, ...selected])).length <= historyLimit) { best = limit; low = limit + 1; }
        else high = limit - 1;
      }
      if (!best) break;
      fitted = pair.map(item => ({ ...item, content: Array.from(item.content).slice(0, best).join('') })) as [TutorHistoryMessage, TutorHistoryMessage];
    }
    selected.unshift(...fitted);
  }
  return selected;
}
const localInputStyle = { width: '100%', minWidth: 0, padding: '10px 11px', border: '1px solid var(--line)', borderRadius: 6, background: 'var(--paper)', color: 'var(--ink)', font: '12px/1.6 Inter, ui-sans-serif, system-ui, sans-serif' };
function loopbackEndpointError(value: string): string {
  try {
    const url = new URL(value);
    const host = url.hostname.toLowerCase();
    if (url.protocol !== 'http:' || url.username || url.password || url.search || url.hash || !['localhost', '127.0.0.1', '[::1]', '::1'].includes(host)) {
      return 'Нужен HTTP endpoint без логина, пароля, query и fragment на localhost, 127.0.0.1 или ::1.';
    }
    return '';
  } catch { return 'Введи корректный URL, например http://127.0.0.1:1234/v1.'; }
}
function cloudEndpointErrorFor(value: string): string {
  try {
    const url = new URL(value);
    if (url.protocol !== 'https:' || url.username || url.password || url.search || url.hash) return 'Нужен HTTPS endpoint без логина, пароля, query и fragment.';
    return '';
  } catch { return 'Введи корректный HTTPS URL, например https://api.openai.com/v1.'; }
}
