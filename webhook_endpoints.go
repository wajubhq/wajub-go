package wajub

import "context"

type WebhookEndpointsService struct{ CrudService }

func (s *WebhookEndpointsService) RotateSecret(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/webhooks/"+id+"/rotate-secret", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "endpoint"), nil
}
