package wajub

import "context"

type AccountsService struct{ CrudService }

func (s *AccountsService) RegenerateToken(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.post(ctx, "/accounts/"+escapeID(id)+"/token", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "account"), nil
}
