package wajub

import "context"

type EventsService struct{ base baseClient }

func (s *EventsService) List(ctx context.Context, params map[string]any) (*PagedResult, error) {
	return createPagedList(ctx, &s.base, "/events", "events", params)
}

func (s *EventsService) Retrieve(ctx context.Context, id string) (map[string]any, error) {
	res, err := s.base.get(ctx, "/events/"+escapeID(id), nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "event"), nil
}

func (s *EventsService) Resend(ctx context.Context, id string, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/events/"+escapeID(id)+"/resend", nil, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "event"), nil
}
