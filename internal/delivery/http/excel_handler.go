package http

import (
	"fmt"
	"reflect"

	"booking-system-api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
)

type ExcelHandler struct {
	svc *service.ReportService
}

func NewExcelHandler(svc *service.ReportService) *ExcelHandler {
	return &ExcelHandler{svc: svc}
}

func (h *ExcelHandler) Register(r fiber.Router) {
	r.Get("/reports/export/excel", h.ExportExcel)
}

func (h *ExcelHandler) ExportExcel(c *fiber.Ctx) error {
	f := excelize.NewFile()
	defer f.Close()

	// Generic function to append sheet and return number of rows written
	appendSheet := func(sheetName string, data any) int {
		f.NewSheet(sheetName)
		v := reflect.ValueOf(data)
		numRows := 0
		if v.Kind() == reflect.Slice && v.Len() > 0 {
			first := v.Index(0)
			if first.Kind() == reflect.Map {
				keys := first.MapKeys()
				for i, k := range keys {
					cell, _ := excelize.CoordinatesToCellName(i+1, 1)
					f.SetCellValue(sheetName, cell, k.String())
				}
				for r := 0; r < v.Len(); r++ {
					row := v.Index(r)
					for i, k := range keys {
						cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
						f.SetCellValue(sheetName, cell, row.MapIndex(k).Interface())
					}
					numRows++
				}
			} else if first.Kind() == reflect.Struct {
				for i := 0; i < first.NumField(); i++ {
					cell, _ := excelize.CoordinatesToCellName(i+1, 1)
					f.SetCellValue(sheetName, cell, first.Type().Field(i).Name)
				}
				for r := 0; r < v.Len(); r++ {
					row := v.Index(r)
					for c := 0; c < row.NumField(); c++ {
						cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
						f.SetCellValue(sheetName, cell, row.Field(c).Interface())
					}
					numRows++
				}
			}
		} else if v.Kind() == reflect.Map {
			f.SetCellValue(sheetName, "A1", "Key")
			f.SetCellValue(sheetName, "B1", "Value")
			keys := v.MapKeys()
			for r, k := range keys {
				cellKey, _ := excelize.CoordinatesToCellName(1, r+2)
				cellVal, _ := excelize.CoordinatesToCellName(2, r+2)
				f.SetCellValue(sheetName, cellKey, k.String())
				f.SetCellValue(sheetName, cellVal, v.MapIndex(k).Interface())
				numRows++
			}
		}
		
		// Auto fit columns (simple logic)
		f.SetColWidth(sheetName, "A", "H", 20)
		
		return numRows
	}

	ctx := c.Context()

	// Helper to add simple line chart if data exists
	addLineChart := func(sheetName string, numRows int, title string, xCol string, yCol string, chartPos string) {
		if numRows > 0 {
			f.AddChart(sheetName, chartPos, &excelize.Chart{
				Type: excelize.Line,
				Series: []excelize.ChartSeries{
					{
						Name:       fmt.Sprintf("%s!$%s$1", sheetName, yCol),
						Categories: fmt.Sprintf("%s!$%s$2:$%s$%d", sheetName, xCol, xCol, numRows+1),
						Values:     fmt.Sprintf("%s!$%s$2:$%s$%d", sheetName, yCol, yCol, numRows+1),
					},
				},
				Title: []excelize.RichTextRun{{Text: title}},
			})
		}
	}
	
	addPieChart := func(sheetName string, numRows int, title string, xCol string, yCol string, chartPos string) {
		if numRows > 0 {
			f.AddChart(sheetName, chartPos, &excelize.Chart{
				Type: excelize.Pie,
				Series: []excelize.ChartSeries{
					{
						Name:       fmt.Sprintf("%s!$%s$1", sheetName, yCol),
						Categories: fmt.Sprintf("%s!$%s$2:$%s$%d", sheetName, xCol, xCol, numRows+1),
						Values:     fmt.Sprintf("%s!$%s$2:$%s$%d", sheetName, yCol, yCol, numRows+1),
					},
				},
				Title: []excelize.RichTextRun{{Text: title}},
			})
		}
	}

	// 1. Booking Summary
	d1, _ := h.svc.BookingSummary(ctx, nil, nil)
	appendSheet("Booking Summary", d1)

	// 2. Resource Usage
	d2, _ := h.svc.ResourceUsage(ctx)
	appendSheet("Resource Usage", d2)

	// 3. Fuel Expenses
	d3, _ := h.svc.FuelExpenses(ctx)
	appendSheet("Fuel Expenses", d3)

	// 4. Maintenance Cost
	d4, _ := h.svc.MaintenanceCost(ctx)
	appendSheet("Maintenance Cost", d4)

	// 5. Driver Ratings
	d5, _ := h.svc.DriverRatings(ctx)
	appendSheet("Driver Ratings", d5)

	// 6. Driver Activity
	d6, _ := h.svc.DriverActivity(ctx)
	appendSheet("Driver Activity", d6)

	// 7. Overdue Bookings
	d7, _ := h.svc.OverdueBookings(ctx)
	appendSheet("Overdue Bookings", d7)

	// 8. Overview
	d8, _ := h.svc.Overview(ctx, "monthly")
	appendSheet("Overview", d8)

	// 9. Booking Trend
	d9, _ := h.svc.BookingTrend(ctx, "monthly", 12)
	nr9 := appendSheet("Booking Trend", d9)
	addLineChart("Booking Trend", nr9, "Trend Booking (12 Bulan)", "A", "B", "E2")

	// 10. Bookings by Department
	d10, _ := h.svc.BookingsByDepartment(ctx, nil, nil)
	nr10 := appendSheet("Bookings Dept", d10)
	addPieChart("Bookings Dept", nr10, "Distribusi Booking per Departemen", "A", "B", "E2")

	// 11. Bookings by Resource
	d11, _ := h.svc.BookingsByResource(ctx, nil, nil)
	nr11 := appendSheet("Bookings Resource", d11)
	addPieChart("Bookings Resource", nr11, "Distribusi Booking per Resource", "A", "B", "E2")

	// 12. Approval Performance
	d12, _ := h.svc.ApprovalPerformance(ctx, nil, nil)
	appendSheet("Approval Perf", d12)

	// 13. Cost Summary
	d13, _ := h.svc.CostSummary(ctx, nil, nil)
	appendSheet("Cost Summary", d13)

	// 14. Cost by Vehicle
	d14, _ := h.svc.CostByVehicle(ctx, nil, nil)
	appendSheet("Cost by Vehicle", d14)

	// 15. Cost by Department
	d15, _ := h.svc.CostByDepartment(ctx, nil, nil)
	nr15 := appendSheet("Cost by Dept", d15)
	addPieChart("Cost by Dept", nr15, "Biaya per Departemen", "A", "B", "E2")

	// 16. Cost Trend
	d16, _ := h.svc.CostTrend(ctx, "monthly", 12)
	nr16 := appendSheet("Cost Trend", d16)
	addLineChart("Cost Trend", nr16, "Trend Biaya (12 Bulan)", "A", "B", "E2")

	// 17. Driver Performance
	d17, _ := h.svc.DriverPerformance(ctx, nil, nil)
	appendSheet("Driver Perf", d17)

	// 18. Department Summary
	d18, _ := h.svc.DepartmentSummary(ctx, nil, nil)
	appendSheet("Dept Summary", d18)

	f.DeleteSheet("Sheet1")

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", "attachment; filename=\"full_report.xlsx\"")

	buf, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	return c.SendStream(buf)
}
