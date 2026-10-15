package httpapi

import (
	"academy/internal/assessment"
	"academy/internal/credentialrender"
	"academy/internal/credentials"
	"academy/internal/learning"
	"academy/internal/projects"
	"academy/internal/store"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const credentialNotice = "Локальный учебный credential Open Go Academy. Самостоятельная установка — не доверенный внешний эмитент. Подпись подтверждает целостность относительно ключа этой установки; она не удостоверяет личность, должность, стаж, отсутствие списывания или аккредитацию. Формат oga-local-v1; соответствие Open Badges / W3C VC не подтверждено. QR ведёт на локальный адрес, доступный только пока эта установка работает."

func credentialError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrCredentialNotFound):
		fail(w, 404, "credential не найден или не опубликован")
	case errors.Is(err, store.ErrCredentialConflict):
		fail(w, 409, "для этой проверки credential уже выпущен")
	case errors.Is(err, store.ErrCredentialPrerequisites):
		fail(w, 409, "сначала подтвердите критерии экзамена")
	default:
		fail(w, 500, "не удалось обработать credential")
	}
}
func (a *API) credentialEnvelope(c store.Credential) (credentials.Envelope, error) {
	var e credentials.Envelope
	d := json.NewDecoder(strings.NewReader(c.EnvelopeJSON))
	d.DisallowUnknownFields()
	if err := d.Decode(&e); err != nil {
		return e, err
	}
	if len(a.IssuerKey) != ed25519.PrivateKeySize {
		return e, credentials.ErrInvalid
	}
	if err := credentials.Verify(a.IssuerKey.Public().(ed25519.PublicKey), e); err != nil {
		return e, err
	}
	p := e.Payload
	if p.ID != c.ID || p.AttemptID != c.AttemptID || p.Rank != c.Rank || p.ExamID != c.ExamID || p.CriteriaVersion != c.Version || p.CriteriaHash != c.CriteriaHash || e.KeyID != c.KeyID {
		return e, credentials.ErrInvalid
	}
	return e, nil
}
func (a *API) credentialSummary(c store.Credential) map[string]any {
	e, err := a.credentialEnvelope(c)
	return map[string]any{"id": c.ID, "attempt_id": c.AttemptID, "rank": c.Rank, "subject_name": e.Payload.SubjectName, "created_at": c.CreatedAt, "published": c.Published, "revoked_at": c.RevokedAt, "signature_valid": err == nil, "verify_url": e.Payload.VerifyURL, "criteria_version": c.Version}
}
func (a *API) verifyCredential(w http.ResponseWriter, c store.Credential) {
	e, err := a.credentialEnvelope(c)
	status := "active"
	if c.RevokedAt != nil {
		status = "revoked"
	}
	if err != nil {
		status = "invalid"
	}
	publicKey := ""
	if len(a.IssuerKey) == ed25519.PrivateKeySize {
		publicKey = base64.RawURLEncoding.EncodeToString(a.IssuerKey.Public().(ed25519.PublicKey))
	}
	write(w, 200, map[string]any{"signature_valid": err == nil, "revoked": c.RevokedAt != nil, "status": status, "issuer_trust": "local-self-issued", "notice": credentialNotice, "payload": e.Payload, "key_id": c.KeyID, "public_key": publicKey})
}
func (a *API) credentialRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/credentials", func(w http.ResponseWriter, r *http.Request) {
		records, err := a.Store.Credentials("local")
		if err != nil {
			credentialError(w, err)
			return
		}
		items := []any{}
		for _, c := range records {
			items = append(items, a.credentialSummary(c))
		}
		write(w, 200, items)
	})
	mux.HandleFunc("POST /api/credentials", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			AttemptID string `json:"attempt_id"`
			Name      string `json:"name"`
			Consent   bool   `json:"consent"`
		}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректный запрос выпуска")
			return
		}
		q.Name = strings.TrimSpace(q.Name)
		if !q.Consent || utf8.RuneCountInString(q.Name) < 1 || utf8.RuneCountInString(q.Name) > 64 || !utf8.ValidString(q.Name) {
			fail(w, 422, "укажи имя или псевдоним до 64 символов и согласие на сохранение в документе")
			return
		}
		if len(a.IssuerKey) != ed25519.PrivateKeySize {
			fail(w, 503, "ключ локального эмитента недоступен")
			return
		}
		attempt, err := a.Store.ExamAttempt("local", q.AttemptID)
		if err != nil {
			fail(w, 404, "попытка не найдена")
			return
		}
		exam, knownExam := assessment.Find(attempt.ExamID, attempt.Version)
		if !knownExam || attempt.ExamID != exam.ID || attempt.Version != exam.Version || attempt.CriteriaHash != exam.Hash() || attempt.Status != "completed" || !attempt.Passed || attempt.Score != 100 {
			credentialError(w, store.ErrCredentialPrerequisites)
			return
		}
		b := make([]byte, 16)
		if _, err = rand.Read(b); err != nil {
			credentialError(w, err)
			return
		}
		id := hex.EncodeToString(b)
		e, err := credentials.Sign(a.IssuerKey, credentials.Payload{ID: id, IssuerName: "Open Go Academy — локальная установка", SubjectName: q.Name, Rank: attempt.Rank, ExamID: attempt.ExamID, CriteriaVersion: attempt.Version, CriteriaHash: attempt.CriteriaHash, AttemptID: attempt.ID, IssuedAt: time.Now().UTC().Format(time.RFC3339Nano), VerifyURL: a.VerifyOrigin + "/verify/" + id, Competencies: exam.Criteria})
		if err != nil {
			fail(w, 422, "некорректные поля локального документа")
			return
		}
		raw, _ := json.Marshal(e)
		c, err := a.Store.IssueCredential("local", store.Credential{ID: id, ProfileID: "local", AttemptID: attempt.ID, ExamID: attempt.ExamID, Version: attempt.Version, Rank: attempt.Rank, CriteriaHash: attempt.CriteriaHash, KeyID: e.KeyID, EnvelopeJSON: string(raw)})
		if err != nil {
			credentialError(w, err)
			return
		}
		write(w, 201, a.credentialSummary(c))
	})
	mux.HandleFunc("PUT /api/credentials/{id}/privacy", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Published bool `json:"published"`
			Consent   bool `json:"consent"`
		}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректная настройка публикации")
			return
		}
		if q.Published && !q.Consent {
			fail(w, 422, "публикация имени, ранга, даты и критериев требует согласия")
			return
		}
		if err := a.Store.SetCredentialPublished("local", r.PathValue("id"), q.Published); err != nil {
			credentialError(w, err)
			return
		}
		write(w, 200, map[string]bool{"published": q.Published})
	})
	mux.HandleFunc("POST /api/credentials/{id}/revoke", func(w http.ResponseWriter, r *http.Request) {
		var q struct{}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректный запрос отзыва")
			return
		}
		if err := a.Store.RevokeCredential("local", r.PathValue("id"), "owner-request"); err != nil {
			credentialError(w, err)
			return
		}
		write(w, 200, map[string]bool{"revoked": true})
	})
	mux.HandleFunc("GET /api/credentials/{id}/verify", func(w http.ResponseWriter, r *http.Request) {
		c, err := a.Store.Credential("local", r.PathValue("id"))
		if err != nil {
			credentialError(w, err)
			return
		}
		a.verifyCredential(w, c)
	})
	mux.HandleFunc("GET /api/verify/{id}", func(w http.ResponseWriter, r *http.Request) {
		c, err := a.Store.PublicCredential(r.PathValue("id"))
		if err != nil {
			credentialError(w, err)
			return
		}
		a.verifyCredential(w, c)
	})
	mux.HandleFunc("POST /api/credentials/check", func(w http.ResponseWriter, r *http.Request) {
		var e credentials.Envelope
		if decode(w, r, &e) != nil {
			fail(w, 400, "неподдерживаемый JSON-конверт")
			return
		}
		if len(a.IssuerKey) != ed25519.PrivateKeySize || credentials.Verify(a.IssuerKey.Public().(ed25519.PublicKey), e) != nil {
			write(w, 200, map[string]any{"signature_valid": false, "status": "invalid", "notice": credentialNotice})
			return
		}
		c, err := a.Store.Credential("local", e.Payload.ID)
		if err != nil {
			write(w, 200, map[string]any{"signature_valid": true, "status": "unavailable", "notice": credentialNotice})
			return
		}
		stored, err := a.credentialEnvelope(c)
		left, _ := json.Marshal(stored)
		right, _ := json.Marshal(e)
		if err != nil || string(left) != string(right) {
			write(w, 200, map[string]any{"signature_valid": false, "status": "invalid", "notice": credentialNotice})
			return
		}
		a.verifyCredential(w, c)
	})
	for _, format := range []string{"json", "pdf", "png", "qr"} {
		mux.HandleFunc("GET /api/credentials/{id}/"+format, func(w http.ResponseWriter, r *http.Request) {
			c, err := a.Store.Credential("local", r.PathValue("id"))
			if err != nil {
				credentialError(w, err)
				return
			}
			e, err := a.credentialEnvelope(c)
			if err != nil {
				fail(w, 409, "подпись документа не прошла проверку; экспорт запрещён")
				return
			}
			var data []byte
			mime := "application/json"
			ext := format
			switch format {
			case "json":
				data = []byte(c.EnvelopeJSON)
			case "pdf":
				mime = "application/pdf"
				data, err = credentialrender.PDF(e, c.RevokedAt != nil)
			case "png":
				mime = "image/png"
				data, err = credentialrender.PNG(e, c.RevokedAt != nil)
			case "qr":
				mime = "image/png"
				ext = "png"
				data, err = credentialrender.QRPNG(e)
			}
			if err != nil {
				fail(w, 500, "не удалось сформировать документ")
				return
			}
			w.Header().Set("Content-Type", mime)
			w.Header().Set("Content-Disposition", `attachment; filename="academy-`+c.ID+`.`+ext+`"`)
			w.Write(data)
		})
	}
	mux.HandleFunc("PUT /api/passport/privacy", func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Published bool `json:"published"`
			Consent   bool `json:"consent"`
		}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректные настройки паспорта")
			return
		}
		if q.Published && !q.Consent {
			fail(w, 422, "публикация имени, навыков и ранга требует согласия")
			return
		}
		if err := a.Store.SetPassportPublic("local", q.Published); err != nil {
			credentialError(w, err)
			return
		}
		write(w, 200, map[string]bool{"public": q.Published})
	})
	mux.HandleFunc("GET /api/passport", func(w http.ResponseWriter, r *http.Request) { a.passport(w, false) })
	mux.HandleFunc("GET /api/passport/public", func(w http.ResponseWriter, r *http.Request) { a.passport(w, true) })
}
func (a *API) passport(w http.ResponseWriter, public bool) {
	visible, err := a.Store.PassportPublic("local")
	if err != nil {
		credentialError(w, err)
		return
	}
	if public && !visible {
		fail(w, 404, "паспорт не опубликован")
		return
	}
	p, err := a.Store.Profile()
	if err != nil {
		fail(w, 404, "профиль не найден")
		return
	}
	progress, err := a.Store.Progress()
	if err != nil {
		credentialError(w, err)
		return
	}
	awards, err := a.Store.RankAwards("local")
	if err != nil {
		credentialError(w, err)
		return
	}
	snapshot := learning.WithAwards(learning.WithCatalog(learning.Build(progress), progress, a.Content), awards)
	verifiedTimes := map[string]time.Time{}
	activityByDate := map[string]int{}
	for exerciseID, attemptID := range progress.Evidence {
		var storedExercise, createdAt string
		var passed int
		err := a.Store.DB().QueryRow(`SELECT exercise_id,passed,created_at FROM attempts WHERE id=?`, attemptID).Scan(&storedExercise, &passed, &createdAt)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			credentialError(w, err)
			return
		}
		if storedExercise != exerciseID || passed != 1 {
			continue
		}
		created, parseErr := time.Parse(time.RFC3339Nano, createdAt)
		if parseErr != nil {
			continue
		}
		verifiedTimes[exerciseID] = created
		activityByDate[created.UTC().Format("2006-01-02")]++
	}
	activity := make([]map[string]any, 0, len(activityByDate))
	activityDates := make([]string, 0, len(activityByDate))
	for day := range activityByDate {
		activityDates = append(activityDates, day)
	}
	sort.Strings(activityDates)
	cumulative := 0
	for _, day := range activityDates {
		count := activityByDate[day]
		cumulative += count
		activity = append(activity, map[string]any{"date": day, "verified_count": count, "cumulative_count": cumulative})
	}
	records, err := a.Store.Credentials("local")
	if err != nil {
		credentialError(w, err)
		return
	}
	items := []any{}
	for _, c := range records {
		if !public || c.Published {
			items = append(items, a.credentialSummary(c))
		}
	}
	earned := []any{}
	for _, id := range progress.Completed {
		title, description := "", ""
		switch id {
		case "hello":
			title = "Первая программа"
			description = "Проверено упражнение с приветствием Go."
		case "repair":
			title = "Охотник за ошибками"
			description = "Исправлена и проверена первая программа."
		case "p2:automatic":
			title = "API Builder"
			description = "Пройдены изолированные автоматические проверки P2; архитектурное ревью отдельно."
		case "p1:automatic":
			title = "CLI Builder"
			description = "Пройдены автоматические проверки P1; ручное ревью отдельно."
		}
		if title != "" {
			if earnedAt, ok := verifiedTimes[id]; ok {
				earned = append(earned, map[string]any{"id": id, "title": title, "description": description, "earned_at": earnedAt})
			}
		}
	}
	projectOverview := make([]map[string]any, 0, len(projects.All()))
	for _, catalogProject := range projects.All() {
		automaticPassed := false
		for _, completedID := range progress.Completed {
			if completedID == catalogProject.ID+":automatic" {
				automaticPassed = true
				break
			}
		}
		projectOverview = append(projectOverview, map[string]any{
			"id":                     catalogProject.ID,
			"title":                  catalogProject.Title,
			"automatic_passed":       automaticPassed,
			"manual_review_required": true,
		})
	}
	issuer := map[string]any{"name": "Open Go Academy — локальная установка", "trust_notice": credentialNotice, "key_id": "", "public_key": ""}
	if len(a.IssuerKey) == ed25519.PrivateKeySize {
		pub := a.IssuerKey.Public().(ed25519.PublicKey)
		issuer["key_id"] = credentials.KeyID(pub)
		issuer["public_key"] = base64.RawURLEncoding.EncodeToString(pub)
	}
	if public {
		write(w, 200, map[string]any{"profile": map[string]string{"name": p.Name}, "learning": map[string]any{"current_rank": snapshot.CurrentRank, "verified": snapshot.Verified, "skills": snapshot.Skills}, "credentials": items, "issuer": issuer})
		return
	}
	write(w, 200, map[string]any{"profile": map[string]string{"name": p.Name}, "learning": snapshot, "achievements": earned, "projects": projectOverview, "activity": activity, "credentials": items, "awards": awards, "public": visible, "issuer": issuer})
}
