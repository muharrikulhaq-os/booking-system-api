package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"booking-system-api/internal/config"
	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Dokumen PDF maintenance (docs/RANCANGAN_MAINTENANCE_VENDOR.md §6). Dibuat
// ulang dari data setiap kali diunduh — tidak disimpan — supaya isinya selalu
// mutakhir. Kop surat & penandatangan dari Pengaturan Dokumen.

const (
	PDFRequest  = "request"  // Surat Pengajuan Maintenance
	PDFHandover = "handover" // Berita Acara Serah Terima (ke vendor)
	PDFReturn   = "return"   // Berita Acara Pengembalian (dari vendor)
)

var (
	idDays   = []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	idMonths = []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli",
		"Agustus", "September", "Oktober", "November", "Desember"}
)

func idDate(t time.Time) string {
	w := t.In(util.WIB)
	return fmt.Sprintf("%d %s %d", w.Day(), idMonths[int(w.Month())], w.Year())
}

func idDateTime(t time.Time) string {
	return idDate(t) + ", pukul " + t.In(util.WIB).Format("15.04") + " WIB"
}

func rupiah(v any) string {
	f, ok := v.(float64)
	if !ok {
		return "-"
	}
	s := strconv.FormatInt(int64(f+0.5), 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "Rp " + b.String()
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

// GeneratePDF membuat salah satu dokumen maintenance. Nama file disarankan
// ikut dikembalikan untuk header Content-Disposition.
func (s *MaintenanceService) GeneratePDF(ctx context.Context, id int32, kind string) ([]byte, string, error) {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return nil, "", util.ErrNotFound
	}
	if !m.RequestNo.Valid {
		return nil, "", util.NewError(409, "surat belum dibuat - ajukan maintenance terlebih dahulu", util.ErrConflict)
	}
	switch kind {
	case PDFHandover:
		if !m.HandoverAt.Valid {
			return nil, "", util.NewError(409, "berita acara serah terima tersedia setelah kendaraan diserahkan ke vendor", util.ErrConflict)
		}
	case PDFReturn:
		if !m.ReturnedAt.Valid {
			return nil, "", util.NewError(409, "berita acara pengembalian tersedia setelah kendaraan kembali", util.ErrConflict)
		}
	case PDFRequest:
	default:
		return nil, "", util.ErrNotFound
	}
	settings, _ := s.q.GetDocumentSettings(ctx)
	d := newLetter(settings)
	switch kind {
	case PDFRequest:
		d.requestLetter(m, settings)
	case PDFHandover:
		d.handoverReport(m, settings, false)
	case PDFReturn:
		d.handoverReport(m, settings, true)
	}
	var buf bytes.Buffer
	if err := d.pdf.Output(&buf); err != nil {
		return nil, "", err
	}
	name := strings.NewReplacer("/", "-", " ", "_").Replace(m.RequestNo.String)
	prefix := map[string]string{PDFRequest: "Surat_Pengajuan", PDFHandover: "BA_Serah_Terima", PDFReturn: "BA_Pengembalian"}[kind]
	return buf.Bytes(), fmt.Sprintf("%s_%s_%s.pdf", prefix, name, strings.ReplaceAll(m.PlateNumber, " ", "")), nil
}

// ─── Kerangka surat ──────────────────────────────────────────────────────────

type letter struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

const (
	lm       = 20.0 // margin kiri
	rm       = 20.0 // margin kanan
	contentW = 210.0 - lm - rm
)

func newLetter(st repository.DocumentSettings) *letter {
	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(lm, 12, rm)
	p.SetAutoPageBreak(true, 18)
	d := &letter{pdf: p, tr: p.UnicodeTranslatorFromDescriptor("")}
	p.SetFooterFunc(func() {
		p.SetY(-12)
		p.SetFont("Helvetica", "I", 7.5)
		p.SetTextColor(130, 130, 130)
		p.CellFormat(0, 4, d.tr(fmt.Sprintf("Dicetak dari Reservasi KCE - %s - halaman %d",
			time.Now().In(util.WIB).Format("02/01/2006 15.04"), p.PageNo())), "", 0, "C", false, 0, "")
		p.SetTextColor(0, 0, 0)
	})
	p.AddPage()
	d.header(st)
	return d
}

// header: kop surat — logo (bila ada) + nama perusahaan + alamat, garis ganda.
func (d *letter) header(st repository.DocumentSettings) {
	p := d.pdf
	top := p.GetY()
	textX := lm
	logoH := 0.0
	if st.LogoURL.Valid && st.LogoURL.String != "" {
		rel := strings.TrimPrefix(strings.TrimPrefix(st.LogoURL.String, "/uploads/"), "/")
		abs := filepath.Join(config.C.UploadDir, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); err == nil {
			ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(abs)), ".")
			if ext == "jpeg" {
				ext = "jpg"
			}
			opt := fpdf.ImageOptions{ImageType: ext, ReadDpi: true}
			info := p.RegisterImageOptions(abs, opt)
			if p.Err() || info == nil || info.Height() <= 0 {
				p.ClearError() // logo rusak/format tak didukung → kop tanpa logo
			} else {
				// Muat dalam kotak maks. 55 × 18 mm, rasio aspek dipertahankan
				// (logo perusahaan sering melebar).
				w, h := 55.0, 55.0*info.Height()/info.Width()
				if h > 18 {
					h, w = 18, 18*info.Width()/info.Height()
				}
				p.ImageOptions(abs, lm, top, w, h, false, opt, 0, "")
				textX, logoH = lm+w+6, h
			}
		}
	}
	name := st.CompanyName
	if strings.TrimSpace(name) == "" {
		name = "(Nama perusahaan - atur di Pengaturan Dokumen)"
	}
	p.SetXY(textX, top+1)
	p.SetFont("Helvetica", "B", 15)
	p.MultiCell(contentW-(textX-lm), 7, d.tr(strings.ToUpper(name)), "", "L", false)
	p.SetFont("Helvetica", "", 9)
	var lines []string
	if st.CompanyAddress != "" {
		lines = append(lines, st.CompanyAddress)
	}
	var contact []string
	if st.CompanyPhone != "" {
		contact = append(contact, "Telp. "+st.CompanyPhone)
	}
	if st.CompanyEmail != "" {
		contact = append(contact, "Email: "+st.CompanyEmail)
	}
	if len(contact) > 0 {
		lines = append(lines, strings.Join(contact, "   |   "))
	}
	for _, l := range lines {
		p.SetX(textX)
		p.MultiCell(contentW-(textX-lm), 4.5, d.tr(l), "", "L", false)
	}
	y := p.GetY() + 2
	if y < top+logoH+2 {
		y = top + logoH + 2
	}
	p.SetLineWidth(0.8)
	p.Line(lm, y, 210-rm, y)
	p.SetLineWidth(0.2)
	p.Line(lm, y+1.2, 210-rm, y+1.2)
	p.SetY(y + 6)
}

