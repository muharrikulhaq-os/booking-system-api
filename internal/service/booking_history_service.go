package service

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Riwayat booking (admin): daftar dengan filter gabungan + ekspor Excel
// banyak sheet. Lihat docs/RANCANGAN_RIWAYAT_BOOKING.md.

// BookingHistoryQuery: parameter mentah dari query string.
type BookingHistoryQuery struct {
	From, To     string // YYYY-MM-DD (WIB), wajib
	ResourceType string
	BookingType  string
	Statuses     []string
	Search       string
	SortBy       string // kunci kolom (lihat repository.BookingHistorySortColumns)
	SortOrder    string // asc | desc
}

const historyMaxDays = 366
const historyMaxExportRows = 20000

var historyStatusLabel = map[string]string{
	"PENDING": "Menunggu", "APPROVED": "Disetujui", "REJECTED": "Ditolak",
	"ONGOING": "Berlangsung", "COMPLETED": "Selesai", "CANCELLED": "Dibatalkan",
	"OVERDUE": "Terlambat", "RETURNED": "Sudah Kembali", "EXPIRED": "Kedaluwarsa",
	"IGNORED": "Terabaikan",
}

// Urutan tampilan status di sheet Info.
var historyStatusOrder = []string{"PENDING", "APPROVED", "ONGOING", "OVERDUE", "RETURNED",
	"COMPLETED", "REJECTED", "CANCELLED", "EXPIRED", "IGNORED"}

var historyActionLabel = map[string]string{
	"CREATE": "Booking Dibuat", "APPROVE": "Disetujui", "REJECT": "Ditolak", "CANCEL": "Dibatalkan",
	"ASSIGN": "Driver & Kendaraan Ditugaskan", "START": "Dimulai", "COMPLETE": "Selesai",
	"RATE_DRIVER": "Rating Supir Diberikan", "RATE_ROOM": "Rating Ruangan Diberikan",
	"SUBSTITUTE_RESOURCE": "Resource Dialihkan", "MERGE": "Booking Digabungkan",
	"SUBMIT_RETURN_REPORT": "Laporan Pengembalian Dikirim", "OVERDUE": "Terlambat Dikembalikan",
	"EXPIRED": "Hangus (Tidak Dimulai)", "IGNORED": "Hangus (Tanpa Respons Admin)",
}

var historyExpenseCategory = map[string]string{"TOLL": "Tol", "PARKING": "Parkir", "OTHER": "Lainnya"}

// Bagian ekspor yang bisa dipilih (sheet Info & Booking selalu ada).
var HistorySections = []string{"passengers", "departure", "fuel", "return", "expenses", "ratings", "activity", "attachments"}

var historySectionLabel = map[string]string{
	"passengers": "Penumpang", "departure": "Keberangkatan", "fuel": "Pengisian BBM",
	"return": "Laporan Pengembalian", "expenses": "Biaya Tol & Parkir", "ratings": "Rating",
	"activity": "Riwayat Aktivitas", "attachments": "Lampiran & Foto",
}

func badHistory(msg string) error { return util.NewError(400, msg, util.ErrBadRequest) }

// ParseBookingHistoryFilter memvalidasi filter. Periode wajib (maks. 1 tahun),
// dihitung per hari WIB; tanggal akhir inklusif.
func ParseBookingHistoryFilter(p BookingHistoryQuery) (repository.BookingHistoryFilter, error) {
	var f repository.BookingHistoryFilter
	from, err := time.ParseInLocation("2006-01-02", p.From, util.WIB)
	if err != nil {
		return f, badHistory("tanggal awal wajib diisi (format YYYY-MM-DD)")
	}
	to, err := time.ParseInLocation("2006-01-02", p.To, util.WIB)
	if err != nil {
		return f, badHistory("tanggal akhir wajib diisi (format YYYY-MM-DD)")
	}
	if to.Before(from) {
		return f, badHistory("tanggal akhir tidak boleh sebelum tanggal awal")
	}
	if to.Sub(from) > historyMaxDays*24*time.Hour {
		return f, badHistory("periode maksimal 1 tahun")
	}
	f.From, f.To = from, to.AddDate(0, 0, 1)
	switch p.ResourceType {
	case "", "VEHICLE", "ROOM":
		f.ResourceType = p.ResourceType
	default:
		return f, badHistory("jenis resource tidak dikenal")
	}
	switch p.BookingType {
	case "", "SPD", "NON_SPD":
		f.BookingType = p.BookingType
	default:
		return f, badHistory("tipe booking tidak dikenal")
	}
	for _, s := range p.Statuses {
		s = strings.TrimSpace(strings.ToUpper(s))
		if s == "" {
			continue
		}
		if _, ok := historyStatusLabel[s]; !ok {
			return f, badHistory("status tidak dikenal: " + s)
		}
		f.Statuses = append(f.Statuses, s)
	}
	f.Search = strings.TrimSpace(p.Search)
	f.SortBy, f.SortOrder = p.SortBy, p.SortOrder // tak dikenal → urutan bawaan
	return f, nil
}

