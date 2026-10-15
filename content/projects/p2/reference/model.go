package main

import "time"

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}
type Session struct {
	UserID  string
	Expires time.Time
}
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Link struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	TargetURL string    `json:"target_url"`
	Enabled   bool      `json:"enabled"`
	Tags      []Tag     `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type LinkInput struct {
	Slug      string   `json:"slug"`
	TargetURL string   `json:"target_url"`
	Enabled   *bool    `json:"enabled"`
	Tags      []string `json:"tags"`
}
type TagInput struct {
	Name string `json:"name"`
}

// Store methods must enforce ownership within the query/transaction that reads or mutates the row.
type Store interface {
	CreateUser(email string, passwordHash []byte) (User, error)
	UserByEmail(email string) (User, []byte, error)
	CreateSession(tokenHash, csrfHash, userID string, expires time.Time) error
	Session(tokenHash string) (User, string, time.Time, error)
	DeleteSession(tokenHash string) error
	CreateLink(owner string, in LinkInput) (Link, error)
	ListLinks(owner string, limit, offset int) ([]Link, error)
	GetLink(owner, id string) (Link, error)
	UpdateLink(owner, id string, in LinkInput) (Link, error)
	DeleteLink(owner, id string) error
	CreateTag(owner string, in TagInput) (Tag, error)
	ListTags(owner string) ([]Tag, error)
	UpdateTag(owner, id string, in TagInput) (Tag, error)
	DeleteTag(owner, id string) error
	RedirectTarget(slug string) (string, error)
}
