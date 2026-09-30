package service

import (
	"context"
	"errors"
	"testing"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

func TestDecideResourceStatus(t *testing.T) {
	const (
		avail   = repository.ResourceStatusAVAILABLE
		inUse   = repository.ResourceStatusINUSE
		maint   = repository.ResourceStatusMAINTENANCE
		inactiv = repository.ResourceStatusINACTIVE
	)
	cases := []struct {
		name    string
		facts   repository.ResourceStatusFacts
		release bool
		want    repository.ResourceStatus
	}{
		{"trip menang atas maintenance (B10)", repository.ResourceStatusFacts{Status: maint, HasActiveTrip: true, HasActiveMaintain: true}, true, inUse},
		{"trip selesai, maintenance berlangsung", repository.ResourceStatusFacts{Status: inUse, HasActiveMaintain: true}, false, maint},
		{"trip selesai, tanpa maintenance", repository.ResourceStatusFacts{Status: inUse}, false, avail},
		{"maintenance masa depan tidak mengunci (B11)", repository.ResourceStatusFacts{Status: avail}, false, avail},
		{"maintenance selesai melepas kunci (B12)", repository.ResourceStatusFacts{Status: maint}, true, avail},
		{"MAINTENANCE manual dipertahankan saat booking selesai", repository.ResourceStatusFacts{Status: maint}, false, maint},
		{"INACTIVE dipertahankan", repository.ResourceStatusFacts{Status: inactiv}, true, inactiv},
		{"INACTIVE tidak berubah karena maintenance", repository.ResourceStatusFacts{Status: inactiv, HasActiveMaintain: true}, true, inactiv},
	}
	for _, c := range cases {
		if got := decideResourceStatus(c.facts, c.release); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

type statusFake struct {
	repository.ExtendedQuerier
	facts   repository.ResourceStatusFacts
	updated *repository.ResourceStatus
}

func (f *statusFake) GetResourceStatusFacts(ctx context.Context, resourceID int32) (repository.ResourceStatusFacts, error) {
	return f.facts, nil
}

func (f *statusFake) UpdateResourceStatus(ctx context.Context, arg repository.UpdateResourceStatusParams) (repository.Resource, error) {
	s := arg.Status
	f.updated = &s
	return repository.Resource{ID: arg.ID, Status: arg.Status}, nil
}

func TestSyncResourceStatus_WritesOnlyOnChange(t *testing.T) {
	f := &statusFake{facts: repository.ResourceStatusFacts{Status: repository.ResourceStatusAVAILABLE}}
	syncResourceStatus(context.Background(), f, 7, true)
	if f.updated != nil {
		t.Fatalf("status sudah benar, tidak boleh ditulis ulang (got %s)", *f.updated)
	}
	f.facts = repository.ResourceStatusFacts{Status: repository.ResourceStatusAVAILABLE, HasActiveMaintain: true}
	syncResourceStatus(context.Background(), f, 7, false)
	if f.updated == nil || *f.updated != repository.ResourceStatusMAINTENANCE {
		t.Fatalf("expected MAINTENANCE, got %v", f.updated)
	}
}

func TestGuardManualStatusChange(t *testing.T) {
	cases := []struct {
		name     string
		facts    repository.ResourceStatusFacts
		next     repository.ResourceStatus
		conflict bool
	}{
		{"trip berjalan → MAINTENANCE ditolak", repository.ResourceStatusFacts{Status: repository.ResourceStatusINUSE, HasActiveTrip: true}, repository.ResourceStatusMAINTENANCE, true},
		{"trip berjalan → AVAILABLE ditolak", repository.ResourceStatusFacts{Status: repository.ResourceStatusINUSE, HasActiveTrip: true}, repository.ResourceStatusAVAILABLE, true},
		{"trip berjalan → IN_USE boleh", repository.ResourceStatusFacts{Status: repository.ResourceStatusINUSE, HasActiveTrip: true}, repository.ResourceStatusINUSE, false},
		{"maintenance berlangsung → AVAILABLE ditolak", repository.ResourceStatusFacts{Status: repository.ResourceStatusMAINTENANCE, HasActiveMaintain: true}, repository.ResourceStatusAVAILABLE, true},
		{"maintenance berlangsung → INACTIVE boleh", repository.ResourceStatusFacts{Status: repository.ResourceStatusMAINTENANCE, HasActiveMaintain: true}, repository.ResourceStatusINACTIVE, false},
		{"idle → MAINTENANCE manual boleh", repository.ResourceStatusFacts{Status: repository.ResourceStatusAVAILABLE}, repository.ResourceStatusMAINTENANCE, false},
	}
	for _, c := range cases {
		err := guardManualStatusChange(context.Background(), &statusFake{facts: c.facts}, 1, c.next)
		if got := errors.Is(err, util.ErrConflict); got != c.conflict {
			t.Errorf("%s: conflict=%v, err=%v", c.name, got, err)
		}
	}
}
