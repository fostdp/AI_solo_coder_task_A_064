package reliability_analyzer

import (
	"power-twin-backend/internal/config"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/powerflow"
	"power-twin-backend/internal/powerflow_engine"
	"power-twin-backend/internal/repository"
	"time"
)

type N1AnalysisMsg struct {
	N1Results []model.N1Result
	Timestamp time.Time
}

type Analyzer struct {
	FlowResultIn chan powerflow_engine.PowerFlowResultMsg
	AnalysisOut  chan N1AnalysisMsg
	sqliteRepo   *repository.SQLiteRepo
	config       *config.N1Config
}

func NewAnalyzer(flowResultIn chan powerflow_engine.PowerFlowResultMsg, sqliteRepo *repository.SQLiteRepo, cfg *config.N1Config) *Analyzer {
	return &Analyzer{
		FlowResultIn: flowResultIn,
		AnalysisOut:  make(chan N1AnalysisMsg, 64),
		sqliteRepo:   sqliteRepo,
		config:       cfg,
	}
}

func (a *Analyzer) Start() {
	go func() {
		for msg := range a.FlowResultIn {
			n1Results, err := a.RunN1Analysis()
			if err != nil {
				continue
			}
			analysisMsg := N1AnalysisMsg{
				N1Results: n1Results,
				Timestamp: time.Now(),
			}
			select {
			case a.AnalysisOut <- analysisMsg:
			default:
			}
		}
	}()
}

func (a *Analyzer) RunN1Analysis() ([]model.N1Result, error) {
	subs, err := a.sqliteRepo.GetSubstations()
	if err != nil {
		return nil, err
	}
	feeders, err := a.sqliteRepo.GetFeeders()
	if err != nil {
		return nil, err
	}

	calc := powerflow.NewCalculator(subs, feeders)
	telemetryMap := make(map[string]model.DeviceTelemetry)
	analyzer := powerflow.NewN1Analyzer()

	n1Results, err := analyzer.Analyze(calc, feeders, telemetryMap, a.config.FeederCapacityRatio)
	if err != nil {
		return nil, err
	}

	return n1Results, nil
}
