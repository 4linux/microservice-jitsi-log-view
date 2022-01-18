package iterators

import "microservice-jitsi-log-view/types"

// IterLogs takes a slice of Jitsilog pointers and returns
// an unbuffered chan of *Jitsilog.
func IterLogs(logs types.JitsilogSlice) types.JitsilogIterator {
	ch := make(chan *types.Jitsilog)

	go func() {
		for _, log := range logs {
			ch <- log
		}
		close(ch)
	}()

	return ch
}

// IteratorToSlice turns an iterator back to a slice.
// The iterator is completely consumed.
func IteratorToSlice(sliceToAppendTo types.JitsilogSlice, logs types.JitsilogIterator) types.JitsilogSlice {
	for log := range logs {
		sliceToAppendTo = append(sliceToAppendTo, log)
	}
	return sliceToAppendTo
}

// FilterByAction filters all logs by a certain action,
// "login" or "logout"
func FilterByAction(action string, logs types.JitsilogIterator) types.JitsilogIterator {
	ch := make(chan *types.Jitsilog)

	go func() {
		for log := range logs {
			if log.Action == action {
				ch <- log
			}
		}
		close(ch)
	}()

	return ch
}

//GroupByResult is one result of a GroupBy* function.
// It has a string field and a JitsilogSlice.
type GroupByResult struct {
	Field string
	Logs  types.JitsilogSlice
}

// GroupByField groups logs by a certain field, returned
// by fn when given a single *Jitsilog.
func GroupByField(fn func(*types.Jitsilog) string, logs types.JitsilogIterator) <-chan GroupByResult {
	ch := make(chan GroupByResult)

	go func() {
		entries := make(map[string]types.JitsilogSlice)
		for log := range logs {
			field := fn(log)
			entries[field] = append(entries[field], log)
		}

		for field, logs := range entries {
			ch <- GroupByResult{field, logs}
		}
		close(ch)
	}()

	return ch
}