func (d *letter) title(t, sub string) {
	p := d.pdf
	p.SetFont("Helvetica", "BU", 12.5)
	p.CellFormat(0, 7, d.tr(t), "", 1, "C", false, 0, "")
	if sub != "" {
		p.SetFont("Helvetica", "", 9.5)
		p.CellFormat(0, 5, d.tr(sub), "", 1, "C", false, 0, "")
	}
	p.Ln(3)
}

func (d *letter) para(text string) {
	d.pdf.SetFont("Helvetica", "", 10.5)
	d.pdf.MultiCell(0, 5.2, d.tr(text), "", "J", false)
	d.pdf.Ln(1.5)
}

func (d *letter) section(t string) {
	p := d.pdf
	p.Ln(0.5)
	p.SetFont("Helvetica", "B", 10.5)
	p.CellFormat(0, 6, d.tr(t), "", 1, "L", false, 0, "")
}

// kv: tabel label–isi bergaris tipis.
func (d *letter) kv(rows [][2]string) {
	p := d.pdf
	const labelW = 52.0
	p.SetDrawColor(200, 200, 200)
	for _, r := range rows {
		p.SetFont("Helvetica", "", 10)
		lines := d.split(orDash(r[1]), contentW-labelW-4)
		h := float64(len(lines))*4.8 + 1.6
		if p.GetY()+h > 297-22 {
			p.AddPage()
		}
		y := p.GetY()
		p.SetFillColor(245, 246, 250)
		p.Rect(lm, y, labelW, h, "FD")
		p.Rect(lm+labelW, y, contentW-labelW, h, "D")
		p.SetXY(lm+2, y+0.8)
		p.SetFont("Helvetica", "B", 9.5)
		p.CellFormat(labelW-4, 4.8, d.tr(r[0]), "", 0, "L", false, 0, "")
		p.SetFont("Helvetica", "", 10)
		for i, l := range lines {
			p.SetXY(lm+labelW+2, y+0.8+float64(i)*4.8)
			p.CellFormat(contentW-labelW-4, 4.8, l, "", 0, "L", false, 0, "")
		}
		p.SetY(y + h)
	}
	p.SetDrawColor(0, 0, 0)
	p.Ln(2)
}

