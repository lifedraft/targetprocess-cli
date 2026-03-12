package targetprocess

import "strings"

// QueryParams holds parameters for a v2 API query.
type QueryParams struct {
	Where   string
	Select  string
	OrderBy string
	Take    int
	Skip    int
}

// SearchOption configures a Search request.
type SearchOption func(*searchOpts)

type searchOpts struct {
	selectExpr string
	take       int
	orderBy    string
}

const defaultTake = 25

func resolveSearchOpts(opts []SearchOption) searchOpts {
	so := searchOpts{take: defaultTake}
	for _, o := range opts {
		o(&so)
	}
	return so
}

// WithSelect sets the fields to return in the v2 query select clause.
// Example: WithSelect("id", "name", "entityState.name as state")
func WithSelect(fields ...string) SearchOption {
	return func(o *searchOpts) {
		o.selectExpr = strings.Join(fields, ",")
	}
}

// WithTake sets the maximum number of results to return.
func WithTake(n int) SearchOption {
	return func(o *searchOpts) {
		o.take = n
	}
}

// WithOrderBy sets the sort expression (e.g., "createDate desc").
func WithOrderBy(expr string) SearchOption {
	return func(o *searchOpts) {
		o.orderBy = expr
	}
}
