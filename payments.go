package wajub

import (
	"context"
)

type PaymentsService struct{ base baseClient }

func (s *PaymentsService) Create(ctx context.Context, params map[string]any, opts *RequestOptions) (Payment, error) {
	res, err := s.base.post(ctx, "/payments", params, opts)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(res), nil
}

func (s *PaymentsService) Retrieve(ctx context.Context, id string) (Payment, error) {
	res, err := s.base.get(ctx, "/payments/"+escapeID(id), nil)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(map[string]any{"transaction": pickResource(res, "transaction")}), nil
}

func (s *PaymentsService) List(ctx context.Context, params map[string]any) (*PagedResult, error) {
	return createPagedList(ctx, &s.base, "/payments", "transactions", params)
}

func (s *PaymentsService) Cancel(ctx context.Context, id string, opts *RequestOptions) (Payment, error) {
	res, err := s.base.delete(ctx, "/payments/"+escapeID(id), opts)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(map[string]any{"transaction": pickResource(res, "transaction")}), nil
}

func (s *PaymentsService) Process(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (Payment, error) {
	res, err := s.base.post(ctx, "/payments/"+escapeID(id), params, opts)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(map[string]any{"transaction": pickResource(res, "transaction")}), nil
}

func (s *PaymentsService) ProcessSplit(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (Payment, error) {
	res, err := s.base.post(ctx, "/payments/"+escapeID(id)+"/splits", params, opts)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(map[string]any{"transaction": pickResource(res, "transaction")}), nil
}

func (s *PaymentsService) ListRefunds(ctx context.Context, id string, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/payments/"+escapeID(id)+"/refunds", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "refunds")
	return data, meta, nil
}

// Refund refunds the payment id, in full or for the "amount" in params, and
// returns the refund. Pass its "reason" in params. params is not changed.
func (s *PaymentsService) Refund(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (Refund, error) {
	body := make(map[string]any, len(params)+1)
	for key, value := range params {
		body[key] = value
	}
	body["payment"] = id

	res, err := s.base.post(ctx, "/refunds", body, opts)
	if err != nil {
		return Refund{}, err
	}
	return parseRefund(res), nil
}

// RetrieveRefund returns the refund refundID, of any payment.
func (s *PaymentsService) RetrieveRefund(ctx context.Context, refundID string) (Refund, error) {
	res, err := s.base.get(ctx, "/refunds/"+escapeID(refundID), nil)
	if err != nil {
		return Refund{}, err
	}
	return parseRefund(res), nil
}