// signatures: kolom tanda tangan (1 atau 2 pihak): [peran, organisasi, nama, jabatan].
func (d *letter) signatures(cols [][4]string) {
	p := d.pdf
	if p.GetY()+40 > 297-18 {
		p.AddPage()
	}
	p.Ln(3)
	y := p.GetY()
	w := contentW / float64(len(cols))
	if len(cols) == 1 {
		w = 75
	}
	for i, c := range cols {
		x := lm + float64(i)*w
		if len(cols) == 1 {
			x = 210 - rm - w
		}
		p.SetXY(x, y)
		p.SetFont("Helvetica", "", 10)
		p.CellFormat(w, 5, d.tr(c[0]), "", 2, "C", false, 0, "")
		p.SetFont("Helvetica", "B", 10)
		p.CellFormat(w, 5, d.tr(c[1]), "", 2, "C", false, 0, "")
		p.SetXY(x, y+28)
		name := strings.TrimSpace(c[2])
		if name == "" {
			name = "(.................................)"
			p.SetFont("Helvetica", "", 10)
		} else {
			p.SetFont("Helvetica", "BU", 10)
		}
		p.CellFormat(w, 5, d.tr(name), "", 2, "C", false, 0, "")
		if t := strings.TrimSpace(c[3]); t != "" {
			p.SetFont("Helvetica", "", 9.5)
			p.CellFormat(w, 5, d.tr(t), "", 2, "C", false, 0, "")
		}
	}
	p.SetY(y + 42)
}

// ─── Surat Pengajuan ─────────────────────────────────────────────────────────

