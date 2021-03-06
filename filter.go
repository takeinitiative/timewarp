package timewarp

import "time"

// Query is a function that finds the first matching slot in a time range.
type Query func(input TimeRange) (output *TimeRange)

// Filter changes the query into a filter.
func (q Query) Filter() Filter {
	return func(input TimeRange) []*TimeRange {
		var result []*TimeRange

		for input.Duration() > 0 {
			if output := q(input); output != nil {
				result = append(result, output)
				input.Start = output.End
			} else {
				break
			}
		}

		return result
	}
}

// Not turns the query into an inverse filter.
func (q Query) Not() Filter {
	return q.Filter().Negate()
}

// And combines queries to produce a union filter.
func (q Query) And(r ...Query) Filter {
	return q.Filter().And(r...)
}

// In combines queries to produce an intersection filter.
func (q Query) In(r ...Query) Filter {
	return q.Filter().In(r...)
}

// Of is the query implementation of Ordinal
func (q Query) Of(i int, r Query) Filter {
	return q.Filter().Of(i, r)
}

// Filter is a function that returns all matching slots in a time range.
type Filter func(input TimeRange) []*TimeRange

// Negate returns a filter that returns the inverse results
func (f Filter) Negate() Filter {
	return func(input TimeRange) []*TimeRange {
		var result []*TimeRange

		for _, s := range f(input) {
			if input.Start.Before(s.Start) {
				result = append(result, &TimeRange{input.Start, s.Start})
			}
			input.Start = s.End
		}

		if input.Start.Before(input.End) {
			result = append(result, &input)
		}

		return result
	}
}

// Union returns a filter that's result comprises of multiple filters
func (f Filter) Union(filters ...Filter) Filter {
	return func(input TimeRange) []*TimeRange {
		var result = f(input)

		for _, f := range filters {
			result = append(result, f(input)...)
		}

		return result
	}
}

// And is same as Union, but passes a query instead of a filter
func (f Filter) And(queries ...Query) Filter {
	var filters []Filter
	for _, q := range queries {
		filters = append(filters, q.Filter())
	}
	return f.Union(filters...)
}

// Intersect returns a filter that's result must satisfy all filters
func (f Filter) Intersect(filters ...Filter) Filter {
	return func(input TimeRange) []*TimeRange {
		var result = f(input)

		for _, f := range filters {
			var output []*TimeRange

			for _, s := range result {
				output = append(output, f(*s)...)
			}

			result = output
		}

		return result
	}
}

// In is the same as Intersect but passes a query instead of a filter
func (f Filter) In(queries ...Query) Filter {
	var filters []Filter
	for _, q := range queries {
		filters = append(filters, q.Filter())
	}
	return f.Intersect(filters...)
}

// Ordinal returns a filter of ranges within the ordinal range
func (f Filter) Ordinal(order int, filter Filter) Filter {
	if order == 0 {
		panic("ordinal cannot be zero")
	}

	return func(input TimeRange) (result []*TimeRange) {
		for _, v := range filter(input) {
			var r = f(*v)

			// find the range that satisfies the ordinal
			var output = &TimeRange{}
			if size := len(r); order < 0 {
				if -order > size {
					continue
				}
				output = r[order+size]
			} else {
				if order > size {
					continue
				}
				output = r[order-1]
			}

			// continue if the objective value exists, but out of scope
			if !output.Start.Before(input.End) || !output.End.After(input.Start) {
				continue
			}

			// adjust start and end to meet input criteria
			if output.Start.Before(input.Start) {
				output.Start = input.Start
			}
			if output.End.After(input.End) {
				output.End = input.End
			}
			result = append(result, output)
		}
		return
	}
}

// Of is the same as Ordinal, but passes a query instead of a filter
func (f Filter) Of(order int, q Query) Filter {
	return f.Ordinal(order, q.Filter())
}

// Apply calls the filter function
func (f Filter) Apply(start, end time.Time) []*TimeRange {
	return f(TimeRange{start, end})
}

// ApplySeconds calls the filter function for seconds
func (f Filter) ApplySeconds(start, end int64) []*TimeRange {
	return f.Apply(time.Unix(start, 0), time.Unix(end, 0))
}