// History: daftar riwayat untuk halaman admin.
func (s *BookingService) History(ctx context.Context, f repository.BookingHistoryFilter, page, limit int) ([]map[string]any, int64, error) {
	rows, err := s.q.ListBookingHistory(ctx, f, int32(limit), int32((page-1)*limit))
	if err != nil {
		return nil, 0, err
	}
	total, err := s.q.CountBookingHistory(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = map[string]any{
			"id": r.ID, "status": r.Status, "resourceType": r.ResourceType, "bookingType": r.BookingType,
			"resourceName": r.ResourceName, "plateNumber": r.PlateNumber,
			"requester": map[string]any{"name": r.RequesterName, "department": r.Department},
			"startDate": r.StartDate, "endDate": r.EndDate, "createdAt": r.CreatedAt,
			"driverName": r.DriverName, "roomKeepers": r.RoomKeepers,
			"returnedAt": nullTime(r.ReturnedAt), "completedAt": nullTime(r.CompletedAt),
		}
	}
	return out, total, nil
}

// HistoryExportOptions: bagian yang disertakan + basis URL untuk link.
type HistoryExportOptions struct {
	Sections []string // kosong = semua
	FileBase string   // mis. https://reservasi.kce.co.id → link foto /uploads/...
	WebBase  string   // basis web untuk link detail booking ("" = tanpa link)
	ActorID  int32    // pengekspor (ditulis di sheet Info)
}

