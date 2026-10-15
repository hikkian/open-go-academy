package httpapi

import (
	"academy/internal/content"
	"academy/internal/store"
	"academy/internal/tutor"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type TutorExecutor interface {
	Complete(context.Context, tutor.Request) (tutor.Response, error)
}
type TutorSettings struct {
	Cloud                                      bool
	ProviderLabel, EconomyModel, AdvancedModel string
	DailyBudget, MaxOutputTokens               int
}

var tutorModes = []string{"explain", "simpler", "deeper", "example", "hint", "debug", "review", "quiz", "interview", "architecture", "study-plan"}

const tutorPrivacy = "Отправляются только введённый вопрос, открытый текст выбранного урока и код при отдельном согласии. Имя, прогресс, скрытые тесты, эталонные решения и ответы экзамена не отправляются. Автоматическое скрытие секретов не гарантирует их распознавание: проверь текст перед отправкой. Ответ модели не подтверждает выполнение кода, экзамен или ранг. Подписка Codex не оплачивает API провайдера."

func (a *API) tutorRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tutor", func(w http.ResponseWriter, r *http.Request) {
		usage, err := a.Store.AIUsage(time.Now().UTC().Format("2006-01-02"))
		if err != nil {
			fail(w, 500, "не удалось прочитать расход")
			return
		}
		cfg := a.TutorSettings
		label := cfg.ProviderLabel
		if label == "" {
			label = "Провайдер не настроен"
		}
		write(w, 200, map[string]any{"enabled": a.Tutor != nil, "cloud": cfg.Cloud, "provider_label": label, "models": map[string]string{"economy": cfg.EconomyModel, "advanced": cfg.AdvancedModel}, "daily_budget": cfg.DailyBudget, "max_output_tokens": cfg.MaxOutputTokens, "usage": usage, "modes": tutorModes, "privacy_notice": tutorPrivacy})
	})
	mux.HandleFunc("POST /api/tutor", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Mode      string `json:"mode"`
			Message   string `json:"message"`
			LessonID  string `json:"lesson_id"`
			Source    string `json:"source"`
			Consent   bool   `json:"consent"`
			ShareCode bool   `json:"share_code"`
		}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректный запрос Tutor")
			return
		}
		known := false
		for _, mode := range tutorModes {
			known = known || q.Mode == mode
		}
		if !known || !q.Consent || strings.TrimSpace(q.Message) == "" || !utf8.ValidString(q.Message) || utf8.RuneCountInString(q.Message) > 2000 || len(q.Source) > 8192 || !utf8.ValidString(q.Source) || (!q.ShareCode && q.Source != "") {
			fail(w, 422, "выбери режим, введи вопрос до 2000 символов и подтверди согласие; код до 8 КиБ требует отдельного разрешения")
			return
		}
		if a.Tutor == nil {
			fail(w, 503, "AI не настроен. Уроки, запуск Go, проекты и экзамены доступны без модели.")
			return
		}
		contextText := ""
		if q.LessonID != "" {
			lesson, ok := a.Content.FindLesson(q.LessonID)
			if !ok {
				fail(w, 404, "урок не найден")
				return
			}
			// Keep a bounded public overview even for long course chapters.
			// Reference solutions and private grading cases never enter context.
			contextText = tutorLessonContext(lesson)

		}
		if q.ShareCode && q.Source != "" {
			contextText += "\nКод ученика (не исполнялся Tutor):\n" + q.Source
		}
		select {
		case a.tutorGate <- struct{}{}:
			defer func() { <-a.tutorGate }()
		default:
			fail(w, 429, "предыдущий запрос Tutor ещё выполняется")
			return
		}
		// Reserve the maximum of four provider attempts, counting UTF-8 bytes as tokens.
		// Retries/timeouts have uncertain billing and consume this entire conservative reserve.
		reserve := 4 * (len(q.Message) + len(contextText) + 4096 + a.TutorSettings.MaxOutputTokens)
		day := time.Now().UTC().Format("2006-01-02")
		id, err := a.Store.ReserveAI(day, reserve, a.TutorSettings.DailyBudget)
		if errors.Is(err, store.ErrAIBudget) {
			fail(w, 429, "дневного бюджета недостаточно для безопасного резерва; новый день бюджета начинается в 00:00 UTC")
			return
		}
		if err != nil {
			fail(w, 500, "не удалось зарезервировать бюджет")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 65*time.Second)
		defer cancel()
		result, callErr := a.Tutor.Complete(ctx, tutor.Request{Mode: q.Mode, Message: tutor.Redact(q.Message), Context: tutor.Redact(contextText)})
		if err = a.Store.FinishAI(id, result.InputTokens, result.OutputTokens, result.Estimated || callErr != nil, callErr == nil); err != nil {
			fail(w, 500, "расход не подтверждён; резерв сохранён, повторная отправка может увеличить расход")
			return
		}
		if callErr != nil {
			fail(w, 502, "провайдер не ответил корректно; резерв списан консервативно, повторная отправка выполняется только вручную")
			return
		}
		usage, err := a.Store.AIUsage(day)
		if err != nil {
			fail(w, 500, "ответ получен, но расход не удалось прочитать")
			return
		}
		write(w, 200, map[string]any{"text": result.Text, "model": result.Model, "input_tokens": result.InputTokens, "output_tokens": result.OutputTokens, "estimated": result.Estimated, "cached": result.Cached, "usage": usage})
	})
}

func tutorLessonContext(lesson content.Lesson) string {
	text := fmt.Sprintf("Открытый учебный материал %s: %s\nЦели урока:\n%s\nСловарь:\n", lesson.ID, lesson.Title, strings.Join(lesson.LearningOutcomes, "\n"))
	for _, entry := range lesson.Glossary {
		text += fmt.Sprintf("%s — %s: %s\n", entry.Term, entry.Translation, entry.Definition)
	}
	const limit = 16384
	if len(text) > limit {
		text = text[:limit]
		for !utf8.ValidString(text) {
			text = text[:len(text)-1]
		}
	}
	return text
}
