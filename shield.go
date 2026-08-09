package wajub

import "context"

type ShieldService struct{ base baseClient }

func (s *ShieldService) GetSettings(ctx context.Context) (map[string]any, error) {
	res, err := s.base.get(ctx, "/shield/settings", nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "shield", "settings"), nil
}

func (s *ShieldService) UpdateSettings(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.put(ctx, "/shield/settings", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "shield", "settings"), nil
}

func (s *ShieldService) Stats(ctx context.Context, params map[string]any) (map[string]any, error) {
	res, err := s.base.get(ctx, "/shield/stats", params)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "shield", "stats"), nil
}

func (s *ShieldService) ListBlocklist(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/shield/blocklist", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "blocklist", "entries")
	return data, meta, nil
}

func (s *ShieldService) AddToBlocklist(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/shield/blocklist", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "blocklist", "entry"), nil
}

func (s *ShieldService) RemoveFromBlocklist(ctx context.Context, id string, opts *RequestOptions) error {
	_, err := s.base.delete(ctx, "/shield/blocklist/"+id, opts)
	return err
}
