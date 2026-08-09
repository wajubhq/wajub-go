package wajub

import "context"

type GlobalService struct{ base baseClient }

func (s *GlobalService) Ping(ctx context.Context) (map[string]any, error) {
	return s.base.get(ctx, "/", nil)
}

func (s *GlobalService) Channels(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/channels", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "channels")
	return data, meta, nil
}

func (s *GlobalService) Countries(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/countries", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "countries")
	return data, meta, nil
}

func (s *GlobalService) Currencies(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/currencies", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "currencies")
	return data, meta, nil
}
