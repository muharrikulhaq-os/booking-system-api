package middleware

import (
	"context"
	"sync"
	"time"
)

// AccountState adalah status akun terkini di database.
type AccountState struct {
	Active bool
	Role   string
}

// accountCacheTTL: batas waktu status akun di-cache. Akun yang dinonaktifkan
// (atau diubah role-nya) terputus paling lambat selama ini, tanpa membebani
// database dengan satu query per request.
const accountCacheTTL = 15 * time.Second

type cachedAccount struct {
	state   AccountState
	expires time.Time
}

var (
	accountLookup func(ctx context.Context, userID int) (AccountState, error)
	accountCache  sync.Map // int → cachedAccount
)

// SetAccountLookup memasang pembaca status akun (dari main.go). Tanpa ini,
// Auth() hanya memeriksa token seperti sebelumnya.
func SetAccountLookup(f func(ctx context.Context, userID int) (AccountState, error)) {
	accountLookup = f
}

// accountAllowed: false bila akun sudah dinonaktifkan atau role-nya berubah
// sejak token dibuat (AU-05b) - klien lalu refresh token (role baru) atau
// keluar (akun nonaktif). Bila database sedang bermasalah, request tetap
// diteruskan (token sudah sah) daripada memutus semua pengguna.
func accountAllowed(ctx context.Context, userID int, tokenRole string) bool {
	if accountLookup == nil {
		return true
	}
	now := time.Now()
	if v, ok := accountCache.Load(userID); ok {
		if ca := v.(cachedAccount); now.Before(ca.expires) {
			return ca.state.Active && ca.state.Role == tokenRole
		}
	}
	st, err := accountLookup(ctx, userID)
	if err != nil {
		return true
	}
	accountCache.Store(userID, cachedAccount{state: st, expires: now.Add(accountCacheTTL)})
	return st.Active && st.Role == tokenRole
}