// ExportHistory menghasilkan file .xlsx beserta nama filenya.
func (s *BookingService) ExportHistory(ctx context.Context, f repository.BookingHistoryFilter, opt HistoryExportOptions) ([]byte, string, error) {
	total, err := s.q.CountBookingHistory(ctx, f)
	if err != nil {
		return nil, "", err
	}
	if total > historyMaxExportRows {
		return nil, "", badHistory(fmt.Sprintf(
			"terlalu banyak booking (%d) untuk diekspor sekaligus - persempit periode atau filter (maks. %d)",
			total, historyMaxExportRows))
	}
	sections := map[string]bool{}
	for _, k := range opt.Sections {
		if _, ok := historySectionLabel[k]; !ok {
			return nil, "", badHistory("bagian ekspor tidak dikenal: " + k)
		}
		sections[k] = true
	}
	if len(opt.Sections) == 0 {
		for _, k := range HistorySections {
			sections[k] = true
		}
	}

	rows, err := s.q.ListBookingHistory(ctx, f, 0, 0)
	if err != nil {
		return nil, "", err
	}
	ids := make([]int32, len(rows))
	resName := map[int32]string{}
	for i, r := range rows {
		ids[i] = r.ID
		name := r.ResourceName
		if r.PlateNumber != "" {
			name += " (" + r.PlateNumber + ")"
		}
		resName[r.ID] = name
	}

	x := newHistoryWorkbook(opt.FileBase)
	defer x.f.Close()

	// ── Info ──
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.Status]++
	}
	included := []string{"Booking"}
	for _, k := range HistorySections {
		if sections[k] {
			included = append(included, historySectionLabel[k])
		}
	}
	info := [][]any{
		{"Riwayat Booking"},
		{"Periode", f.From.In(util.WIB).Format("02/01/2006") + " - " + f.To.AddDate(0, 0, -1).In(util.WIB).Format("02/01/2006")},
		{"Jenis resource", map[string]string{"": "Semua", "VEHICLE": "Kendaraan", "ROOM": "Ruangan"}[f.ResourceType]},
		{"Tipe booking", map[string]string{"": "Semua", "SPD": "SPD", "NON_SPD": "Non-SPD"}[f.BookingType]},
		{"Status", historyStatusList(f.Statuses)},
		{"Kata kunci", dash(f.Search)},
		{"Bagian", strings.Join(included, ", ")},
		{"Diekspor", time.Now().In(util.WIB).Format("02/01/2006 15:04") + " WIB"},
		{"Diekspor oleh", dash(s.userName(ctx, opt.ActorID))},
		{"Jumlah booking", len(rows)},
		{},
		{"Status", "Jumlah"},
	}
	for _, st := range historyStatusOrder {
		if counts[st] > 0 {
			info = append(info, []any{historyStatusLabel[st], counts[st]})
		}
	}
	x.info(info)

	// ── Booking ──
	bookingRows := make([][]any, len(rows))
	for i, r := range rows {
		bType := "-"
		if r.ResourceType == "VEHICLE" {
			bType = map[string]string{"SPD": "SPD", "NON_SPD": "Non-SPD"}[r.BookingType]
		}
		detail := ""
		if opt.WebBase != "" {
			detail = strings.TrimRight(opt.WebBase, "/") + "/booking/" + itoa(r.ID)
		}
		bookingRows[i] = []any{
			r.ID, historyStatusLabel[r.Status], map[string]string{"VEHICLE": "Kendaraan", "ROOM": "Ruangan"}[r.ResourceType],
			bType, r.ResourceName, r.PlateNumber, r.RoomLocation,
			r.RequesterName, r.EmployeeID, r.Department, r.RequesterPhone, r.Purpose, r.PassengerCount,
			r.PickupLocation, r.Destination, r.StartDate, r.EndDate, r.CreatedAt,
			r.ApprovedBy, r.ApprovedAt, r.RejectNote, r.DriverName, r.RoomKeepers, r.OriginalResource,
			r.StartedAt, r.StartedBy, r.ReturnedAt, r.CompletedAt, r.OvertimeMinutes,
			r.TollBudget, r.ParkingAllowance, historyLink{detail},
		}
	}
	x.sheet("Booking", []string{"No. Booking", "Status", "Jenis", "Tipe", "Resource", "No. Polisi", "Lokasi Ruangan",
		"Pemohon", "NIP", "Departemen", "No. HP Pemohon", "Keperluan", "Jumlah Penumpang",
		"Lokasi Jemput", "Tujuan", "Jadwal Mulai", "Jadwal Selesai", "Dibuat",
		"Disetujui Oleh", "Disetujui Pada", "Catatan Penolakan", "Supir", "Penjaga Ruangan", "Dialihkan Dari",
		"Dimulai Pada", "Dimulai Oleh", "Kembali Pada", "Selesai Pada", "Overtime (menit)",
		"Anggaran Tol", "Bekal Parkir", "Link Detail"}, bookingRows)

	// ── Bagian per booking ──
	withRes := func(recs []repository.HistoryRecord, conv func([]any) []any) [][]any {
		out := make([][]any, len(recs))
		for i, rc := range recs {
			out[i] = append([]any{rc.BookingID, resName[rc.BookingID]}, conv(rc.Values)...)
		}
		return out
	}
	same := func(v []any) []any { return v }
	head := func(cols ...string) []string { return append([]string{"No. Booking", "Resource"}, cols...) }

	if sections["passengers"] {
		recs, err := s.q.HistoryPassengers(ctx, ids)
		if err != nil {
			return nil, "", err
		}
		x.sheet("Penumpang", head("Nama", "Jenis", "NIP", "Departemen", "No. HP", "Keterangan"), withRes(recs, same))
	}
	if sections["departure"] {
		var dep [][]any
		for _, r := range rows {
			if !r.StartedAt.Valid && !r.OdometerStart.Valid && r.StartPhotoURL == "" {
				continue
			}
			dep = append(dep, []any{r.ID, resName[r.ID], r.StartedAt, r.StartedBy, r.OdometerStart,
				r.StartLocation, historyLink{x.fileURL(r.StartPhotoURL)}})
		}
		x.sheet("Keberangkatan", head("Dimulai Pada", "Dimulai Oleh", "Odometer Awal (km)", "Lokasi Mulai", "Foto Keberangkatan"), dep)
	}
	if sections["fuel"] {
		recs, err := s.q.HistoryFuel(ctx, ids)
		if err != nil {
			return nil, "", err
		}
		x.sheet("Pengisian BBM", head("Waktu", "Kendaraan", "Supir", "Bahan Bakar", "Jumlah", "Satuan", "Harga/Satuan",
			"Total Biaya", "Odometer Sebelum", "Odometer Sesudah", "Jarak (km)", "SPBU", "Sumber", "Alasan",
			"Lokasi", "Catatan", "Foto Struk", "Dicatat Oleh", "Dibatalkan Pada", "Alasan Batal"),
			withRes(recs, func(v []any) []any { v[16] = historyLink{x.fileURL(str(v[16]))}; return v }))
	}
	if sections["return"] {
		recs, err := s.q.HistoryReturnReports(ctx, ids)
		if err != nil {
			return nil, "", err
		}
		x.sheet("Laporan Pengembalian", head("Dikirim Pada", "Dikirim Oleh", "Odometer Akhir (km)", "Lokasi",
			"Catatan", "Jumlah Foto (lihat sheet Lampiran & Foto)"), withRes(recs, same))
	}
	if sections["expenses"] {
		recs, err := s.q.HistoryTripExpenses(ctx, ids)
		if err != nil {
			return nil, "", err
		}
		x.sheet("Biaya Tol & Parkir", head("Waktu", "Kategori", "Gerbang Tol", "Gerbang Masuk", "Gerbang Keluar",
			"Nominal", "Cara Bayar", "Catatan", "Foto Bukti", "Dicatat Oleh", "Dibatalkan Pada", "Alasan Batal"),
			withRes(recs, func(v []any) []any {
				if l, ok := historyExpenseCategory[str(v[1])]; ok {
					v[1] = l
				}
				v[8] = historyLink{x.fileURL(str(v[8]))}
				return v
			}))
	}
	if sections["ratings"] {
		recs, err := s.q.HistoryRatings(ctx, ids)
		if err != nil {
			return nil, "", err
		}
		x.sheet("Rating", head("Untuk", "Nama", "Bintang", "Ulasan", "Waktu"), withRes(recs, same))
	}
	if sections["activity"] {
		recs, err := s.q.HistoryActivity(ctx, ids)
		if err != nil {
			return nil, "", err
		}
		x.sheet("Riwayat Aktivitas", head("Waktu", "Aksi", "Keterangan", "Oleh"),
			withRes(recs, func(v []any) []any {
				if l, ok := historyActionLabel[str(v[1])]; ok {
					v[1] = l
				}
				return v
			}))
	}
	if sections["attachments"] {
		recs, err := s.q.HistoryAttachments(ctx, ids)
		if err != nil {
			return nil, "", err
		}
		x.sheet("Lampiran & Foto", head("Diunggah", "Oleh", "Jenis", "Nama File", "Link"),
			withRes(recs, func(v []any) []any {
				kind := "Lampiran"
				if str(v[3]) == "return_photo" {
					kind = "Foto laporan pengembalian"
				} else if d := str(v[3]); d != "" {
					kind = "Lampiran: " + d
				}
				return []any{v[0], v[1], kind, v[2], historyLink{x.fileURL(str(v[4]))}}
			}))
	}

	x.f.SetActiveSheet(0)
	var buf bytes.Buffer
	if err := x.f.Write(&buf); err != nil {
		return nil, "", err
	}
	name := fmt.Sprintf("Riwayat Booking %s sd %s.xlsx",
		f.From.In(util.WIB).Format("2006-01-02"), f.To.AddDate(0, 0, -1).In(util.WIB).Format("2006-01-02"))
	return buf.Bytes(), name, nil
}

