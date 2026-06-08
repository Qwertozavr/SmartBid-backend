package price

import "context"

type EstimateInput struct {
	Title       string
	Description *string
	Photo       []byte
}

type Estimate struct {
	RecommendedRubles int64
}

type Estimator interface {
	Estimate(ctx context.Context, input EstimateInput) (Estimate, error)
}
