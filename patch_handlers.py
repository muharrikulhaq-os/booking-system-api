import os

with open('internal/delivery/http/remaining_handlers.go', 'r') as f:
    handlers = f.read()

# Replace CreateBBM
old_create_bbm = """func (h *FuelExpenseHandler) CreateBBM(c *fiber.Ctx) error {
	var req service.CreateFuelExpenseBBMRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	// get driverId from current user's driver profile
	driverID := queryInt32(c, "driverId")
	if driverID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "driverId is required")
	}
	data, err := h.svc.CreateBBM(c.Context(), req, *driverID)
	if err != nil {
		return err
	}
	return util.Created(c, "BBM expense recorded", data)
}"""

new_create_bbm = """func (h *FuelExpenseHandler) CreateBBM(c *fiber.Ctx) error {
	var req service.CreateFuelExpenseBBMRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	driverID := queryInt32(c, "driverId")
	recordedByID := int32(middleware.GetUserID(c))
	data, err := h.svc.CreateBBM(c.Context(), req, recordedByID, driverID)
	if err != nil {
		return err
	}
	return util.Created(c, "BBM expense recorded", data)
}"""

handlers = handlers.replace(old_create_bbm, new_create_bbm)

# Replace CreateListrik
old_create_listrik = """func (h *FuelExpenseHandler) CreateListrik(c *fiber.Ctx) error {
	var req service.CreateFuelExpenseListrikRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	driverID := queryInt32(c, "driverId")
	if driverID == nil {
		return fiber.NewError(fiber.StatusBadRequest, "driverId is required")
	}
	data, err := h.svc.CreateListrik(c.Context(), req, *driverID)
	if err != nil {
		return err
	}
	return util.Created(c, "Listrik expense recorded", data)
}"""

new_create_listrik = """func (h *FuelExpenseHandler) CreateListrik(c *fiber.Ctx) error {
	var req service.CreateFuelExpenseListrikRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	driverID := queryInt32(c, "driverId")
	recordedByID := int32(middleware.GetUserID(c))
	data, err := h.svc.CreateListrik(c.Context(), req, recordedByID, driverID)
	if err != nil {
		return err
	}
	return util.Created(c, "Listrik expense recorded", data)
}"""

handlers = handlers.replace(old_create_listrik, new_create_listrik)

with open('internal/delivery/http/remaining_handlers.go', 'w') as f:
    f.write(handlers)

print("Patched handlers.")
