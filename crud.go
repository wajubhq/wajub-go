package wajub

import "context"

type CrudService struct {
	base     baseClient
	path     string
	singular string
	plural   string
}

func (s *CrudService) Create(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/"+s.path, params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, s.singular), nil
}

func (s *CrudService) Retrieve(ctx context.Context, id string) (map[string]any, error) {
	res, err := s.base.get(ctx, "/"+s.path+"/"+escapeID(id), nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, s.singular), nil
}

func (s *CrudService) List(ctx context.Context, params map[string]any) (*PagedResult, error) {
	return createPagedList(ctx, &s.base, "/"+s.path, s.plural, params)
}

func (s *CrudService) Update(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.put(ctx, "/"+s.path+"/"+escapeID(id), params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, s.singular), nil
}

func (s *CrudService) Delete(ctx context.Context, id string, opts *RequestOptions) error {
	_, err := s.base.delete(ctx, "/"+s.path+"/"+escapeID(id), opts)
	return err
}
