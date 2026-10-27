package api

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KeKe45/resourcelocate/internal/domain"
)

// records serves every finished hand-over as a CSV file, without names,
// contacts or notes. It is public so judges, funders or a city council can
// check the totals line by line.
func (s *Server) records(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ls, err := s.store.Listings(ctx)
	if err != nil {
		writeErr(w, err)
		return
	}
	ts, err := s.store.Transfers(ctx)
	if err != nil {
		writeErr(w, err)
		return
	}
	name := "resourcelocate-records-" + time.Now().UTC().Format("2006-01-02") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)

	num := func(f float64) string { return strconv.FormatFloat(f, 'f', 2, 64) }
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"record_id", "completed_on", "material", "item", "area", "receiver_type",
		"pieces", "kg", "what_the_receiver_did",
	})
	for _, rec := range domain.Records(ts, byID(ls)) {
		pieces, kg := "", ""
		if rec.Pieces > 0 {
			pieces = strconv.Itoa(rec.Pieces)
		} else {
			kg = num(rec.Kg)
		}
		_ = cw.Write([]string{
			rec.ID, rec.CompletedOn, rec.Material, cell(rec.Item), rec.Area, rec.ReceiverRole,
			pieces, kg, rec.Outcome,
		})
	}
	cw.Flush()
}

// cell keeps user text from being run as a formula when the file is opened
// in a spreadsheet.
func cell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
