package http

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"

	"booking-system-api/internal/config"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"
)

// Riwayat booking (admin): GET /bookings/history & /bookings/history/export.
// Query: from, to (YYYY-MM-DD, wajib), resourceType, bookingType,
// status (dipisah koma), search; ekspor: sections (dipisah koma, kosong = semua).

func historyQuery(c *fiber.Ctx) service.BookingHistoryQuery {
	return service.BookingHistoryQuery{
		From:         c.Query("from"),
		To:           c.Query("to"),
		ResourceType: c.Query("resourceType"),
		BookingType:  c.Query("bookingType"),
		Statuses:     splitList(c.Query("status")),
		Search:       c.Query("search"),
	}
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (h *BookingHandler) History(c *fiber.Ctx) error {
	f, err := service.ParseBookingHistoryFilter(historyQuery(c))
	if err != nil {
		return err
	}
	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	data, total, err := h.svc.History(c.Context(), f, page, limit)
	if err != nil {
		return err
	}
	return util.Paginated(c, "Booking history retrieved", data, total, page, limit)
}

func (h *BookingHandler) ExportHistory(c *fiber.Ctx) error {
	f, err := service.ParseBookingHistoryFilter(historyQuery(c))
	if err != nil {
		return err
	}
	file, name, err := h.svc.ExportHistory(c.Context(), f, service.HistoryExportOptions{
		Sections: splitList(c.Query("sections")),
		FileBase: publicBaseURL(c),
		WebBase:  webBaseURL(),
		ActorID:  auditActor(c).UserID,
	})
	if err != nil {
		return err
	}
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", `attachment; filename="`+name+`"; filename*=UTF-8''`+url.PathEscape(name))
	return c.Send(file)
}

// publicBaseURL: alamat publik API untuk link /uploads (PUBLIC_BASE_URL bila
// diisi; selain itu dari header proxy / request).
func publicBaseURL(c *fiber.Ctx) string {
	if b := strings.TrimSpace(config.C.PublicBaseURL); b != "" {
		return strings.TrimRight(b, "/")
	}
	proto := c.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = c.Protocol()
	}
	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = string(c.Request().Host())
	}
	return proto + "://" + host
}

// webBaseURL: origin web pertama di FRONTEND_ORIGIN untuk link detail booking.
func webBaseURL() string {
	for _, o := range strings.Split(config.C.FrontendOrigin, ",") {
		if o = strings.TrimSpace(o); o != "" && o != "*" {
			return strings.TrimRight(o, "/")
		}
	}
	return ""
}
