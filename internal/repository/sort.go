package repository

import "strings"

// BuildOrderBy resolves a safe "ORDER BY" clause fragment (e.g. `r.name ASC`)
// for a List query, given untrusted sortBy/sortOrder query params.
//
// SQL column/table identifiers can't be parameterized like values ($1, $2,
// ...), so this never lets sortBy touch the query string directly - it's
// only ever used to look up a real SQL expression in `allowed` (a fixed map
// hand-built per table by the service layer). An unrecognized or empty
// sortBy falls back to defaultClause (a complete "<col> <DIR>" string, not
// just a column name) so a typo'd or malicious sortBy degrades to the
// table's normal default order instead of erroring.
func BuildOrderBy(sortBy, sortOrder string, allowed map[string]string, defaultClause string) string {
	col, ok := allowed[sortBy]
	if !ok {
		return defaultClause
	}
	dir := "ASC"
	if strings.EqualFold(sortOrder, "desc") {
		dir = "DESC"
	}
	return col + " " + dir
}
