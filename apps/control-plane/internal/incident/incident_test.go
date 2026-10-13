package incident

import (
	"strings"
	"testing"
	"time"
)

func exampleInput(at time.Time) Input {
	return Input{
		Source:     "airflow",
		PipelineID: "billing.daily_close",
		Summary:    "Pipeline processing failed",
		Severity:   SeverityHigh,
		DetectedAt: at.Add(-time.Minute),
	}
}

func mustIncident(t *testing.T, at time.Time) *Incident {
	t.Helper()

	result, err := New(exampleInput(at), at)

	if err != nil {
		t.Fatal(err)
	}

	return result
}

func TestNewIncident(t *testing.T) {

	at := time.Date(
		2026, 10, 12, 12, 0, 0, 0, time.UTC,
	)

	first := mustIncident(t, at)
	second := mustIncident(t, at)

	if first.ID() == second.ID() || len(first.ID()) != 36 {
		t.Fatal("incident IDs must be unique UUIDs")
	}

	if first.Status() != StatusDetected ||
		first.Severity() != SeverityHigh {
		t.Fatal("incorrect initial incident state")
	}

	if first.PipelineID() != "billing.daily_close" ||
		first.DetectedAt() != at.Add(-time.Minute) {
		t.Fatal("incorrect incident fields")
	}
}

func TestNewIncidentRejectsInvalidInput(t *testing.T) {

	at := time.Now().UTC()

	cases := []struct {
		name   string
		change func(*Input)
	}{
		{
			"blank source",
			func(i *Input) { i.Source = "  " },
		},
		{
			"blank pipeline",
			func(i *Input) { i.PipelineID = "" },
		},
		{
			"blank summary",
			func(i *Input) { i.Summary = "" },
		},
		{
			"summary too long",
			func(i *Input) {
				i.Summary = strings.Repeat("X", 513)
			},
		},
		{
			"invalid severity",
			func(i *Input) {
				i.Severity = Severity("urgent")
			},
		},
		{
			"missing timestamp",
			func(i *Input) {
				i.DetectedAt = time.Time{}
			},
		},
	}

	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {

			input := exampleInput(at)
			tc.change(&input)

			if _, err := New(input, at); err == nil {
				t.Fatal(
					"expected invalid input to be rejected",
				)
			}
		})
	}
}

func TestLifecycleRequiresVerification(t *testing.T) {

	at := time.Date(
		2026, 10, 12, 12, 0, 0, 0, time.UTC,
	)

	item := mustIncident(t, at)

	err := item.Move(
		StatusInvestigating,
		"triage started",
		nil,
		at.Add(time.Second),
	)

	if err != nil {
		t.Fatal(err)
	}

	err = item.Move(
		StatusMitigating,
		"retry proposed",
		nil,
		at.Add(2*time.Second),
	)

	if err != nil {
		t.Fatal(err)
	}

	// Resolution without evidence must fail.
	err = item.Move(
		StatusResolved,
		"retry succeeded",
		nil,
		at.Add(3*time.Second),
	)

	if err == nil {
		t.Fatal(
			"resolution without verification must fail",
		)
	}

	if item.Status() != StatusMitigating ||
		len(item.History()) != 2 {
		t.Fatal(
			"failed transition must not mutate incident",
		)
	}

	proof := &Verification{
		ID:        "validation-check-42",
		Passed:    true,
		CheckedAt: at.Add(2500 * time.Millisecond),
	}

	err = item.Move(
		StatusResolved,
		"data validated",
		proof,
		at.Add(3*time.Second),
	)

	if err != nil {
		t.Fatal(err)
	}

	err = item.Move(
		StatusClosed,
		"incident reviewed",
		nil,
		at.Add(4*time.Second),
	)

	if err != nil {
		t.Fatal(err)
	}

	// Closed incidents are terminal.
	err = item.Move(
		StatusInvestigating,
		"reopen",
		nil,
		at.Add(5*time.Second),
	)

	if err == nil {
		t.Fatal(
			"closed incident must remain terminal",
		)
	}

	history := item.History()

	if len(history) != 4 ||
		history[2].VerificationID != "validation-check-42" {
		t.Fatal("unexpected history")
	}

	// Callers must not mutate internal history.
	history[0].Reason = "tampered"

	if item.History()[0].Reason == "tampered" {
		t.Fatal(
			"history should be immutable to callers",
		)
	}
}

func TestInvalidTransitionsAndTimestamps(t *testing.T) {

	at := time.Now().UTC()
	item := mustIncident(t, at)

	err := item.Move(
		StatusClosed,
		"skip steps",
		nil,
		at.Add(time.Second),
	)

	if err == nil {
		t.Fatal("skipped transition permitted")
	}

	err = item.Move(
		StatusInvestigating,
		"backdated",
		nil,
		at.Add(-time.Second),
	)

	if err == nil {
		t.Fatal("backdated transition permitted")
	}

	err = item.Move(
		StatusInvestigating,
		"",
		nil,
		at.Add(time.Second),
	)

	if err == nil {
		t.Fatal("blank reason permitted")
	}

	err = item.Move(
		StatusDismissed,
		"false alarm",
		nil,
		at.Add(time.Second),
	)

	if err != nil {
		t.Fatal(err)
	}

	err = item.Move(
		StatusInvestigating,
		"not allowed",
		nil,
		at.Add(2*time.Second),
	)

	if err == nil {
		t.Fatal("dismissed incident was reopened")
	}
}

func TestFailedVerificationCannotResolve(t *testing.T) {

	at := time.Now().UTC()
	item := mustIncident(t, at)

	err := item.Move(
		StatusInvestigating,
		"analysis",
		nil,
		at.Add(time.Second),
	)

	if err != nil {
		t.Fatal(err)
	}

	err = item.Move(
		StatusMitigating,
		"repair",
		nil,
		at.Add(2*time.Second),
	)

	if err != nil {
		t.Fatal(err)
	}

	cases := []Verification{
		{
			ID:        "check-1",
			Passed:    false,
			CheckedAt: at.Add(2500 * time.Millisecond),
		},
		{
			ID:        "",
			Passed:    true,
			CheckedAt: at.Add(2500 * time.Millisecond),
		},
		{
			ID:        "check-2",
			Passed:    true,
			CheckedAt: at.Add(time.Second),
		},
		{
			ID:        "check-3",
			Passed:    true,
			CheckedAt: at.Add(4 * time.Second),
		},
	}

	for _, v := range cases {

		err := item.Move(
			StatusResolved,
			"not valid",
			&v,
			at.Add(3*time.Second),
		)

		if err == nil {
			t.Fatalf(
				"invalid verification permitted: %+v",
				v,
			)
		}
	}

	if item.Status() != StatusMitigating {
		t.Fatal(
			"status changed after invalid verification",
		)
	}
}
