package powerflow

import (
	"fmt"
	"power-twin-backend/internal/model"
)

type N1Analyzer struct{}

func NewN1Analyzer() *N1Analyzer {
	return &N1Analyzer{}
}

func (a *N1Analyzer) Analyze(calculator *PowerFlowCalculator, feeders []model.Feeder, telemetryMap map[string]model.DeviceTelemetry) ([]model.N1Result, error) {
	var results []model.N1Result

	for _, feeder := range feeders {
		modifiedFeeders := make([]model.Feeder, 0, len(feeders)-1)
		for _, f := range feeders {
			if f.ID != feeder.ID {
				modifiedFeeders = append(modifiedFeeders, f)
			}
		}

		tempCalc := NewCalculator(calculator.Substations, modifiedFeeders)
		tempCalc.SetTelemetry(telemetryMap)

		pfResult, err := tempCalc.Solve(50, 1e-6)
		if err != nil {
			results = append(results, model.N1Result{
				FaultBranch:        feeder.ID,
				Overloads:          []string{},
				Safe:               false,
				TransferSuggestion: fmt.Sprintf("支路 %s 故障后潮流计算失败，需人工评估", feeder.ID),
			})
			continue
		}

		var overloads []string
		for _, mf := range modifiedFeeders {
			if power, ok := pfResult.BranchPowers[mf.ID]; ok {
				loadRate := power / (mf.RatedCurrent * 1500) * 100
				if loadRate > 100 {
					overloads = append(overloads, mf.ID)
				}
			}
		}

		safe := len(overloads) == 0
		var suggestion string
		if !safe {
			suggestion = a.GenerateTransferSuggestion(feeder, modifiedFeeders)
		}

		results = append(results, model.N1Result{
			FaultBranch:        feeder.ID,
			Overloads:          overloads,
			Safe:               safe,
			TransferSuggestion: suggestion,
		})
	}

	return results, nil
}

func (a *N1Analyzer) GenerateTransferSuggestion(overloadedFeeder model.Feeder, adjacentFeeders []model.Feeder) string {
	var alternatives []model.Feeder
	for _, f := range adjacentFeeders {
		if f.SourceID == overloadedFeeder.SourceID || f.TargetID == overloadedFeeder.SourceID ||
			f.SourceID == overloadedFeeder.TargetID || f.TargetID == overloadedFeeder.TargetID {
			if f.RatedCurrent > overloadedFeeder.RatedCurrent*0.8 {
				alternatives = append(alternatives, f)
			}
		}
	}

	if len(alternatives) > 0 {
		var names []string
		for _, alt := range alternatives {
			names = append(names, fmt.Sprintf("支路 %s (%s→%s)", alt.ID, alt.SourceID, alt.TargetID))
		}
		return fmt.Sprintf("建议将部分负荷转移至相邻支路: %v，以减轻 %s 故障后的过载", names, overloadedFeeder.ID)
	}
	return fmt.Sprintf("支路 %s 故障后无可用替代路径，建议加强该区段供电能力或限制负荷", overloadedFeeder.ID)
}
