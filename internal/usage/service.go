package usage

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type Service struct {
	queries repository.Querier
}

func NewService(queries repository.Querier) *Service {
	return &Service{queries: queries}
}

//nolint:revive
type UsageEventItem struct {
	ID           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	Route        string `json:"route"`
	Method       string `json:"method"`
	RequestCost  string `json:"request_cost"`
	Currency     string `json:"currency"`
	StatusCode   *int32 `json:"status_code,omitempty"`
	LatencyMs    *int64 `json:"latency_ms,omitempty"`
	ResponseSize *int64 `json:"response_size,omitempty"`
	RequestID    string `json:"request_id"`
	ProviderName string `json:"provider_name"`
	ProviderID   string `json:"provider_id"`
	EndpointID   string `json:"endpoint_id"`
	UsageStatus  string `json:"usage_status"`
}

//nolint:revive
type UsageListResponse struct {
	Data       []UsageEventItem `json:"data"`
	NextCursor *string          `json:"next_cursor"`
}

//nolint:revive
type UsageFilter struct {
	ProviderID uuid.UUID
	EndpointID uuid.UUID
	StatusCode int32
	StartDate  time.Time
	EndDate    time.Time
	HasFilters bool
}

type ProviderEarning struct {
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	Total      string `json:"total"`
}

type EndpointEarning struct {
	EndpointID string `json:"endpoint_id"`
	Route      string `json:"route"`
	Total      string `json:"total"`
}

type DailyEarning struct {
	Date   string `json:"date"`
	Amount string `json:"amount"`
}

type EarningsResponse struct {
	TotalEarnings     string            `json:"total_earnings"`
	UnsettledEarnings string            `json:"unsettled_earnings"`
	ByProvider        []ProviderEarning `json:"by_provider"`
	ByEndpoint        []EndpointEarning `json:"by_endpoint"`
	Sparkline         []DailyEarning    `json:"sparkline"`
}

func (s *Service) ListUsageByConsumer(ctx context.Context, consumerID uuid.UUID, filter UsageFilter, cursorTs time.Time, cursorID uuid.UUID, limit int32) (*UsageListResponse, error) {
	if filter.HasFilters {
		rows, err := s.queries.ListUsageEventsByConsumerFiltered(ctx, repository.ListUsageEventsByConsumerFilteredParams{
			ConsumerID: consumerID,
			Column2:    filter.EndpointID,
			Column3:    filter.StatusCode,
			Column4:    filter.StartDate,
			Column5:    filter.EndDate,
			Limit:      limit + 1,
			Column7:    cursorTs,
			Column8:    cursorID,
			Column9:    filter.ProviderID,
		})
		if err != nil {
			return nil, fmt.Errorf("list filtered usage: %w", err)
		}
		return toUsageListFromFilteredConsumer(rows, limit)
	}

	rows, err := s.queries.ListUsageEventsByConsumerCursor(ctx, repository.ListUsageEventsByConsumerCursorParams{
		ConsumerID: consumerID,
		Limit:      limit + 1,
		Column3:    cursorTs,
		Column4:    cursorID,
	})
	if err != nil {
		return nil, fmt.Errorf("list usage: %w", err)
	}
	return toUsageListFromCursorConsumer(rows, limit)
}

func (s *Service) ListUsageByProvider(ctx context.Context, providerID uuid.UUID, filter UsageFilter, cursorTs time.Time, cursorID uuid.UUID, limit int32) (*UsageListResponse, error) {
	if filter.HasFilters {
		rows, err := s.queries.ListUsageEventsByProviderFiltered(ctx, repository.ListUsageEventsByProviderFilteredParams{
			ProviderID: providerID,
			Column2:    filter.EndpointID,
			Column3:    filter.StatusCode,
			Column4:    filter.StartDate,
			Column5:    filter.EndDate,
			Limit:      limit + 1,
			Column7:    cursorTs,
			Column8:    cursorID,
		})
		if err != nil {
			return nil, fmt.Errorf("list filtered provider usage: %w", err)
		}
		return toUsageListFromFilteredProvider(rows, limit)
	}

	rows, err := s.queries.ListUsageEventsByProviderCursor(ctx, repository.ListUsageEventsByProviderCursorParams{
		ProviderID: providerID,
		Limit:      limit + 1,
		Column3:    cursorTs,
		Column4:    cursorID,
	})
	if err != nil {
		return nil, fmt.Errorf("list provider usage: %w", err)
	}
	return toUsageListFromCursorProvider(rows, limit)
}

func toUsageListFromCursorConsumer(rows []repository.ListUsageEventsByConsumerCursorRow, limit int32) (*UsageListResponse, error) {
	count := len(rows)
	hasNext := count > int(limit)
	if hasNext {
		count = int(limit)
	}
	items := make([]UsageEventItem, count)
	for i := range count {
		r := rows[i]
		item, err := rowToItem(r.ID, r.CreatedAt, r.Route, r.Method,
			r.RequestCost, r.Currency, r.StatusCode, r.LatencyMs, r.ResponseSize,
			r.RequestID, r.ProviderID, r.EndpointID, r.ProviderName, r.Status)
		if err != nil {
			return nil, err
		}
		items[i] = item
	}
	return buildResponse(items, hasNext), nil
}

