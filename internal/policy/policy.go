package policy

import (
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"pulsegrid/internal/domain"
)

type Decision struct {
	Allowed    bool          `json:"allowed"`
	Reason     string        `json:"reason,omitempty"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

type Limiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]*bucket
}

type bucket struct {
	count int
	start time.Time
}

type Suppression struct {
	mu     sync.RWMutex
	emails map[string]Suppressed
}

type Suppressed struct {
	Email     string     `json:"email"`
	Reason    string     `json:"reason"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Validator struct {
	MaxName int
	MaxBody int
	MaxTags int
}

func NewLimiter(limit int, window time.Duration) *Limiter {
	if limit < 1 {
		limit = 100
	}
	if window <= 0 {
		window = time.Minute
	}
	return &Limiter{limit: limit, window: window, buckets: map[string]*bucket{}}
}

func (l *Limiter) Allow(key string, now time.Time) Decision {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	value := l.buckets[key]
	if value == nil || now.Sub(value.start) >= l.window {
		l.buckets[key] = &bucket{count: 1, start: now}
		return Decision{Allowed: true}
	}
	if value.count >= l.limit {
		return Decision{Allowed: false, Reason: "rate limit exceeded", RetryAfter: l.window - now.Sub(value.start)}
	}
	value.count++
	return Decision{Allowed: true}
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.buckets, key)
}

func (l *Limiter) Snapshot() map[string]int {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := map[string]int{}
	for key, value := range l.buckets {
		result[key] = value.count
	}
	return result
}

func NewSuppression() *Suppression { return &Suppression{emails: map[string]Suppressed{}} }

func (s *Suppression) Add(email, reason string, expiresAt *time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emails[strings.ToLower(strings.TrimSpace(email))] = Suppressed{Email: strings.ToLower(strings.TrimSpace(email)), Reason: reason, CreatedAt: time.Now().UTC(), ExpiresAt: cloneTime(expiresAt)}
}

func (s *Suppression) Remove(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.emails, strings.ToLower(strings.TrimSpace(email)))
}

func (s *Suppression) Check(email string, now time.Time) Decision {
	s.mu.RLock()
	value, ok := s.emails[strings.ToLower(strings.TrimSpace(email))]
	s.mu.RUnlock()
	if !ok {
		return Decision{Allowed: true}
	}
	if value.ExpiresAt != nil && value.ExpiresAt.Before(now) {
		s.Remove(email)
		return Decision{Allowed: true}
	}
	return Decision{Allowed: false, Reason: value.Reason}
}

func (s *Suppression) List() []Suppressed {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Suppressed, 0, len(s.emails))
	for _, value := range s.emails {
		result = append(result, value)
	}
	return result
}

func NewValidator() Validator { return Validator{MaxName: 160, MaxBody: 100000, MaxTags: 20} }

func (v Validator) Organization(input string) error {
	return v.name(input, "organization")
}

func (v Validator) Contact(email, name string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email: %w", domain.ErrInvalidInput)
	}
	return v.name(name, "contact")
}

func (v Validator) Template(subject, body string) error {
	if err := v.name(subject, "subject"); err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" || len([]rune(body)) > v.MaxBody {
		return fmt.Errorf("body length is invalid: %w", domain.ErrInvalidInput)
	}
	return nil
}

func (v Validator) Tags(tags []string) error {
	if len(tags) > v.MaxTags {
		return fmt.Errorf("too many tags: %w", domain.ErrInvalidInput)
	}
	for _, tag := range tags {
		if len([]rune(tag)) > 40 {
			return fmt.Errorf("tag is too long: %w", domain.ErrInvalidInput)
		}
	}
	return nil
}

func (v Validator) name(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > v.MaxName {
		return fmt.Errorf("%s length is invalid: %w", field, domain.ErrInvalidInput)
	}
	return nil
}

func CanSend(contact domain.Contact, suppression *Suppression, now time.Time) Decision {
	if contact.Status != domain.ContactSubscribed {
		return Decision{Allowed: false, Reason: "contact is not subscribed"}
	}
	if suppression != nil {
		return suppression.Check(contact.Email, now)
	}
	return Decision{Allowed: true}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
