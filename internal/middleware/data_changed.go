package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Topik data yang dikirim lewat event WebSocket DATA_CHANGED. Nama topik
// adalah kontrak dengan klien (mobile: SyncTopic.wireName, web: query key).
const (
	TopicBooking     = "booking"
	TopicVehicle     = "vehicle"
	TopicRoom        = "room"
	TopicDriver      = "driver"
	TopicUser        = "user"
	TopicRoomKeeper  = "roomKeeper"
	TopicFuel        = "fuel"
	TopicMaintenance = "maintenance"
)

const (
	// HeaderDataChanged ada di respons request tulis yang sukses, berisi topik
	// yang berubah (dipisah koma) — klien pengirim meng-invalidate datanya
	// sendiri seketika tanpa menunggu WebSocket.
	HeaderDataChanged = "X-Data-Changed"
	// HeaderClientID dikirim klien (id acak per tab/aplikasi) dan diteruskan
	// sebagai `origin` di event DATA_CHANGED, supaya pengirim bisa
	// mengabaikan event miliknya sendiri (sudah ditangani lewat header di
	// atas) dan tidak fetch dua kali.
	HeaderClientID = "X-Client-Id"
)

// DataChanged memanggil publish setelah request TULIS (POST/PUT/PATCH/DELETE)
// sukses, dengan topik data yang berubah dan id klien pengirimnya. Klien
// memakainya untuk memuat ulang hanya data yang memang berubah — tanpa
// polling.
//
// Topik yang dikirim adalah data yang DITULIS langsung; data turunan (mis.
// status kendaraan ikut berubah saat booking dimulai) ditangani klien yang
// bergantung pada topik tersebut.
func DataChanged(publish func(topics []string, origin string)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.Next(); err != nil {
			return err // gagal → tidak ada data yang berubah
		}
		switch c.Method() {
		case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		default:
			return nil
		}
		if status := c.Response().StatusCode(); status < 200 || status >= 300 {
			return nil
		}
		if topics := TopicsForPath(c.Path()); len(topics) > 0 {
			c.Set(HeaderDataChanged, strings.Join(topics, ","))
			publish(topics, clientID(c.Get(HeaderClientID)))
		}
		return nil
	}
}

// clientID menerima id klien hanya bila berbentuk token pendek yang aman
// (huruf, angka, '-', '_'); selain itu dianggap tidak ada.
//
// Nilai yang dikembalikan adalah SALINAN: string dari c.Get() di Fiber
// menunjuk ke buffer fasthttp yang dipakai ulang request berikutnya, jadi
// tanpa salinan origin yang dipakai di luar handler bisa berubah isi.
func clientID(v string) string {
	v = safeClientID(v)
	if v == "" {
		return ""
	}
	return strings.Clone(v)
}

func safeClientID(v string) string {
	if v == "" || len(v) > 64 {
		return ""
	}
	for _, r := range v {
		ok := r == '-' || r == '_' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !ok {
			return ""
		}
	}
	return v
}

// TopicsForPath memetakan path request tulis ke topik data yang berubah.
// nil = request tidak mengubah data bersama (login, notifikasi pribadi, dsb).
func TopicsForPath(path string) []string {
	p := strings.TrimPrefix(path, "/api/v1")
	seg := strings.Split(strings.Trim(p, "/"), "/")
	if len(seg) == 0 || seg[0] == "" {
		return nil
	}
	last := seg[len(seg)-1]

	switch seg[0] {
	case "auth":
		if len(seg) > 1 && seg[1] == "register" {
			return []string{TopicUser, TopicDriver, TopicRoomKeeper}
		}
		return nil // login/logout/refresh/password — tidak mengubah data bersama
	case "users":
		if len(seg) > 2 && seg[1] == "me" && seg[2] != "profile-photo" {
			return nil // notifikasi & device token pribadi
		}
		// Driver & penjaga ruangan adalah pengguna: nama, foto, dan status
		// aktif akunnya ikut tampil di daftar mereka.
		return []string{TopicUser, TopicDriver, TopicRoomKeeper}
	case "bookings", "guest-bookings":
		return []string{TopicBooking}
	case "vehicles":
		if last == "fixed-driver" {
			return []string{TopicVehicle, TopicDriver}
		}
		return []string{TopicVehicle}
	case "drivers":
		if last == "fixed-vehicle" {
			return []string{TopicDriver, TopicVehicle}
		}
		return []string{TopicDriver}
	case "rooms":
		if last == "room-keeper" {
			return []string{TopicRoom, TopicRoomKeeper}
		}
		return []string{TopicRoom}
	case "room-keepers":
		return []string{TopicRoomKeeper}
	case "fuel-expenses", "fuel-vouchers", "fuel-balances":
		if last == "preview" {
			return nil // hanya menghitung, tidak menyimpan
		}
		// Pengisian/voucher/saldo memajukan odometer & mengubah saldo kendaraan.
		return []string{TopicFuel, TopicVehicle}
	case "fuel-types", "fuel-stations", "master-settings", "settings":
		return []string{TopicFuel}
	case "maintenance":
		return []string{TopicMaintenance}
	case "attachments":
		// DELETE /attachments/:id — pemiliknya (booking/kendaraan/ruangan)
		// tidak terlihat dari path.
		return []string{TopicBooking, TopicVehicle, TopicRoom}
	}
	return nil
}
