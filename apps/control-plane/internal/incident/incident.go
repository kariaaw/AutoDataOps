package incident

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Severity represents business impact.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

func (s Severity) valid() bool {
	switch s {
	case SeverityLow,
		SeverityMedium,
		SeverityHigh,
		SeverityCritical:
		return true
	default:
		return false
	}
}

// Status represents the incident lifecycle.
type Status string

const (
	StatusDetected      Status = "detected"
	StatusInvestigating Status = "investigating"
	StatusMitigating    Status = "mitigating"
	StatusResolved      Status = "resolved"
	StatusClosed        Status = "closed"
	StatusDismissed     Status = "dismissed"
)

// Input describes a newly detected incident.
type Input struct {
	Source     string
	PipelineID string
	Summary    string
	Severity   Severity
	DetectedAt time.Time
}

// Verification represents a successful or failed
// validation check.
//
// The application layer must authenticate this
// evidence before allowing a trusted caller to
// supply it to the domain.
type Verification struct {
	ID        string
	Passed    bool
	CheckedAt time.Time
}

// Transition records a lifecycle state change.
type Transition struct {
	From           Status
	To             Status
	Reason         string
	VerificationID string
	At             time.Time
}

// Incident is the domain aggregate.
//
// Fields are private to prevent callers from
// directly bypassing lifecycle rules.
type Incident struct {
	id         string
	source     string
	pipelineID string
	summary    string
	severity   Severity
	status     Status
	detectedAt time.Time
	createdAt  time.Time
	updatedAt  time.Time
	history    []Transition
}

// New validates input and creates an incident.
func New(input Input, now time.Time) (*Incident, error) {

	input.Source = strings.TrimSpace(input.Source)
	input.PipelineID = strings.TrimSpace(input.PipelineID)
	input.Summary = strings.TrimSpace(input.Summary)

	if err := bounded(input.Source, "source", 128); err != nil {
		return nil, err
	}

	if err := bounded(input.PipelineID, "pipeline_id", 128); err != nil {
		return nil, err
	}

	if err := bounded(input.Summary, "summary", 512); err != nil {
		return nil, err
	}

	if !input.Severity.valid() {
		return nil, errors.New("invalid severity")
	}

	if input.DetectedAt.IsZero() || now.IsZero() {
		return nil, errors.New("timestamps must not be zero")
	}

	id, err := randomUUID()

	if err != nil {
		return nil, fmt.Errorf(
			"generate incident id: %w",
			err,
		)
	}

	now = now.UTC()

	return &Incident{
		id:         id,
		source:     input.Source,
		pipelineID: input.PipelineID,
		summary:    input.Summary,
		severity:   input.Severity,
		status:     StatusDetected,
		detectedAt: input.DetectedAt.UTC(),
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

// Read-only accessors.
func (i *Incident) ID() string {
	return i.id
}

func (i *Incident) Status() Status {
	return i.status
}

func (i *Incident) Severity() Severity {
	return i.severity
}

func (i *Incident) Source() string {
	return i.source
}

func (i *Incident) PipelineID() string {
	return i.pipelineID
}

func (i *Incident) Summary() string {
	return i.summary
}

func (i *Incident) DetectedAt() time.Time {
	return i.detectedAt
}

func (i *Incident) CreatedAt() time.Time {
	return i.createdAt
}

func (i *Incident) UpdatedAt() time.Time {
	return i.updatedAt
}

// History returns a defensive copy.
func (i *Incident) History() []Transition {
	return append([]Transition(nil), i.history...)
}

// Move performs a controlled lifecycle transition.
//
// The application layer must verify permissions,
// evidence authenticity, and database consistency.
func (i *Incident) Move(
	next Status,
	reason string,
	verification *Verification,
	at time.Time,
) error {

	if i == nil {
		return errors.New("incident is nil")
	}

	reason = strings.TrimSpace(reason)

	if err := bounded(reason, "reason", 1024); err != nil {
		return err
	}

	if at.IsZero() || at.Before(i.updatedAt) {
		return errors.New(
			"transition time must not precede last update",
		)
	}

	if !allowed(i.status, next) {
		return fmt.Errorf(
			"invalid incident transition: %s -> %s",
			i.status,
			next,
		)
	}

	var verificationID string

	if next == StatusResolved {

		if verification == nil ||
			!verification.Passed ||
			strings.TrimSpace(verification.ID) == "" ||
			verification.CheckedAt.IsZero() ||
			verification.CheckedAt.Before(i.updatedAt) ||
			verification.CheckedAt.After(at) {

			return errors.New(
				"resolution requires a recent successful verification",
			)
		}

		verificationID = strings.TrimSpace(
			verification.ID,
		)

		if utf8.RuneCountInString(verificationID) > 256 {
			return errors.New(
				"verification id is too long",
			)
		}

	} else if verification != nil {

		return errors.New(
			"verification is only accepted when resolving",
		)
	}

	i.history = append(
		i.history,
		Transition{
			From:           i.status,
			To:             next,
			Reason:         reason,
			VerificationID: verificationID,
			At:             at.UTC(),
		},
	)

	i.status = next
	i.updatedAt = at.UTC()

	return nil
}

// allowed defines the finite state machine.
func allowed(from, to Status) bool {

	switch from {

	case StatusDetected:
		return to == StatusInvestigating ||
			to == StatusDismissed

	case StatusInvestigating:
		return to == StatusMitigating ||
			to == StatusDismissed

	case StatusMitigating:
		return to == StatusInvestigating ||
			to == StatusResolved

	case StatusResolved:
		return to == StatusInvestigating ||
			to == StatusClosed

	default:
		return false
	}
}

// bounded validates required text lengths.
func bounded(value, field string, max int) error {

	n := utf8.RuneCountInString(value)

	if n == 0 || n > max {
		return fmt.Errorf(
			"%s length must be between 1 and %d characters",
			field,
			max,
		)
	}

	return nil
}

// randomUUID creates an RFC 4122-style UUID v4
// using cryptographically secure randomness.
func randomUUID() (string, error) {

	var b [16]byte

	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	), nil
}
