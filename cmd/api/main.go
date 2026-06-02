package main

import (
	"log"

	"booking-system-api/internal/config"
	httph "booking-system-api/internal/delivery/http"
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/repository"
	"booking-system-api/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlog "github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	config.Load()
	repository.Connect(config.C.DatabaseURL)
	defer repository.DB.Close()

	db := repository.DB

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    int(config.C.MaxFileSizeMB) * 1024 * 1024,
	})

	app.Use(fiberlog.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
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

	// serve uploaded files (auth-protected via token query param or rely on obscurity of paths)
	app.Get("/files/*", middleware.Auth(), httph.ServeFile(config.C.UploadDir))

	// init services
	authSvc := service.NewAuthService(db)
	userSvc := service.NewUserService(db)
	vehicleSvc := service.NewVehicleService(db)
	roomSvc := service.NewRoomService(db)
	driverSvc := service.NewDriverService(db)
	bookingSvc := service.NewBookingService(db)
	fuelSvc := service.NewFuelExpenseService(db)
	maintSvc := service.NewMaintenanceService(db)
	attachSvc := service.NewAttachmentService(db)
	guestSvc := service.NewGuestBookingService(db)
	settingSvc := service.NewMasterSettingService(db)
	reportSvc := service.NewReportService(db)
	dashboardSvc := service.NewDashboardService(db)

	// register routes
	v1 := app.Group("/api/v1")
	httph.NewAuthHandler(authSvc).Register(v1)
	httph.NewUserHandler(userSvc, attachSvc).Register(v1)
	httph.NewDashboardHandler(dashboardSvc).Register(v1)
	httph.NewVehicleHandler(vehicleSvc, attachSvc).Register(v1)
	httph.NewRoomHandler(roomSvc, attachSvc).Register(v1)
	httph.NewDriverHandler(driverSvc).Register(v1)
	httph.NewBookingHandler(bookingSvc, attachSvc).Register(v1)
	httph.NewFuelExpenseHandler(fuelSvc).Register(v1)
	httph.NewMaintenanceHandler(maintSvc).Register(v1)
	httph.NewAttachmentHandler(attachSvc).Register(v1)
	httph.NewGuestBookingHandler(guestSvc).Register(v1)
	httph.NewMasterSettingHandler(settingSvc).Register(v1)
	httph.NewReportHandler(reportSvc).Register(v1)

	log.Printf("server starting on :%s", config.C.AppPort)
	log.Fatal(app.Listen(":" + config.C.AppPort))
}
