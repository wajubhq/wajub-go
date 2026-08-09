package wajub

import "context"

type BalanceService struct{ base baseClient }

func (s *BalanceService) Retrieve(ctx context.Context) (map[string]any, error) {
	res, err := s.base.get(ctx, "/balance", nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "balance", "data"), nil
}
