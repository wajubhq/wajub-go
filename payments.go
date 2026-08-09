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

func (s *PaymentsService) Initialize(ctx context.Context, params map[string]any, opts *RequestOptions) (Payment, error) {
	return s.Create(ctx, params, opts)
}

func (s *PaymentsService) Retrieve(ctx context.Context, id string) (Payment, error) {
	res, err := s.base.get(ctx, "/payments/"+id, nil)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(map[string]any{"transaction": pickResource(res, "transaction")}), nil
}

func (s *PaymentsService) List(ctx context.Context, params map[string]any) (*PagedResult, error) {
	return createPagedList(ctx, &s.base, "/payments", "transactions", params)
}

func (s *PaymentsService) Cancel(ctx context.Context, id string, opts *RequestOptions) (Payment, error) {
	res, err := s.base.delete(ctx, "/payments/"+id, opts)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(map[string]any{"transaction": pickResource(res, "transaction")}), nil
}

func (s *PaymentsService) Process(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (Payment, error) {
	res, err := s.base.post(ctx, "/payments/"+id, params, opts)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(map[string]any{"transaction": pickResource(res, "transaction")}), nil
}

func (s *PaymentsService) ProcessSplit(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (Payment, error) {
	res, err := s.base.post(ctx, "/payments/"+id+"/splits", params, opts)
	if err != nil {
		return Payment{}, err
	}
	return parsePayment(map[string]any{"transaction": pickResource(res, "transaction")}), nil
}

func (s *PaymentsService) ListRefunds(ctx context.Context, id string, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/payments/"+id+"/refunds", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "refunds")
	return data, meta, nil
}
