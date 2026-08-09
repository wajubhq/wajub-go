package wajub

import "context"

type InvoicesService struct{ CrudService }

func (s *InvoicesService) Send(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/invoices/"+id+"/send", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "invoice"), nil
}

func (s *InvoicesService) MarkPaid(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/invoices/"+id+"/mark-paid", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "invoice"), nil
}

func (s *InvoicesService) Cancel(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/invoices/"+id+"/cancel", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "invoice"), nil
}
