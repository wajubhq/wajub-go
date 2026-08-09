package wajub

import "context"

type ListenService struct{ base baseClient }

func (s *ListenService) Config(ctx context.Context) (map[string]any, error) {
	res, err := s.base.get(ctx, "/listen/config", nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "config", "listen"), nil
}

func (s *ListenService) Auth(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/listen/auth", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "auth", "listen"), nil
}
