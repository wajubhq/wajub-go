package wajub

import "context"

type TaxService struct{ base baseClient }

func (s *TaxService) GetSettings(ctx context.Context) (map[string]any, error) {
	res, err := s.base.get(ctx, "/tax/settings", nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "tax", "settings"), nil
}

func (s *TaxService) UpdateSettings(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.put(ctx, "/tax/settings", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "tax", "settings"), nil
}

func (s *TaxService) Rates(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/tax/rates", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "rates", "tax_rates")
	return data, meta, nil
}

func (s *TaxService) Calculate(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/tax/calculate", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "tax", "calculation"), nil
}

func (s *TaxService) Reports(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/tax/reports", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "reports", "tax_reports")
	return data, meta, nil
}

func (s *TaxService) ListCodes(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/tax/codes", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "tax_codes", "codes")
	return data, meta, nil
}

func (s *TaxService) RetrieveCode(ctx context.Context, code string) (map[string]any, error) {
	if err := checkIDs(code); err != nil {
		return nil, err
	}
	res, err := s.base.get(ctx, "/tax/codes/"+escapeID(code), nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "tax_code", "code"), nil
}

func (s *TaxService) ListRegistrations(ctx context.Context, params map[string]any) (*PagedResult, error) {
	return createPagedList(ctx, &s.base, "/tax/registrations", "registrations", params)
}

func (s *TaxService) CreateRegistration(ctx context.Context, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	res, err := s.base.post(ctx, "/tax/registrations", params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "registration", "tax_registration"), nil
}

func (s *TaxService) RetrieveRegistration(ctx context.Context, id string) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.get(ctx, "/tax/registrations/"+escapeID(id), nil)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "registration", "tax_registration"), nil
}

func (s *TaxService) UpdateRegistration(ctx context.Context, id string, params map[string]any, opts *RequestOptions) (map[string]any, error) {
	if err := checkIDs(id); err != nil {
		return nil, err
	}
	res, err := s.base.put(ctx, "/tax/registrations/"+escapeID(id), params, opts)
	if err != nil {
		return nil, err
	}
	return pickResource(res, "registration", "tax_registration"), nil
}

func (s *TaxService) DeleteRegistration(ctx context.Context, id string, opts *RequestOptions) error {
	if err := checkIDs(id); err != nil {
		return err
	}
	_, err := s.base.delete(ctx, "/tax/registrations/"+escapeID(id), opts)
	return err
}

func (s *TaxService) Jurisdictions(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/tax/jurisdictions", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "jurisdictions", "tax_jurisdictions")
	return data, meta, nil
}

func (s *TaxService) Thresholds(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/tax/thresholds", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "thresholds", "tax_thresholds")
	return data, meta, nil
}

func (s *TaxService) ThresholdAlerts(ctx context.Context, params map[string]any) ([]map[string]any, map[string]any, error) {
	res, err := s.base.get(ctx, "/tax/thresholds/alerts", params)
	if err != nil {
		return nil, nil, err
	}
	data, meta := pickList(res, "alerts", "threshold_alerts")
	return data, meta, nil
}
