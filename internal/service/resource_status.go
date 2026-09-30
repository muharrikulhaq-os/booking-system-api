package service

import (
	"context"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// decideResourceStatus menentukan status resource dari keadaan nyata, dengan
// prioritas: sedang dipakai trip → IN_USE; dinonaktifkan admin → INACTIVE;
// maintenance sedang berlangsung → MAINTENANCE; selain itu AVAILABLE.
//
// MAINTENANCE yang diset manual (tanpa record) dipertahankan, kecuali
// releaseManualMaintenance — dipakai alur maintenance (selesai/hapus/ubah)
// karena di sana MAINTENANCE memang berasal dari record tersebut.
//
// Dulu setiap alur menimpa status langsung (selesai booking → AVAILABLE,
// buat maintenance → MAINTENANCE, hapus maintenance → AVAILABLE), sehingga
// status saling tertimpa (B10) atau terkunci tanpa alasan (B11, B12).
func decideResourceStatus(f repository.ResourceStatusFacts, releaseManualMaintenance bool) repository.ResourceStatus {
	switch {
	case f.HasActiveTrip:
		return repository.ResourceStatusINUSE
	case f.Status == repository.ResourceStatusINACTIVE:
		return repository.ResourceStatusINACTIVE
	case f.HasActiveMaintain:
		return repository.ResourceStatusMAINTENANCE
	case f.Status == repository.ResourceStatusMAINTENANCE && !releaseManualMaintenance:
		return repository.ResourceStatusMAINTENANCE
	default:
		return repository.ResourceStatusAVAILABLE
	}
}

// syncResourceStatus menghitung ulang & menyimpan status satu resource.
func syncResourceStatus(ctx context.Context, q repository.ExtendedQuerier, resourceID int32, releaseManualMaintenance bool) {
	if resourceID == 0 {
		return
	}
	f, err := q.GetResourceStatusFacts(ctx, resourceID)
	if err != nil {
		return
	}
	if next := decideResourceStatus(f, releaseManualMaintenance); next != f.Status {
		_, _ = q.UpdateResourceStatus(ctx, repository.UpdateResourceStatusParams{ID: resourceID, Status: next})
	}
}

// guardManualStatusChange menolak perubahan status manual yang bertentangan
// dengan keadaan nyata (B10): resource yang sedang dipakai trip, atau
// dijadikan AVAILABLE padahal maintenance-nya masih berlangsung.
func guardManualStatusChange(ctx context.Context, q repository.ExtendedQuerier, resourceID int32, next repository.ResourceStatus) error {
	f, err := q.GetResourceStatusFacts(ctx, resourceID)
	if err != nil {
		return err
	}
	if f.HasActiveTrip && next != repository.ResourceStatusINUSE {
		return util.NewError(409, "sedang dipakai perjalanan/booking yang berjalan - selesaikan booking-nya dulu", util.ErrConflict)
	}
	if next == repository.ResourceStatusAVAILABLE && f.HasActiveMaintain {
		return util.NewError(409, "kendaraan masih dalam maintenance yang berlangsung - selesaikan maintenance-nya dulu", util.ErrConflict)
	}
	return nil
}
