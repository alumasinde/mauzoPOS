package catalog

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"mauzo/internal/core/db"
	"mauzo/internal/core/security"
)

// importColumns is the canonical CSV header (order matters).
var importColumns = []string{
	"sku", "name", "kind", "sell_by", "price", "cost", "vat_rate", "deposit",
	"track_stock", "category", "brand", "unit", "barcodes",
}

// ImportResult is a per-row outcome of a CSV import.
type ImportResult struct {
	Created int        `json:"created"`
	Updated int        `json:"updated"`
	Errors  []RowError `json:"errors"`
}

// ImportCSV upserts products from a CSV stream. Category/brand names are
// auto-created when missing; barcodes is a |-separated list. Existing SKUs are
// updated; rows failing validation are collected and skipped, not fatal.
func (s *Service) ImportCSV(ctx context.Context, tenantID uint64, r io.Reader) (ImportResult, error) {
	rd := csv.NewReader(r)
	rd.TrimLeadingSpace = true
	records, err := rd.ReadAll()
	if err != nil {
		return ImportResult{}, ErrImportRow
	}
	return s.importRecords(ctx, tenantID, records)
}

// ImportXLSX upserts products from the first sheet of an .xlsx workbook. The
// header row must use the same column names as the CSV format.
func (s *Service) ImportXLSX(ctx context.Context, tenantID uint64, r io.Reader) (ImportResult, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return ImportResult{}, ErrImportRow
	}
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	if sheet == "" {
		return ImportResult{}, ErrImportRow
	}
	rows, err := f.GetRows(sheet, excelize.Options{RawCellValue: true})
	if err != nil {
		return ImportResult{}, ErrImportRow
	}
	return s.importRecords(ctx, tenantID, rows)
}

// importRecords upserts rows where records[0] is the header. Row numbering in
// errors is 1-based and matches the source file (header = row 1).
func (s *Service) importRecords(ctx context.Context, tenantID uint64, records [][]string) (ImportResult, error) {
	if len(records) == 0 {
		return ImportResult{}, ErrImportRow
	}
	colIdx := map[string]int{}
	for i, h := range records[0] {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	if _, ok := colIdx["sku"]; !ok {
		return ImportResult{}, ErrImportRow
	}
	if _, ok := colIdx["name"]; !ok {
		return ImportResult{}, ErrImportRow
	}
	res := ImportResult{Errors: []RowError{}}
	for i, rec := range records[1:] {
		if emptyRow(rec) {
			continue
		}
		if err := s.importRow(ctx, tenantID, colIdx, rec, &res); err != nil {
			res.Errors = append(res.Errors, RowError{Row: i + 2, Message: err.Error()})
		}
	}
	return res, nil
}

func emptyRow(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func (s *Service) importRow(ctx context.Context, tenantID uint64, col map[string]int, rec []string, res *ImportResult) error {
	get := func(name string) string {
		if i, ok := col[name]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	req := ProductRequest{
		SKU:      get("sku"),
		Name:     get("name"),
		Kind:     or(get("kind"), KindStandard),
		SellBy:   or(get("sell_by"), SellByUnit),
		Price:    or(get("price"), "0"),
		Cost:     get("cost"),
		VatRate:  get("vat_rate"),
		Deposit:  get("deposit"),
		UnitCode: get("unit"),
	}
	if ts := get("track_stock"); ts != "" {
		v, err := strconv.ParseBool(ts)
		if err != nil {
			return fmt.Errorf("track_stock must be true/false")
		}
		req.TrackStock = &v
	}
	if cat := get("category"); cat != "" {
		pub, err := s.ensureCategory(ctx, tenantID, cat)
		if err != nil {
			return fmt.Errorf("category %q: %w", cat, err)
		}
		req.CategoryID = &pub
	}
	if brand := get("brand"); brand != "" {
		pub, err := s.ensureBrand(ctx, tenantID, brand)
		if err != nil {
			return fmt.Errorf("brand %q: %w", brand, err)
		}
		req.BrandID = &pub
	}
	for _, code := range strings.Split(get("barcodes"), "|") {
		if code = strings.TrimSpace(code); code != "" {
			req.Barcodes = append(req.Barcodes, BarcodeRequest{Code: code})
		}
	}
	existing, err := s.repo.BySKU(ctx, tenantID, req.SKU)
	switch {
	case db.IsNotFound(err):
		if _, err := s.Create(ctx, tenantID, req); err != nil {
			return err
		}
		res.Created++
	case err != nil:
		return err
	default:
		if _, err := s.Update(ctx, tenantID, existing.PublicID, req); err != nil {
			return err
		}
		res.Updated++
	}
	return nil
}

// ensureCategory returns the public id of a category named name, creating it.
func (s *Service) ensureCategory(ctx context.Context, tenantID uint64, name string) (string, error) {
	cats, err := s.tax.ListCategories(ctx, tenantID)
	if err != nil {
		return "", err
	}
	for _, c := range cats {
		if strings.EqualFold(c.Name, name) {
			return c.PublicID, nil
		}
	}
	c := Category{TenantID: tenantID, Name: name}
	pub := security.NewPublicID()
	c.PublicID = pub
	if err := s.tax.CreateCategory(ctx, &c); err != nil {
		return "", err
	}
	return pub, nil
}

// ensureBrand returns the public id of a brand named name, creating it.
func (s *Service) ensureBrand(ctx context.Context, tenantID uint64, name string) (string, error) {
	b, err := s.tax.BrandByName(ctx, tenantID, name)
	if err == nil {
		return b.PublicID, nil
	}
	if !db.IsNotFound(err) {
		return "", err
	}
	nb := Brand{TenantID: tenantID, PublicID: security.NewPublicID(), Name: name}
	if err := s.tax.CreateBrand(ctx, &nb); err != nil {
		return "", err
	}
	return nb.PublicID, nil
}

// exportRecords renders the catalog as a table: header row + one row per product.
func (s *Service) exportRecords(ctx context.Context, tenantID uint64) ([][]string, error) {
	rows, err := s.repo.ExportRows(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([][]string, 0, len(rows)+1)
	out = append(out, importColumns)
	for _, r := range rows {
		out = append(out, []string{
			r.SKU, r.Name, r.Kind, r.SellBy, r.Price,
			str(r.Cost), str(r.VatRate), str(r.Deposit),
			strconv.FormatBool(r.TrackStock), str(r.Category), str(r.Brand),
			str(r.Unit), str(r.Barcodes),
		})
	}
	return out, nil
}

// ExportCSV writes the catalog as CSV with the import header — an export can
// be edited and re-imported.
func (s *Service) ExportCSV(ctx context.Context, tenantID uint64, w io.Writer) error {
	records, err := s.exportRecords(ctx, tenantID)
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.WriteAll(records); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

// ExportXLSX writes the catalog as an .xlsx workbook with the same columns.
func (s *Service) ExportXLSX(ctx context.Context, tenantID uint64, w io.Writer) error {
	records, err := s.exportRecords(ctx, tenantID)
	if err != nil {
		return err
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	for i, rec := range records {
		for j, cell := range rec {
			ref, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				return err
			}
			if err := f.SetCellStr(sheet, ref, cell); err != nil {
				return err
			}
		}
	}
	return f.Write(w)
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func or(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
