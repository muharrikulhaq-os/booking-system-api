import os

with open('internal/service/maintenance_service.go', 'r') as f:
    code = f.read()

code = code.replace(
    'ResourceID: vehicle.ResourceId',
    'ResourceId: vehicle.ResourceId'
)

code = code.replace(
    'nullNumeric(req.Cost)',
    'nullNumeric(req.TotalCost)'
)

# Add nullNumeric function
if 'func nullNumeric' not in code:
    code = code.replace(
        'func nullInt32Ptr(v *int32) sql.NullInt32 {',
        'func nullNumeric(f *float64) sql.NullString {\n\tif f == nil {\n\t\treturn sql.NullString{}\n\t}\n\treturn sql.NullString{String: fmt.Sprintf("%g", *f), Valid: true}\n}\n\nfunc nullInt32Ptr(v *int32) sql.NullInt32 {'
    )
    # also add fmt to imports if missing, but it might be there.
    if '"fmt"' not in code:
        code = code.replace(
            '"database/sql"',
            '"database/sql"\n\t"fmt"'
        )

# Remove unused m
code = code.replace(
    'm, err := s.q.GetMaintenanceByID(ctx, id)',
    '_, err := s.q.GetMaintenanceByID(ctx, id)'
)
# Wait, m is used in Delete
code = code.replace(
    '_, err := s.q.GetMaintenanceByID(ctx, id)\n\tif err != nil {\n\t\treturn nil, util.ErrNotFound\n\t}\n\t\n\tvehicle, err := s.q.GetVehicleByID(ctx, req.VehicleID)',
    '_, err := s.q.GetMaintenanceByID(ctx, id)\n\tif err != nil {\n\t\treturn nil, util.ErrNotFound\n\t}\n\t\n\tvehicle, err := s.q.GetVehicleByID(ctx, req.VehicleID)'
)

with open('internal/service/maintenance_service.go', 'w') as f:
    f.write(code)
print("Fixed maintenance_service.go")
