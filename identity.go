package wajub

import "context"

type IdentityService struct{ base baseClient }

func (s *IdentityService) Resolve(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/identity/resolve", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "identity"), nil
}

func (s *IdentityService) Validate(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/identity/validate", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "identity"), nil
}