func (s *BookingService) userName(ctx context.Context, id int32) string {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return ""
	}
	return u.Name
}

func historyStatusList(sts []string) string {
	if len(sts) == 0 {
		return "Semua"
	}
	labels := make([]string, len(sts))
	for i, s := range sts {
		labels[i] = historyStatusLabel[s]
	}
	sort.Strings(labels)
	return strings.Join(labels, ", ")
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

// ── Penulis workbook ────────────────────────────────────────────────────────

// historyLink: sel berisi URL yang bisa diklik ("" = kosong).
type historyLink struct{ url string }

type historyWorkbook struct {
	f                         *excelize.File
	fileBase                  string
	header, date, link, title int
}

func newHistoryWorkbook(fileBase string) *historyWorkbook {
	f := excelize.NewFile()
	x := &historyWorkbook{f: f, fileBase: strings.TrimRight(fileBase, "/")}
	x.header, _ = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "1A19B5"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8E8FC"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
	})
	fmtDate := "dd/mm/yyyy hh:mm"
	x.date, _ = f.NewStyle(&excelize.Style{CustomNumFmt: &fmtDate})
	x.link, _ = f.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "2D2CE8", Underline: "single"}})
	x.title, _ = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	return x
}

// fileURL: path upload ("/uploads/x", "booking/x", "/files/x") → URL lengkap.
func (x *historyWorkbook) fileURL(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	p = strings.TrimPrefix(p, "/")
	switch {
	case strings.HasPrefix(p, "uploads/"):
	case strings.HasPrefix(p, "files/"):
		p = "uploads/" + strings.TrimPrefix(p, "files/")
	default:
		p = "uploads/" + p
	}
	return x.fileBase + "/" + p
}

