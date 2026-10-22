import { lazy, Suspense, useCallback, useEffect, useMemo, useState, isValidElement, type ReactNode } from 'react';
import ReactMarkdown, {defaultUrlTransform} from 'react-markdown';
import { api, ApiError, loadProfile, saveProfile, type Exercise, type Lesson, type LearningSnapshot, type LessonCheck, type Profile, type Progress } from './api';
import ProjectWorkspace, { type ProjectSummary } from './ProjectWorkspace';
import Exams from './Exams';
import Tutor from './Tutor';
import Passport, { CredentialVerification, PublicPassport } from './Passport';
import { Blockquote, CheckLine, TaskCard, TaskChips } from './TaskCard';
const GoEditor = lazy(() => import('./GoEditor'));

const ranks = ['Zero', 'Junior', 'Junior+', 'Middle', 'Middle+', 'Senior'];
type RunResult = { stdout?: string; stderr?: string; exit_code?: number; status?: string; passed?: boolean; checks?: LessonCheck[]; files?: Record<string, string> };

export default function App() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [profileLoaded, setProfileLoaded] = useState(false);
  const [screen, setScreen] = useState<'home' | 'lesson' | 'path' | 'skills' | 'course' | 'projects' | 'project' | 'exams' | 'tutor' | 'passport'>('home');
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [selectedProject, setSelectedProject] = useState<ProjectSummary | null>(null);
  const [projectsLoading, setProjectsLoading] = useState(false);
  const [lessons, setLessons] = useState<Lesson[]>([]);
  const [learning, setLearning] = useState<LearningSnapshot | null>(null);
  const [progress, setProgress] = useState<Progress>({ completed: [], attempts: [], xp: 0 });
  const [lesson, setLesson] = useState<Lesson | null>(null);
  const [selectedExercise, setSelectedExercise] = useState<Exercise | null>(null);
  const [files, setFiles] = useState<Record<string, string>>({});
  const [activeFile, setActiveFile] = useState('main.go');
  const [result, setResult] = useState<RunResult | null>(null);
  const [shownSelfChecks, setShownSelfChecks] = useState<Record<string, boolean[]>>({});
  const [solution, setSolution] = useState<string | null>(null);
  const [solutionBusy, setSolutionBusy] = useState(false);
  const [themeBusy, setThemeBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [saveState, setSaveState] = useState('');
  const [focus, setFocus] = useState(false);
  const [split, setSplit] = useState(57);
  const [theme, setTheme] = useState<'light' | 'dark'>((localStorage.getItem('go-academy-theme') as 'light' | 'dark') || 'light');
  const [courseQuery, setCourseQuery] = useState('');
  const [courseResults, setCourseResults] = useState<Lesson[]>([]);
  const [courseLoading, setCourseLoading] = useState(false);

  useEffect(() => { document.documentElement.dataset.theme = theme; localStorage.setItem('go-academy-theme', theme); }, [theme]);
  const refresh = useCallback(async () => {
    setError('');
    try {
      const [nextLessons, nextProgress, nextLearning] = await Promise.all([api<Lesson[]>('/lessons'), api<Progress>('/progress'), api<LearningSnapshot>('/learning')]);
      setLessons(nextLessons); setProgress(nextProgress); setLearning(nextLearning);
    } catch (e) { setError(`Не удалось загрузить курс: ${message(e)}`); }
  }, []);
  useEffect(() => {
    let active = true;
    loadProfile().then(async value => {
      if (!active) return;
      setProfile(value);
      if (value) setTheme(value.theme);
      setProfileLoaded(true);
      await refresh();
    }).catch(e => { if (active) { setError(`Не удалось подключиться к серверу: ${message(e)}`); setProfileLoaded(true); } });
    return () => { active = false; };
  }, [refresh]);

  const openLesson = async (id: string, exerciseId?: string) => {
    setError(''); setNotice(''); setResult(null); setBusy(true);
    try {
      const loaded = await api<Lesson>(`/lessons/${encodeURIComponent(id)}`);
      setLesson(loaded); setSelectedExercise(loaded.exercises?.find(exercise => exercise.id === exerciseId) ?? loaded.exercises?.[0] ?? null); setScreen('lesson');
    } catch (e) { setError(`Не удалось открыть урок: ${message(e)}`); }
    finally { setBusy(false); }
  };
  useEffect(() => {
    if (!lesson || !selectedExercise) return;
    let alive = true; setFiles({}); setActiveFile(selectedExercise.starter_file || 'main.go'); setSaveState('Загружаем черновик…');
    api<{ files?: Record<string, string> }>(`/drafts/${encodeURIComponent(selectedExercise.id)}`).then(draft => {
      if (!alive) return;
      const initial = draft.files && Object.keys(draft.files).length ? draft.files : { [selectedExercise.starter_file || 'main.go']: starter(selectedExercise) };
      setFiles(initial); setActiveFile(Object.keys(initial)[0]); setSaveState('');
    }).catch(e => { if (alive) { setFiles({ [selectedExercise.starter_file || 'main.go']: starter(selectedExercise) }); setSaveState(`Черновик не загружен: ${message(e)}`); } });
    return () => { alive = false; };
  }, [lesson?.id, selectedExercise?.id]);

  const saveDraft = useCallback(async () => {
    if (!selectedExercise || !Object.keys(files).length) return;
    setSaveState('Сохраняем…');
    try {
      await api(`/drafts/${encodeURIComponent(selectedExercise.id)}`, { method: 'PUT', body: JSON.stringify({ files }) });
      setSaveState('Сохранено');
    } catch (e) { setSaveState(`Не сохранено: ${message(e)}`); }
  }, [files, selectedExercise?.id]);

  useEffect(() => {
    if (!selectedExercise || !Object.keys(files).length || saveState.startsWith('Черновик не загружен')) return;
    setSaveState('Сохраняем…');
    const timer = window.setTimeout(() => { void saveDraft(); }, 650);
    return () => window.clearTimeout(timer);
  }, [saveDraft]);

  useEffect(() => {
    const retryWhenOnline = () => {
      if (saveState.startsWith('Не сохранено:')) void saveDraft();
    };
    window.addEventListener('online', retryWhenOnline);
    return () => window.removeEventListener('online', retryWhenOnline);
  }, [saveState, saveDraft]);

  useEffect(() => {
    if (screen !== 'course') return;
    let current = true;
    const timer = window.setTimeout(() => {
      setCourseLoading(true);
      api<Lesson[]>(`/lessons?q=${encodeURIComponent(courseQuery.trim())}`)
        .then(items => { if (current) setCourseResults(items); })
        .catch(e => { if (current) setError(`Не удалось найти уроки: ${message(e)}`); })
        .finally(() => { if (current) setCourseLoading(false); });
    }, 250);
    return () => { current = false; window.clearTimeout(timer); };
  }, [screen, courseQuery]);

  useEffect(() => {
    if (screen !== 'projects') return;
    let current = true;
    setProjectsLoading(true);
    api<ProjectSummary[]>('/projects')
      .then(items => { if (current) setProjects(items); })
      .catch(e => { if (current) setError(`Не удалось загрузить проекты: ${message(e)}`); })
      .finally(() => { if (current) setProjectsLoading(false); });
    return () => { current = false; };
  }, [screen]);

  const updateCode = (code: string) => setFiles(current => ({ ...current, [activeFile]: code }));
  const perform = async (operation: 'run' | 'submit' | 'format') => {
    if (!selectedExercise) return;
    setBusy(true); setError(''); setNotice(''); setResult(null);
    try {
      const response = await api<RunResult>(`/exercises/${encodeURIComponent(selectedExercise.id)}/${operation}`, { method: 'POST', body: JSON.stringify({ files }) });
      setResult(response);
      if (response.files && operation === 'format') { setFiles(response.files); setNotice('Форматирование завершено.'); }
      if (operation === 'submit' && response.passed) { setNotice('Задание принято. Прогресс обновлён.'); await refresh(); }
    } catch (e) {
      // 422 means the code was not admitted to the check (for example a forbidden import): an answer, shown in the result (§1.6).
      if (e instanceof ApiError && e.status === 422) setResult({ status: 'rejected', passed: false, stdout: '', stderr: e.message, checks: [] });
      else setError(`Не удалось выполнить действие: ${message(e)}`);
    }
    finally { setBusy(false); }
  };
  const revealSolution = async () => {
    if (!selectedExercise) return;
    setSolutionBusy(true); setError('');
    try {
      const response = await api<{ source: string }>(`/exercises/${encodeURIComponent(selectedExercise.id)}/solution`);
      setSolution(response.source);
    } catch (e) { setError(`Не удалось загрузить решение: ${message(e)}`); }
    finally { setSolutionBusy(false); }
  };
  const toggleSelfCheck = (scope: string, index: number, shown: boolean) => setShownSelfChecks(state => {
    const answers = [...(state[scope] || [])];
    answers[index] = !shown;
    return { ...state, [scope]: answers };
  });
  const resetCode = () => {
    if (!selectedExercise) return;
    const path = selectedExercise.starter_file || 'main.go'; setFiles({ [path]: starter(selectedExercise) }); setActiveFile(path); setResult(null); setNotice('Черновик заменён начальным кодом.');
  };
  const saveLearner = async (name: string, learnerTheme: 'light' | 'dark') => {
    setBusy(true); setError('');
    try { const { profile: saved } = await saveProfile({ name, theme: learnerTheme }, 'POST'); setProfile(saved); setTheme(saved.theme); await refresh(); }
    catch (e) { setError(`Профиль не сохранён: ${message(e)}`); }
    finally { setBusy(false); }
  };
  const changeTheme = async () => {
    if (!profile || themeBusy) return;
    const nextTheme = theme === 'light' ? 'dark' : 'light';
    setThemeBusy(true); setError('');
    try {
      const { profile: saved } = await saveProfile({ name: profile.name, theme: nextTheme }, 'PUT');
      setProfile(saved); setTheme(saved.theme);
    } catch (e) { setError(`Тема не сохранена: ${message(e)}`); }
    finally { setThemeBusy(false); }
  };

  const rankNames = learning?.ranks.map(rank => rank.name) ?? ranks;
  const currentRank = learning?.current_rank || profile?.rank || 'Zero';
  const currentIndex = Math.max(0, rankNames.indexOf(currentRank));
  const completedLessons = useMemo(() => lessons.filter(item => item.exercises?.length && item.exercises.every(exercise => progress.completed.includes(exercise.id))), [lessons, progress.completed]);
  const completedLessonIds = useMemo(() => new Set(completedLessons.map(item => item.id)), [completedLessons]);
  const nextLesson = useMemo(() => lessons.find(item => !completedLessonIds.has(item.id)), [lessons, completedLessonIds]);
  const exerciseLocations = useMemo(
    () => new Map(lessons.flatMap(item => (item.exercises || []).map(exercise => [exercise.id, { lessonId: item.id, title: exercise.title }] as const))),
    [lessons],
  );

  const followCourseLink = async (href: string) => {
    let target: {lessonId:string;exerciseId?:string;fragment?:string}|undefined;
    if(href.startsWith('lesson:'))target={lessonId:href.slice(7)};
    else if(href.startsWith('exercise:')){const found=exerciseLocations.get(href.slice(9));if(found)target={lessonId:found.lessonId,exerciseId:href.slice(9)}}
    else {const [file,fragment]=href.split('#');const found=lessons.find(item=>item.id+'.md'===file.split('/').pop());if(found)target={lessonId:found.id,fragment}}
    if(!target)return;await openLesson(target.lessonId,target.exerciseId);if(target.fragment)window.requestAnimationFrame(()=>window.requestAnimationFrame(()=>document.getElementById(target!.fragment!)?.scrollIntoView()));
  };
  const isCourseLink = (href?:string) => Boolean(href && (href.startsWith('lesson:') || href.startsWith('exercise:') || href.split('#')[0].endsWith('.md')));
  if (/^\/verify\/[a-f0-9]{32}\/?$/.test(window.location.pathname)) return <CredentialVerification credentialId={window.location.pathname.split('/')[2]} />;
  if (window.location.pathname === '/passport') return <PublicPassport />;
  if (!profileLoaded) return <main className="boot"><span className="go-mark">go</span><p>Открываем учебную мастерскую…</p></main>;
  if (!profile) return <Onboarding onSave={saveLearner} busy={busy} error={error} theme={theme} setTheme={setTheme} />;

  return <div className={focus && screen === 'lesson' ? 'app focus-mode' : 'app'}>
    <header className="topbar">
      <a className="brand" href="#home" onClick={e => { e.preventDefault(); setScreen('home'); }} aria-label="Open Go Academy — главная"><span className="go-mark">go</span><span>Open Go Academy</span></a>
      <nav className="main-nav" aria-label="Главная навигация">
        <button className={screen === 'home' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('home')}>Обзор</button>
        <button className={screen === 'path' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('path')}>Путь обучения</button>
        <button className={screen === 'skills' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('skills')}>Навыки</button>
        <button className={screen === 'course' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('course')}>Курс</button>
        <button className={screen === 'projects' || screen === 'project' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('projects')}>Проекты</button>
        <button className={screen === 'exams' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('exams')}>Экзамены</button>
        <button className={screen === 'tutor' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('tutor')}>ИИ-наставник</button>
        <button className={screen === 'passport' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('passport')}>Паспорт</button>
        <button className={screen === 'lesson' ? 'nav-link active' : 'nav-link'} onClick={() => setScreen('lesson')} disabled={!lesson}>Урок</button>
      </nav>
      <div className="top-actions"><span className="profile-chip"><span className="avatar">{profile.name.slice(0, 1).toUpperCase()}</span>{profile.name}</span><button className="icon-button" onClick={() => void changeTheme()} disabled={themeBusy} aria-label={theme === 'light' ? 'Включить тёмную тему' : 'Включить светлую тему'} title="Сменить тему">◐</button></div>
    </header>
    {(error || notice) && <div className={error ? 'banner error-banner' : 'banner notice-banner'} role={error ? 'alert' : 'status'}>{error || notice}<button onClick={() => { setError(''); setNotice(''); }} aria-label="Закрыть сообщение">×</button></div>}
    {screen === 'home' && <main className="page dashboard">
      <div className="welcome"><div><p className="intro-line">С возвращением, {profile.name}</p><h1>Твоя практика начинается здесь.</h1><p className="lede">Пиши Go, запускай код и двигайся шаг за шагом — от первой строки.</p></div><div className="go-illustration" aria-hidden="true"><span>go</span><i>func main()</i><i>fmt.Println(…)</i></div></div>
      <section className="progress-strip" aria-label="Текущий прогресс"><div><span className="metric-label">Учебный ранг</span><strong>{currentRank}</strong></div><div><span className="metric-label">Опыт</span><strong>{progress.xp} <small>XP</small></strong></div><div><span className="metric-label">Завершено уроков</span><strong>{completedLessons.length} <small>/ {lessons.length}</small></strong></div><div className="rank-track" aria-label={`Уровень ${currentIndex + 1} из ${rankNames.length}`}>{rankNames.map((rank, i) => <span key={rank} className={i <= currentIndex ? 'rank-dot reached' : 'rank-dot'} title={rank} />)}</div></section>
      <div className="dashboard-grid"><section className="next-step"><div className="section-heading"><div><p className="eyebrow">ТВОЙ СЛЕДУЮЩИЙ ШАГ</p><h2>Продолжить обучение</h2></div><button className="text-button" onClick={() => setScreen('path')}>Вся карта</button></div>
        {nextLesson ? <button className="lesson-row" onClick={() => void openLesson(nextLesson.id)}><span className="lesson-icon">{nextLesson.rank === 'Zero' ? '01' : '↗'}</span><span className="lesson-info"><strong>{nextLesson.title}</strong><small>{moduleTitle(nextLesson.module)} · {nextLesson.estimated_minutes} минут</small></span><span className="lesson-rank">{nextLesson.rank}</span><span className="row-chevron">›</span></button> : lessons.length ? <p className="empty-state">Все доступные уроки завершены. Следующие материалы появятся по мере разработки курса.</p> : <p className="empty-state">Уроки не загрузились. Обнови страницу после запуска API.</p>}
        {lessons.length > 0 && <div className="recent-lessons"><h3>Уроки курса</h3>{lessons.slice(0, 5).map(item => <button className="compact-lesson" key={item.id} onClick={() => void openLesson(item.id)}><span className={completedLessonIds.has(item.id) ? 'check-mark done' : 'check-mark'}>{completedLessonIds.has(item.id) ? '✓' : '·'}</span><span>{item.title}</span><small>{item.rank}</small></button>)}</div>}
        <button className="course-link" onClick={() => setScreen('course')}>Искать по всем урокам</button>
        {learning?.review_exercises.length ? <section className="review-list"><h3>К заданиям на повторение</h3><p>Сервер отметил эти задания для дополнительной практики.</p>{learning.review_exercises.map(id => { const location = exerciseLocations.get(id); return location ? <button className="review-item" key={id} onClick={() => void openLesson(location.lessonId, id)}><span>{location.title}</span><small>Открыть задание ›</small></button> : <p className="review-unknown" key={id}>Задание {id} пока недоступно в каталоге уроков.</p>; })}</section> : null}
      </section><aside className="rank-card"><p className="eyebrow">МАРШРУТ</p><h2>От Zero к Senior</h2><p>Шесть уровней. Каждый следующий открывается через практику и проверку навыков.</p><ol className="mini-roadmap">{rankNames.map((rank, i) => <li key={rank} className={i === currentIndex ? 'current' : i < currentIndex ? 'past' : ''}><span className="road-dot">{i < currentIndex ? '✓' : ''}</span><span>{rank}</span>{i > currentIndex && <small>позже</small>}</li>)}</ol><button className="outline-button" onClick={() => setScreen('path')}>Изучить маршрут</button></aside></div>
      <footer className="page-foot">Практические навыки растут с каждой запущенной программой.</footer>
    </main>}
    {screen === 'path' && <main className="page path-page"><div className="page-heading"><p className="eyebrow">КАРТА ОБУЧЕНИЯ</p><h1>Путь обучения</h1><p>Критерии, навыки и доступность материалов приходят из учебного плана сервера.</p></div><p className="rank-trust-note" role="status">Текущий учебный ранг: <strong>{learning?.current_rank || currentRank}</strong> — {learning?.verified ? 'подтверждён по критериям курса.' : 'пока не подтверждён как достигнутый ранг.'} Баллы опыта показывают практику, но сами по себе не подтверждают ранг.</p><div className="path-list">{learning?.ranks.map((rank, i) => { const content = lessons.filter(item => item.rank === rank.name); const current = rank.name === learning.current_rank; return <section className={`path-level ${current ? 'path-current' : ''}`} key={rank.name}><span className="path-index">{String(i + 1).padStart(2, '0')}</span><div className="path-copy"><div><h2>{rank.name}</h2><span className="path-status">{rank.content_available ? `${content.length} ${content.length === 1 ? 'урок доступен' : 'уроков доступно'}` : 'Ещё не опубликован'}</span>{current && <span className="path-status">Текущий уровень</span>}</div>{rank.criteria.length > 0 && <details className="path-details"><summary>Критерии уровня</summary><ul>{rank.criteria.map(item => <li key={item}>{item}</li>)}</ul></details>}{rank.skills.length > 0 && <details className="path-details"><summary>Навыки уровня</summary><ul>{rank.skills.map(id => <li key={id}>{learning.skills.find(skill => skill.id === id)?.title || id}</li>)}</ul></details>}{rank.assessment_available ? <p className="assessment-status">Проверка уровня доступна.</p> : <p className="assessment-status">Проверка ранга пока недоступна.</p>}{content.map(item => <button className="path-lesson" key={item.id} onClick={() => void openLesson(item.id)}>{item.title}<span>Открыть урок ›</span></button>)}</div>{i < learning.ranks.length - 1 && <span className="path-connector" aria-hidden="true" />}</section>; }) || <p className="empty-state">Учебный маршрут не загрузился.</p>}</div></main>}
    {screen === 'skills' && <main className="page skills-page"><div className="page-heading"><p className="eyebrow">НАБЛЮДЕНИЯ ПО ЗАДАНИЯМ</p><h1>Карта навыков</h1><p>Уровни рассчитывает учебный сервис по завершённым заданиям. Изменить уровень вручную нельзя.</p></div><p className="rank-trust-note" role="status">Учебный ранг <strong>{learning?.current_rank || currentRank}</strong> {learning?.verified ? 'подтверждён по критериям курса.' : 'не подтверждён как достигнутый ранг.'} XP не заменяет проверку ранга.</p><section className="skill-scale" aria-labelledby="skill-scale-title"><h2 id="skill-scale-title">Как читать шкалу 0–4</h2><ol>{['Пока нет проверенных свидетельств', 'Понимает термин', 'Выполняет базовые упражнения', 'Самостоятельно решает типовые задачи', 'Решает сложные задачи и обосновывает компромиссы'].map((meaning, level) => <li key={level}><strong>{level}</strong><span>{meaning}</span></li>)}</ol></section>{learning?.skills.length ? <div className="skill-table-wrap" tabIndex={0} aria-label="Таблица навыков; на узком экране её можно прокручивать по горизонтали"><p className="skill-table-hint">На узком экране прокрутите таблицу вправо, чтобы увидеть ранг и свидетельства.</p><table className="skill-table"><caption>Навыки, учебный уровень и подтверждающие задания</caption><thead><tr><th scope="col">Навык</th><th scope="col">Уровень</th><th scope="col">Ранг</th><th scope="col">Свидетельства</th></tr></thead><tbody>{learning.skills.map(skill => <tr key={skill.id}><th scope="row">{skill.title}</th><td><span className="skill-level">{skill.level} / 4</span><span className="skill-meter" aria-hidden="true">{[0, 1, 2, 3].map(step => <i key={step} className={step < skill.level ? 'filled' : ''} />)}</span></td><td>{skill.rank}</td><td>{skill.evidence.length ? <ul className="evidence-list">{skill.evidence.map(id => { const location = exerciseLocations.get(id); const label = skill.id === 'communication' && id.startsWith('sha256:') ? 'P5: итоговое ручное ревью Senior' : id; return <li key={id}>{location ? <button onClick={() => void openLesson(location.lessonId, id)}>{location.title}</button> : <span title={id}>{label}</span>}</li>; })}</ul> : <span className="no-evidence">Пока нет заданий</span>}</td></tr>)}</tbody></table></div> : <p className="empty-state">Сведения о навыках не загрузились.</p>}</main>}
    {screen === 'course' && <main className="page course-page"><div className="page-heading"><p className="eyebrow">КАТАЛОГ ОПУБЛИКОВАННОГО</p><h1>Курс</h1><p>Выбирай тему и продолжай с нужного места. Прогресс сохраняется автоматически.</p></div><label className="course-search-label" htmlFor="course-search">Найти уроки</label><input id="course-search" className="course-search" type="search" value={courseQuery} onChange={event => setCourseQuery(event.target.value)} placeholder="Например, программа или ошибки" /><div className="course-results" aria-live="polite">{courseLoading ? <p className="empty-state">Ищем уроки…</p> : courseResults.length ? courseResults.map(item => <button className="course-result" key={item.id} onClick={() => void openLesson(item.id)}><span className="course-result-main"><strong>{item.title}</strong><small>{moduleTitle(item.module)} · {item.estimated_minutes} минут</small></span><span className="course-result-rank">{item.rank}</span><span>Открыть урок ›</span></button>) : <p className="empty-state">По этому запросу уроков не найдено.</p>}</div></main>}

    {screen === 'projects' && <main className="page projects-page"><div className="page-heading"><p className="eyebrow">ПРАКТИКА ЦЕЛЫХ ПРИЛОЖЕНИЙ</p><h1>Проекты</h1><p>Собирай приложение по этапам, сохраняй файлы и запускай проверки в изолированном исполнителе.</p></div>{projectsLoading ? <p className="empty-state">Загружаем каталог проектов…</p> : projects.length ? <div className="project-catalog">{projects.map(project => <button className="project-card" key={project.id} onClick={() => { setSelectedProject(project); setScreen('project'); }}><span className="project-card-rank">{project.rank}</span><strong>{project.title}</strong><span>{project.description}</span><small>Открыть рабочую область ›</small></button>)}</div> : <p className="empty-state">В каталоге пока нет опубликованных проектов.</p>}</main>}
    {screen === 'project' && selectedProject && <ProjectWorkspace key={selectedProject.id} projectId={selectedProject.id} onBack={() => setScreen('projects')} />}
    {screen === 'exams' && <Exams onRankAwarded={refresh} />}
    {screen === 'passport' && <Passport />}
    {screen === 'tutor' && <Tutor lessonId={lesson?.id} source={screen === 'tutor' && lesson ? files[activeFile] : undefined} lastResult={result && (result.status !== undefined || result.exit_code !== undefined || result.stdout !== undefined || result.stderr !== undefined || result.checks !== undefined) ? { status: result.status, exit_code: result.exit_code, stdout: result.stdout, stderr: result.stderr, checks: { total: result.checks?.length ?? 0, passed: result.checks?.filter(check => check.passed).length ?? 0, failed: result.checks?.filter(check => !check.passed).length ?? 0 } } : null} />}

    {screen === 'lesson' && (lesson ? <main className="workspace" style={{ '--split': `${split}%` } as React.CSSProperties}>
      <div className="workspace-toolbar"><button className="back-link" onClick={() => setScreen('home')}>‹ Обзор</button><span className="crumb">{moduleTitle(lesson.module)}</span><span className="toolbar-spacer" /><label className="split-control">Ширина редактора <input aria-label="Ширина редактора" type="range" min="35" max="70" value={split} onChange={e => setSplit(Number(e.target.value))} /></label><button className="quiet-button" onClick={() => setFocus(!focus)}>{focus ? 'Выйти из фокуса' : 'Режим фокуса'}</button></div>
      <div className="work-grid"><article className="lesson-pane"><div className="lesson-meta"><span className="rank-pill">{lesson.rank}</span><span>{lesson.estimated_minutes} минут</span></div><h1 id={headingID(lesson.title)}>{lesson.title}</h1>{lesson.prerequisites?.some(id => !completedLessonIds.has(id)) && <aside className="prerequisite-note" aria-label="Рекомендуемые предыдущие уроки"><strong>Перед этой темой полезно пройти</strong><ul>{lesson.prerequisites.filter(id => !completedLessonIds.has(id)).map(id => <li key={id}><button className="text-button" onClick={() => void openLesson(id)}>{lessons.find(item => item.id === id)?.title || id}</button></li>)}</ul><p>Если тема знакома, можно продолжить. Подтверждённый ранг выдаётся только после экзамена.</p></aside>}{lesson.learning_outcomes?.length ? <section className="outcomes"><h2>После урока ты сможешь</h2><ul>{lesson.learning_outcomes.map(item => <li key={item}>{item}</li>)}</ul></section> : null}<TaskChips exercises={lesson.exercises ?? []} activeId={selectedExercise?.id} completed={new Set(progress.completed)} onSelect={exercise => { setSelectedExercise(exercise); setResult(null); setSolution(null); setNotice(''); document.getElementById(`task-card-${exercise.id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' }); }} /><div className="markdown-body">{lesson.body ? <ReactMarkdown urlTransform={url => /^(lesson|exercise):[a-z0-9-]+$/.test(url) ? url : defaultUrlTransform(url)} components={{ h1: () => null, h2: ({children}) => <h2 id={headingID(children)}>{children}</h2>, h3: ({children}) => <h3 id={headingID(children)}>{children}</h3>, h4: ({children}) => <h4 id={headingID(children)}>{children}</h4>, a: ({href,children}) => isCourseLink(href) ? <button className="markdown-internal-link" type="button" onClick={() => void followCourseLink(href!)}>{children}</button> : <a href={href} target={href?.startsWith('http') ? '_blank' : undefined} rel="noopener noreferrer">{children}</a>, blockquote: Blockquote, pre: ({ children }) => <pre tabIndex={0}>{children}</pre> }}>{lesson.body}</ReactMarkdown> : <p>В этом уроке пока нет теоретического материала.</p>}</div>{lesson.glossary?.length ? <section className="glossary"><h2>Слова урока</h2>{lesson.glossary.map(item => <p key={item.term}><strong>{item.term}</strong> — {item.translation}. {item.definition}</p>)}</section> : null}<div className="exercise-list"><h2>Практика</h2>{lesson.exercises?.length ? lesson.exercises.map((exercise, i) => <button key={exercise.id} className={selectedExercise?.id === exercise.id ? 'exercise-select selected' : 'exercise-select'} onClick={() => { setSelectedExercise(exercise); setResult(null); setSolution(null); setNotice(''); }}><span>{String(i + 1).padStart(2, '0')}</span><span><strong>{exercise.title}</strong><small>{difficultyTitle(exercise.difficulty)}</small></span><span>›</span></button>) : <p>Практические задания для этого урока ещё не добавлены.</p>}
        {lesson.self_check?.length ? <section className="self-check lesson-self-check"><h3>Проверь себя</h3><p>Ответы откроются только по твоему запросу.</p>{lesson.self_check.map((item, index) => { const scope = `lesson:${lesson.id}`; const shown = shownSelfChecks[scope]?.[index] ?? false; return <details className="self-check-item" key={`${scope}-${index}`}><summary>{item.question}</summary><button className="text-button answer-reveal" aria-expanded={shown} onClick={() => toggleSelfCheck(scope, index, shown)}>{shown ? 'Скрыть ответ' : 'Показать ответ'}</button>{shown && <p className="self-check-answer">{item.answer}</p>}</details>; })}</section> : null}
        {selectedExercise?.self_check?.length ? <section className="self-check"><h3>Проверь себя в задании</h3><p>Ответы откроются только по твоему запросу.</p>{selectedExercise.self_check.map((item, index) => { const scope = `exercise:${selectedExercise.id}`; const shown = shownSelfChecks[scope]?.[index] ?? false; return <details className="self-check-item" key={`${scope}-${index}`}><summary>{item.question}</summary><button className="text-button answer-reveal" aria-expanded={shown} onClick={() => toggleSelfCheck(scope, index, shown)}>{shown ? 'Скрыть ответ' : 'Показать ответ'}</button>{shown && <p className="self-check-answer">{item.answer}</p>}</details>; })}</section> : null}</div></article>
      <section className="ide-pane" aria-label="Рабочая область кода"><div className="ide-heading"><div><strong>{selectedExercise?.title || 'Редактор кода'}</strong><span>{selectedExercise?.instructions || selectedExercise?.prompt || 'Выбери задание слева, чтобы начать.'}</span></div><div className="save-status-group"><span className="save-status" role="status">{saveState}</span>{saveState.startsWith('Не сохранено:') && <button className="save-retry" onClick={() => void saveDraft()} aria-label="Повторить сохранение черновика">Повторить сохранение</button>}</div></div>{selectedExercise && <><TaskCard index={(lesson.exercises ?? []).findIndex(item => item.id === selectedExercise.id) + 1} exercise={selectedExercise} body={lesson.body} checks={result?.checks ?? []} lessonId={lesson.id} />{selectedExercise.kind === 'function' && <p className="function-exercise-note">Напиши указанную функцию без main: среда передаст ей входные данные. «Запустить» проверяет открытые примеры; «Проверить» — все сценарии. Результаты показаны как значения: строки заключены в кавычки. Для подробной диагностики ошибки используй «Запустить».</p>}<div className="file-tabs" role="tablist" aria-label="Файлы проекта">{Object.keys(files).map(file => <button role="tab" aria-selected={activeFile === file} key={file} onClick={() => setActiveFile(file)}>{file}</button>)}<span className="file-language">Go</span></div><Suspense fallback={<div className="editor-canvas editor-loading">Загружаем редактор…</div>}><GoEditor value={files[activeFile] || ''} onChange={updateCode} /></Suspense><div className="editor-actions"><button className="quiet-button" onClick={resetCode} disabled={busy}>Сбросить код</button><span /><button className="quiet-button" onClick={() => void perform('format')} disabled={busy}>{busy ? 'Выполняется…' : 'Форматировать'}</button><button className="outline-button run-button" onClick={() => void perform('run')} disabled={busy}>{busy ? 'Выполняется…' : 'Запустить'}</button><button className="primary-button" onClick={() => void perform('submit')} disabled={busy}>{busy ? 'Проверяем…' : 'Проверить'}</button></div><div className="console"><div className="console-head"><strong>Результат</strong>{result?.status && <span className={result.passed ? 'result-status pass' : 'result-status'}>{result.passed ? 'Принято' : result.status === 'rejected' ? 'Не принято — код не допущен к проверке' : result.checks?.length ? 'Не принято' : result.status}</span>}</div>{result ? <><pre className="console-output" tabIndex={0}>{[result.stdout, result.stderr].filter(Boolean).join('\n') || (result.exit_code === 0 ? 'Программа завершилась без вывода.' : 'Нет вывода.')}</pre>{result.checks?.length && !result.passed ? <p className="check-verdict-note">Не принято — вот что разошлось.</p> : null}{result.checks?.map(check => <CheckLine key={check.name} check={check} />)}</> : <p className="console-empty">Здесь появятся вывод программы, ошибки компиляции и результаты проверки.</p>}</div><section className="solution-panel"><div><h3>Эталонное решение</h3><p>Сначала попробуй решить задачу самостоятельно. Решение не влияет на прогресс.</p></div><button className="quiet-button solution-toggle" onClick={() => solution === null ? void revealSolution() : setSolution(null)} disabled={solutionBusy} aria-expanded={solution !== null}>{solutionBusy ? 'Загружаем…' : solution === null ? 'Показать решение' : 'Скрыть решение'}</button>{solution !== null && <pre className="solution-source" tabIndex={0}><code>{solution}</code></pre>}</section></>}</section></div>
    </main> : <main className="page"><div className="empty-state">{busy ? 'Загружаем урок…' : 'Выбери урок из обзора или карты обучения.'}</div></main>)}
  </div>;
}

function Onboarding({ onSave, busy, error, theme, setTheme }: { onSave: (name: string, theme: 'light' | 'dark') => void; busy: boolean; error: string; theme: 'light' | 'dark'; setTheme: (v: 'light' | 'dark') => void }) {
  const [name, setName] = useState('');
  return <main className="onboarding"><section className="onboarding-card"><div className="onboard-mark"><span className="go-mark">go</span><span>Open Go Academy</span></div><p className="eyebrow">НАЧНИ С НУЛЯ</p><h1>Привет! Как к тебе обращаться?</h1><p className="onboard-copy">Создадим локальный учебный профиль. Регистрация и ИИ-наставник не нужны.</p><form onSubmit={e => { e.preventDefault(); if (name.trim()) onSave(name.trim(), theme); }}><label htmlFor="learner-name">Твоё имя</label><input id="learner-name" autoComplete="given-name" maxLength={60} value={name} onChange={e => setName(e.target.value)} placeholder="Например, Саша" required /><fieldset><legend>Тема оформления</legend><div className="theme-choices"><button type="button" className={theme === 'light' ? 'theme-choice chosen' : 'theme-choice'} onClick={() => setTheme('light')}>Светлая</button><button type="button" className={theme === 'dark' ? 'theme-choice chosen' : 'theme-choice'} onClick={() => setTheme('dark')}>Тёмная</button></div></fieldset>{error && <p className="form-error" role="alert">{error}</p>}<button className="primary-button onboarding-submit" disabled={busy || !name.trim()}>{busy ? 'Сохраняем…' : 'Создать профиль'}</button></form><small className="privacy-note">Профиль хранится в локальном учебном сервисе.</small></section><aside className="onboard-side"><div className="go-illustration"><span>go</span><i>fmt.Println("Привет!")</i><i>→ твоя первая программа</i></div><p>Никакого опыта не нужно.<br />Начнём с первой программы на Go.</p></aside></main>;
}
function starter(exercise: Exercise) { return exercise.starter_code ?? exercise.starter ?? ''; }
function message(error: unknown) { return error instanceof Error ? error.message : 'неизвестная ошибка'; }

function moduleTitle(id:string){return ({'zero-foundations':'Основы с нуля','junior-core':'Язык Go и инструменты','juniorplus-backend':'Backend: HTTP и PostgreSQL','juniorplus-web':'Backend: HTTP и PostgreSQL','juniorplus-http':'Сети и HTTP','juniorplus-sql':'PostgreSQL и SQL'} as Record<string,string>)[id] || 'Учебный модуль';}
function difficultyTitle(id?:string){return ({easy:'Простое',medium:'Среднее',hard:'Повышенная сложность'} as Record<string,string>)[id||'']||'Практика';}

function headingID(node:ReactNode):string {
 const text=(value:ReactNode):string=>typeof value==='string'||typeof value==='number'?String(value):Array.isArray(value)?value.map(text).join(''):isValidElement<{children?:ReactNode}>(value)?text(value.props.children):'';
 let output='',space=false;for(const ch of text(node).trim().toLowerCase()){if(ch===' '||ch==='\t'){space=output.length>0;continue}if(/[a-z0-9\u0400-\u04ff_-]/u.test(ch)){if(space){output+='-';space=false}output+=ch}}return output;
}
