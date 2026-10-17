package wajub

import "context"

type CustomersService struct{ CrudService }

func (s *CustomersService) Block(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/customers/"+escapeID(id)+"/block", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "customer"), nil
}

func (s *CustomersService) Unblock(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/customers/"+escapeID(id)+"/unblock", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "customer"), nil
}

func (s *CustomersService) Activate(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/customers/"+escapeID(id)+"/activate", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "customer"), nil
}

func (s *CustomersService) Deactivate(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/customers/"+escapeID(id)+"/deactivate", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "customer"), nil
}

func (s *CustomersService) ListTaxIDs(ctx context.Context, customerID string) ([]map[string]any, map[string]any, error) {
	if err := checkIDs(customerID); err != nil {
		return nil, nil, err
	}
	res, err := s.base.get(ctx, "/customers/"+escapeID(customerID)+"/tax_ids", nil)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "tax_ids")
	return data, meta, nil
}

func (s *CustomersService) CreateTaxID(ctx context.Context, customerID string, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(customerID); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/customers/"+escapeID(customerID)+"/tax_ids", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "tax_id"), nil
}

func (s *CustomersService) DeleteTaxID(ctx context.Context, customerID, taxID string, opts *RequestOptions) error {
	if err := checkIDs(customerID, taxID); err != nil {
		return err
	}
	_, err := s.base.delete(ctx, "/customers/"+escapeID(customerID)+"/tax_ids/"+escapeID(taxID), opts)
	return err
}
