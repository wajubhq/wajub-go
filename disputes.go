package wajub

import "context"

type DisputesService struct{ base baseClient }

func (s *DisputesService) List(ctx context.Context, params map[string]any) (*PagedResult, error) {
	return createPagedList(ctx, &s.base, "/disputes", "disputes", params)
}

func (s *DisputesService) Retrieve(ctx context.Context, id string) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.get(ctx, "/disputes/"+escapeID(id), nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "dispute"), nil
}

func (s *DisputesService) SubmitEvidence(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/disputes/"+escapeID(id)+"/submit-evidence", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "dispute"), nil
}

func (s *DisputesService) Accept(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/disputes/"+escapeID(id)+"/accept", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "dispute"), nil
}

func (s *DisputesService) Close(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/disputes/"+escapeID(id)+"/close", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "dispute"), nil
}

func (s *DisputesService) SendMessage(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/disputes/"+escapeID(id)+"/messages", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "dispute"), nil
}
