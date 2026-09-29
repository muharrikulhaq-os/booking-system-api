package middleware

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestTopicsForPath(t *testing.T) {
	cases := []struct {
		path string
		want []string
	}{
		{"/api/v1/bookings", []string{TopicBooking}},
		{"/api/v1/bookings/12/approve", []string{TopicBooking}},
		{"/api/v1/guest-bookings/abc/cancel", []string{TopicBooking}},
		{"/api/v1/vehicles/3/status", []string{TopicVehicle}},
		{"/api/v1/vehicles/3/fixed-driver", []string{TopicVehicle, TopicDriver}},
		{"/api/v1/drivers/4/fixed-vehicle", []string{TopicDriver, TopicVehicle}},
		{"/api/v1/rooms/5/room-keeper", []string{TopicRoom, TopicRoomKeeper}},
		{"/api/v1/room-keepers/6/toggle-active", []string{TopicRoomKeeper}},
		{"/api/v1/fuel-expenses/bbm", []string{TopicFuel}},
		{"/api/v1/maintenance/7/complete", []string{TopicMaintenance}},
		{"/api/v1/users/8/toggle-active", []string{TopicUser, TopicDriver, TopicRoomKeeper}},
		{"/api/v1/users/me/profile-photo", []string{TopicUser, TopicDriver, TopicRoomKeeper}},
		{"/api/v1/attachments/9", []string{TopicBooking, TopicVehicle, TopicRoom}},
		// Tidak mengubah data bersama.
		{"/api/v1/auth/login", nil},
		{"/api/v1/auth/logout", nil},
		{"/api/v1/users/me/notifications/read-all", nil},
		{"/api/v1/users/me/device-tokens", nil},
		{"/api/v1", nil},
	}
	for _, tc := range cases {
		if got := TopicsForPath(tc.path); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("TopicsForPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestDataChanged_PublishesOnlySuccessfulWrites(t *testing.T) {
	var published [][]string
	app := fiber.New()
	app.Use(DataChanged(func(topics []string) { published = append(published, topics) }))
	app.Post("/api/v1/bookings", func(c *fiber.Ctx) error { return c.SendStatus(201) })
	app.Get("/api/v1/bookings", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Patch("/api/v1/vehicles/1/status", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusConflict, "bentrok")
	})
	app.Put("/api/v1/rooms/1", func(c *fiber.Ctx) error { return c.SendStatus(400) })

	// Hanya POST pertama (sukses) yang boleh memicu publish: GET hanya baca,
	// PATCH mengembalikan error, PUT membalas 400.
	for _, r := range []struct{ method, path string }{
		{"POST", "/api/v1/bookings"},
		{"GET", "/api/v1/bookings"},
		{"PATCH", "/api/v1/vehicles/1/status"},
		{"PUT", "/api/v1/rooms/1"},
	} {
		if _, err := app.Test(httptest.NewRequest(r.method, r.path, nil)); err != nil {
			t.Fatalf("%s %s: %v", r.method, r.path, err)
		}
	}

	want := [][]string{{TopicBooking}}
	if !reflect.DeepEqual(published, want) {
		t.Errorf("published = %v, want %v", published, want)
	}
}
