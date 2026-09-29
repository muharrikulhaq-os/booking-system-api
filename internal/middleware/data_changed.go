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

// DataChanged memanggil publish setelah request TULIS (POST/PUT/PATCH/DELETE)
// sukses, dengan topik data yang berubah. Klien memakainya untuk memuat ulang
// hanya data yang memang berubah — tanpa polling.
//
// Topik yang dikirim adalah data yang DITULIS langsung; data turunan (mis.
// status kendaraan ikut berubah saat booking dimulai) ditangani klien yang
// bergantung pada topik tersebut.
func DataChanged(publish func(topics []string)) fiber.Handler {
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
			publish(topics)
		}
		return nil
	}
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
	case "fuel-expenses", "fuel-types", "master-settings", "settings":
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
