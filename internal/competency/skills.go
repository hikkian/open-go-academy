// Package competency defines the canonical identifiers shared by course metadata
// and the learner's skill matrix. A practice tag does not grant a rank.
package competency

import "fmt"

type Definition struct {
	ID    string
	Title string
	Rank  string
}

func All() []Definition {
	return []Definition{
		{"program", "Исходный код и запуск программы", "Zero"},
		{"output", "Вывод и порядок команд", "Zero"},
		{"functions", "Функции и тип возвращаемого значения", "Zero"},
		{"go-core", "Коллекции, структуры и интерфейсы", "Junior"},
		{"testing", "Автоматические тесты и диагностика", "Junior"},
		{"http", "HTTP: запросы, ответы и API", "Junior+"},
		{"sql", "SQL: хранение и запросы данных", "Junior+"},
		{"concurrency", "Конкурентное выполнение и гонки", "Middle"},
		{"reliability", "Надёжность и эксплуатация", "Middle"},
		{"performance", "Измерение производительности", "Middle+"},
		{"distributed", "Распределённые системы", "Middle+"},
		{"architecture", "Архитектура и компромиссы", "Senior"},
		{"communication", "Ревью и инженерное объяснение", "Senior"},
	}
}

func Validate(ids []string) error {
	known := make(map[string]bool)
	for _, definition := range All() {
		known[definition.ID] = true
	}
	seen := make(map[string]bool)
	for _, id := range ids {
		if !known[id] {
			return fmt.Errorf("unknown competency %q", id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate competency %q", id)
		}
		seen[id] = true
	}
	return nil
}