func (d *letter) requestLetter(m repository.MaintenanceItem, st repository.DocumentSettings) {
	p := d.pdf
	submitted := m.CreatedAt
	if m.SubmittedAt.Valid {
		submitted = m.SubmittedAt.Time
	}
	// Nomor / Lampiran / Perihal (kiri) + tanggal (kanan).
	p.SetFont("Helvetica", "", 10.5)
	y := p.GetY()
	for i, r := range [][2]string{
		{"Nomor", m.RequestNo.String},
		{"Lampiran", "-"},
		{"Perihal", "Permohonan Maintenance Kendaraan"},
	} {
		p.SetXY(lm, y+float64(i)*5.5)
		p.CellFormat(22, 5.5, d.tr(r[0]), "", 0, "L", false, 0, "")
		p.CellFormat(4, 5.5, ":", "", 0, "L", false, 0, "")
		if r[0] == "Perihal" {
			p.SetFont("Helvetica", "B", 10.5)
		}
		p.CellFormat(90, 5.5, d.tr(r[1]), "", 0, "L", false, 0, "")
		p.SetFont("Helvetica", "", 10.5)
	}
	p.SetXY(lm, y)
	p.CellFormat(contentW, 5.5, d.tr(idDate(submitted)), "", 0, "R", false, 0, "")
	p.SetY(y + 19)

	vendor := orDash(m.VendorName.String)
	p.CellFormat(0, 5.5, d.tr("Kepada Yth."), "", 1, "L", false, 0, "")
	p.SetFont("Helvetica", "B", 10.5)
	p.CellFormat(0, 5.5, d.tr(vendor), "", 1, "L", false, 0, "")
	p.SetFont("Helvetica", "", 10.5)
	if m.VendorPic.Valid && m.VendorPic.String != "" {
		p.CellFormat(0, 5.5, d.tr("Up. "+m.VendorPic.String), "", 1, "L", false, 0, "")
	}
	if m.VendorAddress.Valid && m.VendorAddress.String != "" {
		p.MultiCell(110, 5.5, d.tr(m.VendorAddress.String), "", "L", false)
	}
	p.CellFormat(0, 5.5, d.tr("di tempat"), "", 1, "L", false, 0, "")
	p.Ln(3)

	company := orDash(st.CompanyName)
	d.para("Dengan hormat,")
	d.para(fmt.Sprintf("Sehubungan dengan kebutuhan perawatan kendaraan operasional, bersama surat ini %s "+
		"mengajukan permohonan maintenance untuk kendaraan dengan rincian sebagai berikut:", company))

	ownership := "Milik perusahaan"
	if m.Ownership == "VENDOR" {
		ownership = "Kendaraan sewa - " + orDash(m.OwnerVendorName.String)
		if m.RentalContract.Valid && m.RentalContract.String != "" {
			ownership += " (kontrak " + m.RentalContract.String + ")"
		}
	}
	odo := "-"
	if m.Odometer.Valid {
		odo = km(m.Odometer.Int32)
	} else if m.CurrentOdometer > 0 {
		odo = km(m.CurrentOdometer)
	}
	d.section("A. Data Kendaraan")
	d.kv([][2]string{
		{"Nomor Polisi", m.PlateNumber},
		{"Kendaraan", fmt.Sprintf("%s (%s %s, %d)", m.VehicleName, m.Brand, m.Model, m.Year)},
		{"Odometer", odo},
		{"Status Kepemilikan", ownership},
	})

	cat := maintenanceCategories[m.Category.String]
	d.section("B. Pekerjaan yang Dimohon")
	rows := [][2]string{{"Jenis", orDash(cat)}, {"Uraian Pekerjaan", m.Description}}
	if m.Complaint.Valid && m.Complaint.String != "" {
		rows = append(rows, [2]string{"Keluhan", m.Complaint.String})
	}
	d.kv(rows)

	d.section("C. Rencana Pelaksanaan")
	when := "-"
	if m.ScheduledDate.Valid {
		when = idDateTime(m.ScheduledDate.Time) + " (jadwal terkonfirmasi)"
	} else if m.PlannedDate.Valid {
		when = idDateTime(m.PlannedDate.Time)
	}
	days := "-"
	if m.EstimatedDays.Valid {
		days = fmt.Sprintf("%d hari", m.EstimatedDays.Int32)
	}
	plan := [][2]string{
		{"Rencana Tanggal", when},
		{"Estimasi Lama", days},
		{"Cara Serah", orDash(pickupMethods[m.PickupMethod.String])},
	}
	if m.Location.Valid && m.Location.String != "" {
		plan = append(plan, [2]string{"Lokasi", m.Location.String})
	}
	if m.EstimatedCost.Valid {
		plan = append(plan, [2]string{"Estimasi Biaya", rupiah(numOrNil(m.EstimatedCost))})
	}
	if m.CostBearer.Valid {
		plan = append(plan, [2]string{"Biaya Ditanggung", costBearers[m.CostBearer.String]})
	}
	d.kv(plan)

	d.para("Mohon kiranya pihak Bapak/Ibu dapat mengonfirmasi jadwal pelaksanaan beserta estimasi biaya " +
		"sebelum pekerjaan dimulai. Atas perhatian dan kerja samanya, kami ucapkan terima kasih.")

	d.signatures([][4]string{{"Hormat kami,", company, st.SignerName, st.SignerTitle}})
}

// ─── Berita Acara ────────────────────────────────────────────────────────────

