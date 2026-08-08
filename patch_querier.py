import os

def patch_file(filepath):
    with open(filepath, 'r') as f:
        code = f.read()
    
    code = code.replace('q *repository.Queries', 'q repository.Querier')
    
    with open(filepath, 'w') as f:
        f.write(code)

patch_file('internal/service/maintenance_service.go')
patch_file('internal/service/fuel_expense_service.go')
print("Patched services to use repository.Querier")