func toUsageListFromCursorProvider(rows []repository.ListUsageEventsByProviderCursorRow, limit int32) (*UsageListResponse, error) {
	count := len(rows)
	hasNext := count > int(limit)
	if hasNext {
		count = int(limit)
	}
	items := make([]UsageEventItem, count)
	for i := range count {
		r := rows[i]
		item, err := rowToItem(r.ID, r.CreatedAt, r.Route, r.Method,
			r.RequestCost, r.Currency, r.StatusCode, r.LatencyMs, r.ResponseSize,
			r.RequestID, r.ProviderID, r.EndpointID, r.ProviderName, r.Status)
		if err != nil {
			return nil, err
		}
		items[i] = item
	}
	return buildResponse(items, hasNext), nil
}

func toUsageListFromFilteredConsumer(rows []repository.ListUsageEventsByConsumerFilteredRow, limit int32) (*UsageListResponse, error) {
	count := len(rows)
	hasNext := count > int(limit)
	if hasNext {
		count = int(limit)
	}
	items := make([]UsageEventItem, count)
	for i := range count {
		r := rows[i]
		item, err := rowToItem(r.ID, r.CreatedAt, r.Route, r.Method,
			r.RequestCost, r.Currency, r.StatusCode, r.LatencyMs, r.ResponseSize,
			r.RequestID, r.ProviderID, r.EndpointID, r.ProviderName, r.Status)
		if err != nil {
			return nil, err
		}
		items[i] = item
	}
	return buildResponse(items, hasNext), nil
}

func toUsageListFromFilteredProvider(rows []repository.ListUsageEventsByProviderFilteredRow, limit int32) (*UsageListResponse, error) {
	count := len(rows)
	hasNext := count > int(limit)
	if hasNext {
		count = int(limit)
	}
	items := make([]UsageEventItem, count)
	for i := range count {
		r := rows[i]
		item, err := rowToItem(r.ID, r.CreatedAt, r.Route, r.Method,
			r.RequestCost, r.Currency, r.StatusCode, r.LatencyMs, r.ResponseSize,
			r.RequestID, r.ProviderID, r.EndpointID, r.ProviderName, r.Status)
		if err != nil {
			return nil, err
		}
		items[i] = item
	}
	return buildResponse(items, hasNext), nil
}

//nolint:revive
func buildResponse(items []UsageEventItem, hasNext bool) *UsageListResponse {
	var nextCursor *string
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		cursor := last.Timestamp + "|" + last.ID
		nextCursor = &cursor
	}
	return &UsageListResponse{
		Data:       items,
		NextCursor: nextCursor,
	}
}

//nolint:revive
func rowToItem(
	id uuid.UUID,
	createdAt time.Time,
	route, method string,
	requestCost pgtype.Numeric,
	currency repository.Currency,
	statusCode pgtype.Int4,
	latencyMs, responseSize pgtype.Int4,
	requestID string,
	providerID, endpointID uuid.UUID,
	providerName string,
	usageStatus repository.UsageStatus,
) (UsageEventItem, error) {
	cost, err := numericToDecimal(requestCost)
	if err != nil {
		return UsageEventItem{}, fmt.Errorf("convert request cost: %w", err)
	}

	var sc *int32
	if statusCode.Valid {
		sc = &statusCode.Int32
	}

	var lat *int64
	if latencyMs.Valid {
		v := int64(latencyMs.Int32)
		lat = &v
	}

	var rs *int64
	if responseSize.Valid {
		v := int64(responseSize.Int32)
		rs = &v
	}

	return UsageEventItem{
		ID:           id.String(),
		Timestamp:    createdAt.UTC().Format(time.RFC3339),
		Route:        route,
		Method:       method,
		RequestCost:  cost.String(),
		Currency:     string(currency),
		StatusCode:   sc,
		LatencyMs:    lat,
		ResponseSize: rs,
		RequestID:    requestID,
		ProviderName: providerName,
		ProviderID:   providerID.String(),
		EndpointID:   endpointID.String(),
		UsageStatus:  string(usageStatus),
	}, nil
}