func (d *letter) handoverReport(m repository.MaintenanceItem, st repository.DocumentSettings, isReturn bool) {
	p := d.pdf
	company := orDash(st.CompanyName)
	vendor := orDash(m.VendorName.String)
	at, odo, fuel, list, note := m.HandoverAt, m.HandoverOdo, m.HandoverFuel, m.HandoverList, m.HandoverNote
	title, suffix := "BERITA ACARA SERAH TERIMA KENDARAAN", "BAST"
	if isReturn {
		at, odo, fuel, list, note = m.ReturnedAt, m.ReturnOdo, m.ReturnFuel, m.ReturnList, m.ReturnNote
		title, suffix = "BERITA ACARA PENGEMBALIAN KENDARAAN", "BAPK"
	}
	d.title(title, "Nomor: "+m.RequestNo.String+"-"+suffix)

	w := at.Time.In(util.WIB)
	if isReturn {
		d.para(fmt.Sprintf("Pada hari ini %s, tanggal %s, pukul %s WIB, %s telah menyerahkan kembali kendaraan "+
			"kepada %s setelah pelaksanaan maintenance berdasarkan surat nomor %s, dengan keterangan sebagai berikut:",
			idDays[int(w.Weekday())], idDate(w), w.Format("15.04"), vendor, company, m.RequestNo.String))
	} else {
		d.para(fmt.Sprintf("Pada hari ini %s, tanggal %s, pukul %s WIB, %s telah menyerahkan kendaraan kepada "+
			"%s untuk keperluan maintenance berdasarkan surat nomor %s, dengan keterangan sebagai berikut:",
			idDays[int(w.Weekday())], idDate(w), w.Format("15.04"), company, vendor, m.RequestNo.String))
	}

	odoStr := "-"
	if odo.Valid {
		odoStr = km(odo.Int32)
	}
	d.section("A. Data Kendaraan")
	d.kv([][2]string{
		{"Nomor Polisi", m.PlateNumber},
		{"Kendaraan", fmt.Sprintf("%s (%s %s, %d)", m.VehicleName, m.Brand, m.Model, m.Year)},
		{"Odometer", odoStr},
		{"Level BBM", orDash(fuel.String)},
	})

	if isReturn {
		d.section("B. Hasil Pekerjaan")
		rows := [][2]string{{"Pekerjaan Dilakukan", m.WorkDone.String}, {"Part Diganti", m.PartsReplaced.String}}
		if m.TotalCost.Valid {
			rows = append(rows, [2]string{"Biaya", rupiah(numOrNil(m.TotalCost))})
		}
		d.kv(rows)
	}

	letter := "B"
	if isReturn {
		letter = "C"
	}
	d.section(letter + ". Kelengkapan Kendaraan")
	checked := jsonChecklist(list)
	const colW = contentW / 2
	p.SetFont("Helvetica", "", 10)
	p.SetDrawColor(200, 200, 200)
	for i, it := range handoverChecklistItems {
		x := lm + float64(i%2)*colW
		if i%2 == 0 && i > 0 {
			p.Ln(6.5)
		}
		y := p.GetY()
		p.Rect(x, y, colW, 6.5, "D")
		mark := "Tidak ada"
		if checked[it.Key] {
			mark = "Ada"
		}
		p.SetXY(x+2, y+0.75)
		p.CellFormat(colW-30, 5, d.tr(it.Label), "", 0, "L", false, 0, "")
		p.SetFont("Helvetica", "B", 10)
		p.CellFormat(26, 5, d.tr(mark), "", 0, "R", false, 0, "")
		p.SetFont("Helvetica", "", 10)
		p.SetY(y)
	}
	p.Ln(10)
	p.SetDrawColor(0, 0, 0)

	if note.Valid && note.String != "" {
		d.section("Catatan Kondisi")
		d.para(note.String)
	}
	d.para("Demikian berita acara ini dibuat dengan sebenarnya untuk dipergunakan sebagaimana mestinya.")

	if isReturn {
		d.signatures([][4]string{
			{"Yang Menyerahkan,", vendor, m.ReturnBy.String, "Pihak Vendor"},
			{"Yang Menerima,", company, "", "Pihak Perusahaan"},
		})
	} else {
		d.signatures([][4]string{
			{"Yang Menyerahkan,", company, "", "Pihak Perusahaan"},
			{"Yang Menerima,", vendor, m.HandoverBy.String, "Pihak Vendor"},
		})
	}
}

// split memecah teks per baris sesuai lebar. SplitText mengukur per rune
// dengan tabel lebar 256 karakter font inti, jadi teks cp1252 hasil tr()
// dibentuk ulang menjadi rune 0–255 (satu byte = satu rune), lalu tiap
// baris dikembalikan ke byte cp1252 untuk dicetak.
func (d *letter) split(text string, w float64) []string {
	b := []byte(d.tr(text))
	rs := make([]rune, len(b))
	for i, c := range b {
		rs[i] = rune(c)
	}
	lines := d.pdf.SplitText(string(rs), w)
	out := make([]string, len(lines))
	for i, l := range lines {
		lb := make([]byte, 0, len(l))
		for _, r := range l {
			lb = append(lb, byte(r))
		}
		out[i] = string(lb)
	}
	return out
}

// km: odometer dengan pemisah ribuan, mis. 45.210 km.
func km(v int32) string {
	return strings.TrimPrefix(rupiah(float64(v)), "Rp ") + " km"
}