func (x *historyWorkbook) info(rows [][]any) {
	const sh = "Info"
	x.f.SetSheetName("Sheet1", sh)
	for i, row := range rows {
		for j, v := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			_ = x.f.SetCellValue(sh, cell, v)
		}
	}
	_ = x.f.SetCellStyle(sh, "A1", "A1", x.title)
	for i, row := range rows {
		if len(row) == 2 && row[0] == "Status" && row[1] == "Jumlah" {
			a, _ := excelize.CoordinatesToCellName(1, i+1)
			b, _ := excelize.CoordinatesToCellName(2, i+1)
			_ = x.f.SetCellStyle(sh, a, b, x.header)
		}
	}
	_ = x.f.SetColWidth(sh, "A", "A", 18)
	_ = x.f.SetColWidth(sh, "B", "B", 60)
}

// sheet menulis satu sheet tabel: header, baris, format tanggal & link,
// lebar kolom, baris header dibekukan + filter otomatis.
func (x *historyWorkbook) sheet(name string, headers []string, rows [][]any) {
	if _, err := x.f.NewSheet(name); err != nil {
		return
	}
	widths := make([]int, len(headers))
	for j, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(j+1, 1)
		_ = x.f.SetCellValue(name, cell, h)
		widths[j] = len(h)
	}
	for i, row := range rows {
		for j, v := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			n := x.setCell(name, cell, v)
			if j < len(widths) && n > widths[j] {
				widths[j] = n
			}
		}
	}
	last, _ := excelize.CoordinatesToCellName(len(headers), 1)
	_ = x.f.SetCellStyle(name, "A1", last, x.header)
	_ = x.f.SetRowHeight(name, 1, 30)
	for j, w := range widths {
		col, _ := excelize.ColumnNumberToName(j + 1)
		width := float64(w) + 2
		if width < 10 {
			width = 10
		}
		if width > 60 {
			width = 60
		}
		_ = x.f.SetColWidth(name, col, col, width)
	}
	_ = x.f.SetPanes(name, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	end, _ := excelize.CoordinatesToCellName(len(headers), len(rows)+1)
	_ = x.f.AutoFilter(name, "A1:"+end, nil)
}

// setCell menulis satu nilai; mengembalikan perkiraan lebar teksnya.
func (x *historyWorkbook) setCell(sh, cell string, v any) int {
	switch t := v.(type) {
	case nil:
		return 0
	case historyLink:
		if t.url == "" {
			return 0
		}
		_ = x.f.SetCellValue(sh, cell, t.url)
		_ = x.f.SetCellHyperLink(sh, cell, t.url, "External")
		_ = x.f.SetCellStyle(sh, cell, cell, x.link)
		return 40
	case time.Time:
		_ = x.f.SetCellValue(sh, cell, t.In(util.WIB))
		_ = x.f.SetCellStyle(sh, cell, cell, x.date)
		return 16
	case sql.NullTime:
		if !t.Valid {
			return 0
		}
		return x.setCell(sh, cell, t.Time)
	case sql.NullInt32:
		if !t.Valid {
			return 0
		}
		v = t.Int32
	case sql.NullFloat64:
		if !t.Valid {
			return 0
		}
		v = t.Float64
	case []byte:
		v = string(t)
	}
	_ = x.f.SetCellValue(sh, cell, v)
	if s, ok := v.(string); ok {
		return len([]rune(s))
	}
	return 10
}
