package main

import (
	"context"
	"log"

	"booking-system-api/internal/config"
	httph "booking-system-api/internal/delivery/http"
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/repository"
	"booking-system-api/internal/service"
	ws "booking-system-api/internal/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlog "github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	config.Load()
	repository.Connect(config.C.DatabaseURL)
	defer repository.DB.Close()

	db := repository.DB

	if err := repository.EnsureReturnReportTable(context.Background(), db); err != nil {
		log.Fatalf("failed to ensure return report table: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    int(config.C.MaxFileSizeMB) * 1024 * 1024,
	})

	app.Use(fiberlog.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: config.C.FrontendOrigin,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))

	// health check
	app.Get("/health", func(c *fiber.Ctx) error {
		if err := repository.DB.Ping(); err != nil {
			return c.Status(503).JSON(fiber.Map{"status": "unhealthy", "db": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// serve uploaded files (auth-protected)
	app.Get("/files/*", httph.ServeFile(config.C.UploadDir))
	// public image serving — FE can embed URLs directly in <img> tags
	app.Get("/uploads/*", httph.ServeFile(config.C.UploadDir))

	// websocket hub
	wsHub := ws.NewHub()
	go wsHub.Run()

	// init services
	authSvc := service.NewAuthService(db)
	userSvc := service.NewUserService(db)
	vehicleSvc := service.NewVehicleService(db)
	roomSvc := service.NewRoomService(db)
	notifSvc := service.NewNotificationService(db, wsHub)
	driverSvc := service.NewDriverService(db)
	bookingSvc := service.NewBookingService(db, notifSvc)
	fuelSvc := service.NewFuelExpenseService(db)
	fuelTypeSvc := service.NewFuelTypeService(db)
	maintSvc := service.NewMaintenanceService(db)
	attachSvc := service.NewAttachmentService(db)
	guestSvc := service.NewGuestBookingService(db)
	settingSvc := service.NewMasterSettingService(db)
	reportSvc := service.NewReportService(db)
	dashboardSvc := service.NewDashboardService(db)

	// register routes
	v1 := app.Group("/api/v1")
	httph.NewWebsocketHandler(wsHub).Register(v1)
	httph.NewAuthHandler(authSvc).Register(v1)
	httph.NewUserHandler(userSvc, attachSvc).Register(v1)
	httph.NewDashboardHandler(dashboardSvc).Register(v1)
	httph.NewVehicleHandler(vehicleSvc, attachSvc).Register(v1)
	httph.NewRoomHandler(roomSvc, attachSvc).Register(v1)
	httph.NewDriverHandler(driverSvc).Register(v1)
	httph.NewBookingHandler(bookingSvc, attachSvc).Register(v1)
	httph.NewFuelExpenseHandler(fuelSvc).Register(v1)
	httph.NewFuelTypeHandler(fuelTypeSvc).Register(v1)
	httph.NewMaintenanceHandler(maintSvc).Register(v1)
	httph.NewAttachmentHandler(attachSvc).Register(v1)
	httph.NewGuestBookingHandler(guestSvc).Register(v1)
	httph.NewMasterSettingHandler(settingSvc).Register(v1)
	httph.NewReportHandler(reportSvc).Register(v1)
	httph.NewExcelHandler(reportSvc).Register(v1)
	httph.NewNotificationHandler(notifSvc).Register(v1)

	log.Printf("server starting on :%s", config.C.AppPort)
	log.Fatal(app.Listen(":" + config.C.AppPort))
}
