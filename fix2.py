import os

with open('internal/service/maintenance_service.go', 'r') as f:
    code = f.read()

# Fix the `_` vs `m` issue
# Let's restore all `_, err := s.q.GetMaintenanceByID` back to `m, err := s.q.GetMaintenanceByID`
code = code.replace(
    '_, err := s.q.GetMaintenanceByID(ctx, id)',
    'm, err := s.q.GetMaintenanceByID(ctx, id)'
)

# Now just specifically for Update:
old_update = """func (s *MaintenanceService) Update(ctx context.Context, id int32, req UpdateMaintenanceRequest) (map[string]any, error) {
	m, err := s.q.GetMaintenanceByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}"""
new_update = """func (s *MaintenanceService) Update(ctx context.Context, id int32, req UpdateMaintenanceRequest) (map[string]any, error) {
	_, err := s.q.GetMaintenanceByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}"""
code = code.replace(old_update, new_update)

with open('internal/service/maintenance_service.go', 'w') as f:
    f.write(code)
print("Restored m correctly")
