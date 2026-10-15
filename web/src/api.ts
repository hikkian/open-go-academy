export type Profile = { id?: string; name: string; theme: 'light' | 'dark'; rank?: string; xp?: number };
export type Exercise = {
  id: string; kind?: 'function'; skills?: string[]; title: string; difficulty?: string; instructions?: string; prompt?: string;
  starter_code?: string; starter?: string; starter_file?: string; public_output?: string; expected_output?: string;
  self_check?: { question: string; answer: string }[];
};
export type Lesson = {
  id: string; title: string; rank: string; module: string; order?: number; curriculum_version?: string; prerequisites?: string[]; estimated_minutes: number; body?: string;
  learning_outcomes?: string[]; glossary?: { term: string; translation: string; definition: string }[]; exercises?: Exercise[];
  self_check?: { question: string; answer: string }[];
};
export type Progress = { completed: string[]; attempts: unknown[]; xp: number };
export type LearningSnapshot = {
  policy_version: string; current_rank: string; verified: boolean;
  ranks: { name: string; criteria: string[]; skills: string[]; content_available: boolean; assessment_available: boolean }[];
  skills: { id: string; title: string; rank: string; level: 0 | 1 | 2 | 3 | 4; evidence: string[] }[];
  review_exercises: string[];
};

export type ExamSummary = { id: string; version: string; title: string; rank: string; estimated_minutes: number; description: string; eligible: boolean; missing_prerequisites: string[]; criteria: string[] };
export type ExamQuestion = { id: string; prompt: string; options: { id: string; text: string }[] };
export type ExamTask = { id: string; title: string; kind: 'coding' | 'debugging'; description: string; starter_code: string; public_examples: { input: string; output: string }[] };
export type PublicExam = { id: string; version: string; title: string; rank: string; description: string; estimated_minutes: number; prerequisites: string[]; criteria: string[]; questions: ExamQuestion[]; tasks: ExamTask[] };
export type ExamAttempt = { id: string; exam_id: string; version: string; revision: number; status: string; score?: number; passed?: boolean; checks_json?: string; answers_json?: string; codes_json?: string; created_at?: string; finished_at?: string; [key: string]: unknown };
export type ExamCheck = { id: string; name: string; passed: boolean; detail?: string };
export type ExamSubmitResult = { attempt: ExamAttempt; score: number; passed: boolean; checks: ExamCheck[]; rank_awarded: boolean; rank: string; trust_notice: string };

let csrfToken = '';
export class ApiError extends Error {
  readonly status: number;
  constructor(message: string, status: number) { super(message); this.name = 'ApiError'; this.status = status; }
}
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body) headers.set('Content-Type', 'application/json');
  if (init.method && init.method !== 'GET' && csrfToken) headers.set('X-CSRF-Token', csrfToken);
  const response = await fetch(`/api${path}`, { ...init, headers, credentials: 'same-origin' });
  if (!response.ok) {
    const detail = await response.text().catch(() => '');
    let readableDetail = detail;
    try {
      const parsed: unknown = JSON.parse(detail);
      if (parsed && typeof parsed === 'object' && 'error' in parsed && typeof parsed.error === 'string') readableDetail = parsed.error;
    } catch { /* Plain-text error bodies are already readable. */ }
    throw new ApiError(readableDetail || `Сервер ответил ${response.status}`, response.status);
  }
  return response.status === 204 ? undefined as T : response.json() as Promise<T>;
}
export async function loadProfile(): Promise<Profile | null> {
  const result = await api<{ profile: Profile | null; csrf_token: string }>('/profile');
  csrfToken = result.csrf_token;
  return result.profile;
}
export async function saveProfile(payload: Pick<Profile, 'name' | 'theme'>, method: 'POST' | 'PUT') {
  const response = await api<Profile | { profile: Profile }>('/profile', { method, body: JSON.stringify(payload) });
  return 'profile' in response ? response : { profile: response };
}
