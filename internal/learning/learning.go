// Package learning derives educational signals from server-verified exercise evidence.
package learning

import "academy/internal/store"

const PolicyVersion = "rank-policy-0.1"

type Rank struct {
	Name                string   `json:"name"`
	Criteria            []string `json:"criteria"`
	Skills              []string `json:"skills"`
	ContentAvailable    bool     `json:"content_available"`
	AssessmentAvailable bool     `json:"assessment_available"`
}
type Skill struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Rank       string   `json:"rank"`
	Level      int      `json:"level"`
	Evidence   []string `json:"evidence"`
	AttemptIDs []int64  `json:"attempt_ids"`
}
type Snapshot struct {
	PolicyVersion   string   `json:"policy_version"`
	CurrentRank     string   `json:"current_rank"`
	Verified        bool     `json:"verified"`
	Ranks           []Rank   `json:"ranks"`
	Skills          []Skill  `json:"skills"`
	ReviewExercises []string `json:"review_exercises"`
}

func Build(p store.Progress) Snapshot {
	s := Snapshot{PolicyVersion: PolicyVersion, CurrentRank: "Zero", Verified: false, ReviewExercises: []string{}, Ranks: []Rank{
		{Name: "Zero", ContentAvailable: true, Criteria: []string{"Самостоятельно запустить и исправить простые Go-программы.", "Освоить типы, условия, циклы и функции.", "Пройти итоговую проверку Zero (ещё не реализована)."}, Skills: []string{"program", "output", "functions"}},
		{Name: "Junior", Criteria: []string{"Go core: коллекции, структуры, указатели, интерфейсы, errors (ошибки), модули.", "CLI Toolkit P1 с тестами и Git-ready экспортом.", "Пройти экзамен Go core и защитить решения."}, Skills: []string{"go-core", "testing"}},
		{Name: "Junior+", Criteria: []string{"HTTP/SQL/auth: понимать протоколы, запросы и границы доступа.", "LinkHub P2 с PostgreSQL, миграциями и integration tests (интеграционными тестами).", "Сдать проверку API и безопасности."}, Skills: []string{"http", "sql"}},
		{Name: "Middle", Criteria: []string{"Concurrency (конкурентность), context, race (гонки), transactions (транзакции).", "TaskFlow P3: надёжность, очереди, мониторинг, CI и нагрузочные проверки.", "Защитить инженерные решения и пройти mixed assessment (смешанную проверку)."}, Skills: []string{"concurrency", "reliability"}},
		{Name: "Middle+", Criteria: []string{"ScaleLab P4: baseline (исходные измерения), profiling (профилирование), before/after.", "Обосновать SLO (цели уровня сервиса), capacity (ёмкость), consistency (согласованность).", "Разобрать сбои и защитить компромиссы."}, Skills: []string{"performance", "distributed"}},
		{Name: "Senior", Criteria: []string{"Architecture Capstone P5: design doc, ADR (запись архитектурного решения), критические прототипы.", "Обосновать security (безопасность), HA (высокую доступность) и эволюцию системы.", "Пройти review (рецензию) с участием человека; учебная проверка не заменяет опыт работы."}, Skills: []string{"architecture", "communication"}},
	}, Skills: []Skill{
		{ID: "program", Title: "Исходный код и запуск программы", Rank: "Zero", Evidence: []string{}},
		{ID: "output", Title: "Вывод и порядок команд", Rank: "Zero", Evidence: []string{}},
		{ID: "functions", Title: "Функции и тип возвращаемого значения", Rank: "Zero", Evidence: []string{}},
		{ID: "go-core", Title: "Коллекции, структуры и интерфейсы", Rank: "Junior", Evidence: []string{}},
		{ID: "testing", Title: "Автоматические тесты и диагностика", Rank: "Junior", Evidence: []string{}},
		{ID: "http", Title: "HTTP: запросы, ответы и API", Rank: "Junior+", Evidence: []string{}},
		{ID: "sql", Title: "SQL: хранение и запросы данных", Rank: "Junior+", Evidence: []string{}},
		{ID: "concurrency", Title: "Конкурентное выполнение и гонки", Rank: "Middle", Evidence: []string{}},
		{ID: "reliability", Title: "Надёжность и эксплуатация", Rank: "Middle", Evidence: []string{}},
		{ID: "performance", Title: "Измерение производительности", Rank: "Middle+", Evidence: []string{}},
		{ID: "distributed", Title: "Распределённые системы", Rank: "Middle+", Evidence: []string{}},
		{ID: "architecture", Title: "Архитектура и компромиссы", Rank: "Senior", Evidence: []string{}},
		{ID: "communication", Title: "Ревью и инженерное объяснение", Rank: "Senior", Evidence: []string{}},
	}}
	done := map[string]bool{}
	for _, id := range p.Completed {
		if p.Evidence[id] > 0 {
			done[id] = true
		}
	}
	evidence := map[string][]string{"program": {"hello"}, "output": {"two-lines"}, "functions": {"repair"}}
	for i := range s.Skills {
		for _, id := range evidence[s.Skills[i].ID] {
			if done[id] {
				s.Skills[i].Level = 2
				s.Skills[i].Evidence = append(s.Skills[i].Evidence, id)
				s.Skills[i].AttemptIDs = append(s.Skills[i].AttemptIDs, p.Evidence[id])
			}
		}
	}
	last := map[string]store.Attempt{}
	for _, a := range p.Attempts {
		if previous, ok := last[a.ExerciseID]; !ok || a.ID > previous.ID {
			last[a.ExerciseID] = a
		}
	}
	for _, id := range []string{"hello", "repair", "two-lines"} {
		if a, ok := last[id]; ok && !a.Passed {
			s.ReviewExercises = append(s.ReviewExercises, id)
		}
	}
	return s
}
