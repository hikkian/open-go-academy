export type Profile = { id?: string; name: string; theme: 'light' | 'dark'; rank?: string; xp?: number };
export type Exercise = {
  id: string; title: string; difficulty?: string; instructions?: string; prompt?: string;
  starter_code?: string; starter?: string; starter_file?: string; public_output?: string; expected_output?: string;
  self_check?: { question: string; answer: string }[];
};
export type Lesson = {
  id: string; title: string; rank: string; module: string; estimated_minutes: number; body?: string;
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

let csrfToken = '';
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
    throw new Error(readableDetail || `Сервер ответил ${response.status}`);
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