func (s *Service) GetEarnings(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (*EarningsResponse, error) {
	providers, err := s.queries.ListProvidersByOwner(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	if len(providers) == 0 {
		return &EarningsResponse{
			TotalEarnings:     "0",
			UnsettledEarnings: "0",
			ByProvider:        []ProviderEarning{},
			ByEndpoint:        []EndpointEarning{},
			Sparkline:         []DailyEarning{},
		}, nil
	}

	totalDec := decimal.Zero
	unsettledDec := decimal.Zero
	providerMap := map[string]*ProviderEarning{}
	endpointMap := map[string]decimal.Decimal{}
	endpointRoute := map[string]string{}
	sparklineMap := map[string]decimal.Decimal{}
	hasDateFilter := !startDate.IsZero() || !endDate.IsZero()

	for _, p := range providers {
		providerMap[p.ID.String()] = &ProviderEarning{
			ProviderID: p.ID.String(),
			Name:       p.Name,
			Total:      "0",
		}

		var total pgtype.Numeric
		if hasDateFilter {
			total, err = s.queries.GetTotalEarningsByProviderInRange(ctx, repository.GetTotalEarningsByProviderInRangeParams{
				ProviderID: p.ID,
				Column2:    startDate,
				Column3:    endDate,
			})
		} else {
			total, err = s.queries.GetTotalEarningsByProvider(ctx, p.ID)
		}
		if err != nil {
			return nil, fmt.Errorf("get total earnings: %w", err)
		}
		pTotal := numericToDecimalOrZero(total)
		totalDec = totalDec.Add(pTotal)
		providerMap[p.ID.String()].Total = pTotal.String()

		unsettled, err := s.queries.GetUnsettledEarningsByProvider(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("get unsettled earnings: %w", err)
		}
		unsettledDec = unsettledDec.Add(numericToDecimalOrZero(unsettled))

		var endpointRows []repository.GetEarningsByEndpointRow
		if hasDateFilter {
			inRangeRows, rErr := s.queries.GetEarningsByEndpointInRange(ctx, repository.GetEarningsByEndpointInRangeParams{
				ProviderID: p.ID,
				Column2:    startDate,
				Column3:    endDate,
			})
			if rErr != nil {
				return nil, fmt.Errorf("get earnings by endpoint: %w", rErr)
			}
			endpointRows = make([]repository.GetEarningsByEndpointRow, len(inRangeRows))
			for i, r := range inRangeRows {
				endpointRows[i] = repository.GetEarningsByEndpointRow(r)
			}
		} else {
			endpointRows, err = s.queries.GetEarningsByEndpoint(ctx, p.ID)
		}
		if err != nil {
			return nil, fmt.Errorf("get earnings by endpoint: %w", err)
		}
		for _, r := range endpointRows {
			total, err := numericToDecimal(r.Total)
			if err != nil {
				return nil, fmt.Errorf("convert endpoint earnings: %w", err)
			}
			eid := r.EndpointID.String()
			endpointMap[eid] = endpointMap[eid].Add(total)
			endpointRoute[eid] = r.Route
		}

		var sparklineRows []repository.GetEarningsSparklineRow
		if hasDateFilter {
			inRangeRows, rErr := s.queries.GetEarningsSparklineInRange(ctx, repository.GetEarningsSparklineInRangeParams{
				ProviderID: p.ID,
				Column2:    startDate,
				Column3:    endDate,
			})
			if rErr != nil {
				return nil, fmt.Errorf("get earnings sparkline: %w", rErr)
			}
			sparklineRows = make([]repository.GetEarningsSparklineRow, len(inRangeRows))
			for i, r := range inRangeRows {
				sparklineRows[i] = repository.GetEarningsSparklineRow(r)
			}
		} else {
			sparklineRows, err = s.queries.GetEarningsSparkline(ctx, p.ID)
		}
		if err != nil {
			return nil, fmt.Errorf("get earnings sparkline: %w", err)
		}
		for _, r := range sparklineRows {
			amt, err := numericToDecimal(r.Amount)
			if err != nil {
				return nil, fmt.Errorf("convert sparkline amount: %w", err)
			}
			var dateStr string
			if r.Date.Valid {
				dateStr = r.Date.Time.Format("2006-01-02")
			}
			sparklineMap[dateStr] = sparklineMap[dateStr].Add(amt)
		}
	}

	byProvider := make([]ProviderEarning, 0, len(providerMap))
	for _, pe := range providerMap {
		byProvider = append(byProvider, *pe)
	}

	byEndpoint := make([]EndpointEarning, 0, len(endpointMap))
	for eid, total := range endpointMap {
		byEndpoint = append(byEndpoint, EndpointEarning{
			EndpointID: eid,
			Route:      endpointRoute[eid],
			Total:      total.String(),
		})
	}

	sparkline := make([]DailyEarning, 0, len(sparklineMap))
	for date, amount := range sparklineMap {
		sparkline = append(sparkline, DailyEarning{Date: date, Amount: amount.String()})
	}
	slices.SortFunc(sparkline, func(a, b DailyEarning) int {
		switch {
		case a.Date < b.Date:
			return -1
		case a.Date > b.Date:
			return 1
		default:
			return 0
		}
	})

	return &EarningsResponse{
		TotalEarnings:     totalDec.String(),
		UnsettledEarnings: unsettledDec.String(),
		ByProvider:        byProvider,
		ByEndpoint:        byEndpoint,
		Sparkline:         sparkline,
	}, nil
}

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, nil
	}
	if n.NaN {
		return decimal.Zero, errors.New("numeric is NaN")
	}
	return decimal.NewFromBigInt(n.Int, n.Exp), nil
}

func numericToDecimalOrZero(n pgtype.Numeric) decimal.Decimal {
	d, err := numericToDecimal(n)
	if err != nil {
		return decimal.Zero
	}
	return d
}
