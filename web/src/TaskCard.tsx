import { isValidElement, useState, type ReactNode } from 'react';
import ReactMarkdown from 'react-markdown';
import type { Exercise, LessonCheck } from './api';

// The task card, the callouts of a lesson, the chips of its tasks and the "expected / actual" block of a failed check
// (docs/LESSON_UI_SPEC.md §1.1–1.5). Stage 1: the card is read from the lesson's Markdown section "### Задание N · title".

// Plain text of a rendered node, as headingID reads it.
export function textOf(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') return String(node);
  if (Array.isArray(node)) return node.map(textOf).join('');
  if (isValidElement<{ children?: ReactNode }>(node)) return textOf(node.props.children);
  return '';
}

const childList = (node: ReactNode): ReactNode[] => (Array.isArray(node) ? node : [node]);

// The Markdown section of one task: from its heading to the next heading of the same or a higher level.
export function taskSection(body: string | undefined, title: string): string {
  if (!body) return '';
  const lines = body.split('\n');
  const start = lines.findIndex(line => {
    const match = /^###\s+Задание\s+\d+\s+·\s+(.*)$/.exec(line.trim());
    return match !== null && match[1].trim() === title.trim();
  });
  if (start < 0) return '';
  const rest = lines.slice(start + 1);
  const end = rest.findIndex(line => /^#{1,3}\s/.test(line));
  return (end < 0 ? rest : rest.slice(0, end)).join('\n');
}

// The text after a bold label, up to the next blank line; the quote markers of a callout are dropped.
function afterLabel(section: string, label: string): string {
  const at = section.indexOf(label);
  if (at < 0) return '';
  return section.slice(at + label.length).split('\n\n')[0].replace(/\n>\s?/g, ' ').replace(/^>\s?/, '').trim();
}

// The steps: from "**Шаги.**" to the next labelled block.
function stepsOf(section: string): string {
  const at = section.indexOf('**Шаги.**');
  if (at < 0) return '';
  const rest = section.slice(at + '**Шаги.**'.length);
  const ends = ['**Как проверить себя.**', '**Если не принято.**', '> **'].map(mark => rest.indexOf(mark)).filter(i => i >= 0);
  return (ends.length ? rest.slice(0, Math.min(...ends)) : rest).trim();
}

// Callouts (§1.4): the label is the first <strong> of the first paragraph, read from the rendered tree, not from HTML.
const calloutKinds: Record<string, 'important' | 'hint' | 'mistake' | 'why'> = {
  'Важно.': 'important', 'Подсказка.': 'hint', 'Типичная ошибка.': 'mistake', 'Зачем.': 'why',
};

function calloutKind(children: ReactNode): (typeof calloutKinds)[string] | null {
  const paragraph = childList(children).find(child => isValidElement<{ children?: ReactNode }>(child) && child.type === 'p');
  if (!isValidElement<{ children?: ReactNode }>(paragraph)) return null;
  const strong = childList(paragraph.props.children).find(child => isValidElement<{ children?: ReactNode }>(child) && child.type === 'strong');
  if (!isValidElement<{ children?: ReactNode }>(strong)) return null;
  return calloutKinds[textOf(strong.props.children)] ?? null;
}

export function Blockquote({ children }: { children?: ReactNode }) {
  const kind = calloutKind(children);
  if (!kind) return <blockquote>{children}</blockquote>;
  if (kind === 'hint') return <details className="callout callout-hint"><summary>Подсказка</summary>{children}</details>;
  return <aside className={`callout callout-${kind}`}>{children}</aside>;
}

// Visible line ends for the expected output (§1.1): every line break is shown as ⏎.
const withLineBreaks = (text: string) => text.replace(/\n/g, '⏎\n');

const narrow = () => typeof window !== 'undefined' && window.matchMedia?.('(max-width: 980px)').matches;

function readOpen(lessonId: string): boolean {
  try {
    const stored = localStorage.getItem(`task-card-open:${lessonId}`);
    return stored === null ? !narrow() : stored === '1';
  } catch {
    return !narrow();
  }
}

function writeOpen(lessonId: string, open: boolean) {
  try {
    localStorage.setItem(`task-card-open:${lessonId}`, open ? '1' : '0');
  } catch {
    // a private window or blocked storage: the card still works, it just does not remember
  }
}

export function TaskCard({ index, exercise, body, checks, lessonId }: { index: number; exercise: Exercise; body?: string; checks: LessonCheck[]; lessonId: string }) {
  const [open, setOpen] = useState(() => readOpen(lessonId));
  const section = taskSection(body, exercise.title);
  const goal = afterLabel(section, '**Цель.**');
  const hint = afterLabel(section, '**Подсказка.**');
  const steps = stepsOf(section);
  const criteria = exercise.criteria ?? [];
  const toggle = () => {
    setOpen(current => {
      writeOpen(lessonId, !current);
      return !current;
    });
  };
  return (
    <section className="task-card" id={`task-card-${exercise.id}`} aria-label={`Карточка задания ${index}`}>
      <header className="task-card-head">
        <p className="eyebrow">Задание {String(index).padStart(2, '0')}</p>
        <button type="button" className="quiet-button task-card-toggle" aria-expanded={open} onClick={toggle}>{open ? 'Свернуть' : 'Развернуть'}</button>
      </header>
      <h2 className="task-card-title">{exercise.title}</h2>
      {goal ? <div className="task-goal"><ReactMarkdown>{goal}</ReactMarkdown></div> : exercise.prompt ? <p className="task-goal">{exercise.prompt}</p> : null}
      {open && (
        <>
          {exercise.public_output ? (
            <section className="task-expected" aria-label="Ожидаемый вывод">
              <h3>Ожидаемый вывод</h3>
              <pre className="task-expected-output" tabIndex={0}>{withLineBreaks(exercise.public_output)}</pre>
            </section>
          ) : null}
          {criteria.length > 0 && (
            <section className="task-criteria-block" aria-label="Критерии">
              <h3>Критерии</h3>
              <ol className="task-criteria">
                {criteria.map((text, i) => {
                  const check = checks.find(item => item.criterion === i);
                  const state = !check ? 'idle' : check.passed ? 'ok' : 'fail';
                  return <li key={text} className={`criterion criterion-${state}`}><span aria-hidden="true">{state === 'ok' ? '✓' : state === 'fail' ? '×' : '○'}</span>{text}</li>;
                })}
              </ol>
            </section>
          )}
          {steps && <details className="task-steps"><summary>Шаги</summary><ReactMarkdown>{steps}</ReactMarkdown></details>}
          {hint && <details className="callout callout-hint"><summary>Подсказка</summary><ReactMarkdown>{hint}</ReactMarkdown></details>}
        </>
      )}
    </section>
  );
}

// The chips of the lesson's tasks (§1.5): the current one is marked, a finished one carries a check mark.
export function TaskChips({ exercises, activeId, completed, onSelect }: { exercises: Exercise[]; activeId?: string; completed: Set<string>; onSelect: (exercise: Exercise) => void }) {
  if (exercises.length === 0) return null;
  return (
    <nav className="task-chips" aria-label="Задания урока">
      <span className="task-chips-label">Задания:</span>
      {exercises.map((exercise, i) => (
        <button key={exercise.id} type="button" className={`task-chip${activeId === exercise.id ? ' current' : ''}${completed.has(exercise.id) ? ' done' : ''}`}
          aria-current={activeId === exercise.id ? 'true' : undefined} onClick={() => onSelect(exercise)}>
          {String(i + 1).padStart(2, '0')}{completed.has(exercise.id) ? ' ✓' : ''}
        </button>
      ))}
    </nav>
  );
}

type Row = { text: string; newline: boolean };

// The grader drops one final newline before it compares lines (internal/grading/grading.go), so the learner's view does the
// same: a final newline is not a difference, and every remaining row is a line.
const dropFinalNewline = (text: string) => (text.endsWith('\n') ? text.slice(0, -1) : text);

function rowsOf(text: string): Row[] {
  if (text === '') return [];
  return dropFinalNewline(text).split('\n').map(line => ({ text: line, newline: true }));
}

// Spaces are shown as · so that a double space or a trailing one is visible (§1.3).
const visible = (text: string) => text.replace(/ /g, '·').replace(/\t/g, '→');

function firstDifference(expected: Row[], actual: Row[]): number {
  const rows = Math.max(expected.length, actual.length);
  for (let i = 0; i < rows; i++) {
    const e = expected[i], a = actual[i];
    if (!e || !a || e.text !== a.text || e.newline !== a.newline) return i;
  }
  return -1;
}

// «Ждали / Получили» for a failed check with its expected and actual text (§1.3).
export function CheckLine({ check }: { check: LessonCheck }) {
  if (check.passed) return <p className="check-result passed"><span>✓</span>{check.name}</p>;
  if (check.expected === undefined) {
    return <>
      <p className="check-result failed"><span>×</span>{check.name}</p>
      {check.hint && <p className="check-hint">{check.hint}</p>}
    </>;
  }
  const expected = rowsOf(check.expected);
  const actual = rowsOf(check.actual ?? '');
  const first = firstDifference(expected, actual);
  const rows = Math.max(expected.length, actual.length);
  return (
    <div className="check-compare">
      <p className="check-result failed"><span>×</span>{check.name}</p>
      <div className="compare-grid">
        <div className="compare-col">
          <h4>Ждали</h4>
          {expected.length === 0 ? <p className="compare-empty">ничего</p> : (
            <ol className="compare-lines">
              {expected.map((row, i) => <li key={i} className={i === first ? 'mismatch' : undefined}>{visible(row.text)}{row.newline ? <span className="nl">⏎</span> : <span className="no-nl">без ⏎</span>}</li>)}
            </ol>
          )}
        </div>
        <div className="compare-col">
          <h4>Получили</h4>
          {actual.length === 0 ? <p className="compare-empty">ничего</p> : (
            <ol className="compare-lines">
              {Array.from({ length: rows }, (_, i) => actual[i] ? (
                <li key={i} className={i === first ? 'mismatch' : i >= expected.length ? 'extra' : undefined}>
                  {i >= expected.length && <span className="sign">+</span>}{visible(actual[i].text)}{actual[i].newline ? <span className="nl">⏎</span> : <span className="no-nl">без ⏎</span>}
                </li>
              ) : (
                <li key={i} className="missing"><span className="sign">−</span>нет строки</li>
              ))}
            </ol>
          )}
        </div>
      </div>
      {check.hint && <p className="check-hint">{check.hint}</p>}
    </div>
  );
}
