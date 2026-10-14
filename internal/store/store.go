// Package store provides the durable single-profile SQLite store.
package store

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var ErrProfileNotFound = errors.New("local profile not found")
var ErrProfileExists = errors.New("local profile already exists")
var ErrProjectSnapshotNotFound = errors.New("project snapshot not found")

type Store struct{ db *sql.DB }
type Profile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Theme string `json:"theme"`
	Rank  string `json:"rank"`
	XP    int    `json:"xp"`
}
type Attempt struct {
	ID         int64             `json:"id"`
	ExerciseID string            `json:"exercise_id"`
	Files      map[string]string `json:"files"`
	Passed     bool              `json:"passed"`
	Output     string            `json:"output"`
	CreatedAt  time.Time         `json:"created_at"`
}
type Progress struct {
	Completed []string         `json:"completed"`
	Attempts  []Attempt        `json:"attempts"`
	Evidence  map[string]int64 `json:"evidence"`
	XP        int              `json:"xp"`
}

//go:embed migrations/*.sql
var migrations embed.FS

// Open opens a SQLite database and applies all numbered migrations atomically.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, err
	}
	if err = migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}
func (s *Store) DB() *sql.DB  { return s.db }
func (s *Store) Close() error { return s.db.Close() }
func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		base := strings.TrimSuffix(strings.TrimPrefix(file, "migrations/"), ".sql")
		n, err := strconv.Atoi(strings.SplitN(base, "_", 2)[0])
		if err != nil || n < 1 {
			return fmt.Errorf("invalid migration filename %s", file)
		}
		var applied int
		if err = db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, n).Scan(&applied); err != nil {
			return err
		}
		if applied != 0 {
			continue
		}
		sqlBytes, err := migrations.ReadFile(file)
		if err != nil {
			return err
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err = tx.Exec(string(sqlBytes)); err == nil {
			_, err = tx.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(?,?)`, n, time.Now().UTC().Format(time.RFC3339Nano))
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", n, err)
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) Profile() (Profile, error) {
	var p Profile
	err := s.db.QueryRow(`SELECT id,name,theme,rank,xp FROM profiles WHERE id='local'`).Scan(&p.ID, &p.Name, &p.Theme, &p.Rank, &p.XP)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	return p, err
}
func (s *Store) CreateProfile(name, theme string) (Profile, error) {
	_, err := s.db.Exec(`INSERT INTO profiles(id,name,theme,rank,xp) VALUES('local',?,?, 'Zero',(SELECT COALESCE(SUM(xp_awarded),0) FROM verified_progress))`, name, theme)
	if err != nil {
		if isConstraint(err) {
			return Profile{}, ErrProfileExists
		}
		return Profile{}, err
	}
	return s.Profile()
}
func (s *Store) UpdateProfile(name, theme string) (Profile, error) {
	r, err := s.db.Exec(`UPDATE profiles SET name=?,theme=? WHERE id='local'`, name, theme)
	if err != nil {
		return Profile{}, err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return Profile{}, ErrProfileNotFound
	}
	return s.Profile()
}
func isConstraint(err error) bool {
	return err != nil && (contains(err.Error(), "constraint") || contains(err.Error(), "UNIQUE"))
}
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
func encodeFiles(files map[string]string) ([]byte, error) {
	if files == nil {
		files = map[string]string{}
	}
	return json.Marshal(files)
}
func decodeFiles(b string) (map[string]string, error) {
	var f map[string]string
	err := json.Unmarshal([]byte(b), &f)
	if f == nil {
		f = map[string]string{}
	}
	return f, err
}
func (s *Store) SaveDraft(exerciseID string, files map[string]string) error {
	b, err := encodeFiles(files)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO drafts(exercise_id,files_json,updated_at) VALUES(?,?,?) ON CONFLICT(exercise_id) DO UPDATE SET files_json=excluded.files_json,updated_at=excluded.updated_at`, exerciseID, string(b), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) Draft(exerciseID string) (map[string]string, error) {
	var b string
	err := s.db.QueryRow(`SELECT files_json FROM drafts WHERE exercise_id=?`, exerciseID).Scan(&b)
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeFiles(b)
}
func (s *Store) RecordAttempt(exerciseID string, files map[string]string, passed bool, output string) (Attempt, error) {
	b, err := encodeFiles(files)
	if err != nil {
		return Attempt{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Attempt{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	res, err := tx.Exec(`INSERT INTO attempts(exercise_id,files_json,passed,output,created_at) VALUES(?,?,?,?,?)`, exerciseID, string(b), boolInt(passed), output, now.Format(time.RFC3339Nano))
	if err != nil {
		return Attempt{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Attempt{}, err
	}
	if passed {
		_, err = tx.Exec(`INSERT OR IGNORE INTO verified_progress(exercise_id,attempt_id,xp_awarded) VALUES(?,?,10)`, exerciseID, id)
		if err != nil {
			return Attempt{}, err
		} /* XP is normalized from verified_progress to remain idempotent. */
		if _, err = tx.Exec(`UPDATE profiles SET xp=(SELECT COALESCE(SUM(xp_awarded),0) FROM verified_progress) WHERE id='local'`); err != nil {
			return Attempt{}, err
		}
	}
	// Keep recent history bounded while retaining the first verified attempt for every exercise.
	if _, err = tx.Exec(`DELETE FROM attempts
		WHERE id NOT IN (SELECT id FROM attempts ORDER BY id DESC LIMIT 100)
		AND id NOT IN (SELECT attempt_id FROM verified_progress)`); err != nil {
		return Attempt{}, err
	}
	if err = tx.Commit(); err != nil {
		return Attempt{}, err
	}
	return Attempt{ID: id, ExerciseID: exerciseID, Files: files, Passed: passed, Output: output, CreatedAt: now}, nil
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func (s *Store) Progress() (Progress, error) {
	p := Progress{Completed: []string{}, Attempts: []Attempt{}, Evidence: map[string]int64{}}
	rows, err := s.db.Query(`SELECT exercise_id FROM verified_progress ORDER BY exercise_id`)
	if err != nil {
		return p, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return p, err
		}
		p.Completed = append(p.Completed, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return p, err
	}
	rows.Close()
	rows, err = s.db.Query(`SELECT id,exercise_id,files_json,passed,output,created_at FROM attempts ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Attempt
		var b, stamp string
		var passed int
		if err = rows.Scan(&a.ID, &a.ExerciseID, &b, &passed, &a.Output, &stamp); err != nil {
			return p, err
		}
		a.Files, err = decodeFiles(b)
		if err != nil {
			return p, err
		}
		a.Passed = passed == 1
		a.CreatedAt, err = time.Parse(time.RFC3339Nano, stamp)
		if err != nil {
			return p, err
		}
		p.Attempts = append(p.Attempts, a)
	}
	if err = rows.Err(); err != nil {
		return p, err
	}
	// The query is newest-first for a bounded scan; return a chronological list to callers.
	for i, j := 0, len(p.Attempts)-1; i < j; i, j = i+1, j-1 {
		p.Attempts[i], p.Attempts[j] = p.Attempts[j], p.Attempts[i]
	}
	p.Evidence, err = s.VerifiedEvidence()
	if err != nil {
		return p, err
	}
	var xp sql.NullInt64
	err = s.db.QueryRow(`SELECT SUM(xp_awarded) FROM verified_progress`).Scan(&xp)
	if err != nil {
		return p, err
	}
	p.XP = int(xp.Int64)
	sort.Strings(p.Completed)
	return p, nil
}

// VerifiedEvidence returns the original successful attempt ID for each completed exercise.
func (s *Store) VerifiedEvidence() (map[string]int64, error) {
	rows, err := s.db.Query(`SELECT exercise_id,attempt_id FROM verified_progress ORDER BY exercise_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	evidence := make(map[string]int64)
	for rows.Next() {
		var exerciseID string
		var attemptID int64
		if err := rows.Scan(&exerciseID, &attemptID); err != nil {
			return nil, err
		}
		evidence[exerciseID] = attemptID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return evidence, nil
}

const (
	MaxProjectFiles       = 16
	MaxProjectSourceBytes = 128 * 1024
	MaxProjectSnapshots   = 10
)

type ProjectSnapshot struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// ProjectFiles returns the current file set for one project.
func (s *Store) ProjectFiles(projectID string) (map[string]string, error) {
	files := make(map[string]string)
	rows, err := s.db.Query(`SELECT path,source FROM project_files WHERE project_id=? ORDER BY path`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var filePath, source string
		if err := rows.Scan(&filePath, &source); err != nil {
			return nil, err
		}
		files[filePath] = source
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// SaveProjectFiles atomically replaces a project's files and records changed versions.
func (s *Store) SaveProjectFiles(projectID string, files map[string]string) error {
	if strings.TrimSpace(projectID) == "" {
		return errors.New("project id is required")
	}
	if err := validateProjectFiles(files); err != nil {
		return err
	}
	if files == nil {
		files = map[string]string{}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	old := make(map[string]string)
	rows, err := tx.Query(`SELECT path,source FROM project_files WHERE project_id=?`, projectID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var filePath, source string
		if err := rows.Scan(&filePath, &source); err != nil {
			rows.Close()
			return err
		}
		old[filePath] = source
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if sameFiles(old, files) {
		return tx.Commit()
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(`DELETE FROM project_files WHERE project_id=?`, projectID); err != nil {
		return err
	}
	filePaths := make([]string, 0, len(files))
	for filePath := range files {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)
	for _, filePath := range filePaths {
		if _, err := tx.Exec(`INSERT INTO project_files(project_id,path,source,updated_at) VALUES(?,?,?,?)`, projectID, filePath, files[filePath], now.Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	encoded, err := json.Marshal(files)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO project_snapshots(project_id,files_json,created_at) VALUES(?,?,?)`, projectID, string(encoded), now.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM project_snapshots WHERE project_id=? AND id NOT IN (SELECT id FROM project_snapshots WHERE project_id=? ORDER BY id DESC LIMIT ?)`, projectID, projectID, MaxProjectSnapshots); err != nil {
		return err
	}
	return tx.Commit()
}

func validateProjectFiles(files map[string]string) error {
	if len(files) > MaxProjectFiles {
		return fmt.Errorf("project has %d files; limit is %d", len(files), MaxProjectFiles)
	}
	total := 0
	for filePath, source := range files {
		if filePath == "" || strings.Contains(filePath, `\`) || strings.HasPrefix(filePath, "/") || strings.HasPrefix(filePath, "../") || filePath == ".." || path.Clean(filePath) != filePath || strings.HasPrefix(filePath, "./") {
			return fmt.Errorf("invalid project file path %q", filePath)
		}
		total += len([]byte(source))
	}
	if total > MaxProjectSourceBytes {
		return fmt.Errorf("project source is %d bytes; limit is %d", total, MaxProjectSourceBytes)
	}
	return nil
}

func sameFiles(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for name, source := range a {
		if other, ok := b[name]; !ok || source != other {
			return false
		}
	}
	return true
}

// ProjectSnapshots lists retained snapshot metadata newest first.
func (s *Store) ProjectSnapshots(projectID string) ([]ProjectSnapshot, error) {
	rows, err := s.db.Query(`SELECT id,created_at FROM project_snapshots WHERE project_id=? ORDER BY id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	snapshots := make([]ProjectSnapshot, 0)
	for rows.Next() {
		var snapshot ProjectSnapshot
		var createdAt string
		if err := rows.Scan(&snapshot.ID, &createdAt); err != nil {
			return nil, err
		}
		snapshot.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return snapshots, nil
}

// ProjectSnapshot returns snapshot files only when the snapshot belongs to projectID.
func (s *Store) ProjectSnapshot(projectID string, id int64) (map[string]string, error) {
	var encoded string
	err := s.db.QueryRow(`SELECT files_json FROM project_snapshots WHERE project_id=? AND id=?`, projectID, id).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProjectSnapshotNotFound
	}
	if err != nil {
		return nil, err
	}
	var files map[string]string
	if err := json.Unmarshal([]byte(encoded), &files); err != nil {
		return nil, err
	}
	if files == nil {
		files = map[string]string{}
	}
	return files, nil
}
