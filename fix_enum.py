with open('sql/schema/000001_init.up.sql', 'r') as f:
    sql = f.read()

sql = sql.replace("CREATE TYPE fuel_type AS ENUM", "CREATE TYPE fuel_category AS ENUM")
sql = sql.replace("type fuel_type NOT NULL", "type fuel_category NOT NULL")

with open('sql/schema/000001_init.up.sql', 'w') as f:
    f.write(sql)

with open('sql/query/fuel_expense.sql', 'r') as f:
    q = f.read()

q = q.replace("fuel_type", "fuel_category")
# Need to make sure we don't break sqlc.narg(fuel_type)::fuel_category etc.
# Wait, let's just do it carefully.

with open('sql/query/fuel_expense.sql', 'w') as f:
    f.write(q)
