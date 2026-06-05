package powerflow

import (
	"power-twin-backend/internal/model"
	"math"
	"time"
)

type PowerFlowCalculator struct {
	Substations []model.Substation
	Feeders     []model.Feeder
	Telemetry   map[string]model.DeviceTelemetry
	NodeIndex   map[string]int
	YBus        [][]complex128
	BaseVoltage float64
}

func NewCalculator(substations []model.Substation, feeders []model.Feeder) *PowerFlowCalculator {
	nodeIndex := make(map[string]int)
	for i, s := range substations {
		nodeIndex[s.ID] = i
	}
	return &PowerFlowCalculator{
		Substations: substations,
		Feeders:     feeders,
		Telemetry:   make(map[string]model.DeviceTelemetry),
		NodeIndex:   nodeIndex,
		BaseVoltage: 1500.0,
	}
}

func (c *PowerFlowCalculator) SetTelemetry(telemetryMap map[string]model.DeviceTelemetry) {
	c.Telemetry = telemetryMap
}

func (c *PowerFlowCalculator) BuildAdmittanceMatrix() {
	n := len(c.Substations)
	c.YBus = make([][]complex128, n)
	for i := range c.YBus {
		c.YBus[i] = make([]complex128, n)
	}

	for _, f := range c.Feeders {
		i, oki := c.NodeIndex[f.SourceID]
		j, okj := c.NodeIndex[f.TargetID]
		if !oki || !okj {
			continue
		}
		z := complex(f.ImpedanceR, f.ImpedanceX)
		y := 1.0 / z
		c.YBus[i][j] -= y
		c.YBus[j][i] -= y
		c.YBus[i][i] += y
		c.YBus[j][j] += y
	}
}

func (c *PowerFlowCalculator) Solve(maxIter int, tolerance float64) (*model.PowerFlowResult, error) {
	n := len(c.Substations)
	c.BuildAdmittanceMatrix()

	voltages := make([]float64, n)
	for i := range voltages {
		voltages[i] = 1.0
	}

	powerInjections := make([]float64, n)
	for _, s := range c.Substations {
		idx := c.NodeIndex[s.ID]
		if t, ok := c.Telemetry[s.ID]; ok {
			powerInjections[idx] = -t.Power / (c.BaseVoltage * c.BaseVoltage)
		}
	}

	slackBus := 0
	converged := false
	iterations := 0

	for iter := 0; iter < maxIter; iter++ {
		iterations++
		maxMismatch := 0.0

		mismatches := make([]float64, n)
		for i := 0; i < n; i++ {
			if i == slackBus {
				continue
			}
			pCalc := 0.0
			for j := 0; j < n; j++ {
				g := real(c.YBus[i][j])
				b := imag(c.YBus[i][j])
				theta := 0.0
				pCalc += voltages[i] * voltages[j] * (g*math.Cos(theta) + b*math.Sin(theta))
			}
			mismatches[i] = powerInjections[i] - pCalc
			if math.Abs(mismatches[i]) > maxMismatch {
				maxMismatch = math.Abs(mismatches[i])
			}
		}

		if maxMismatch < tolerance {
			converged = true
			break
		}

		jacobian := make([][]float64, n)
		for i := range jacobian {
			jacobian[i] = make([]float64, n)
		}
		for i := 0; i < n; i++ {
			if i == slackBus {
				jacobian[i][i] = 1.0
				continue
			}
			for j := 0; j < n; j++ {
				if i == j {
					g := real(c.YBus[i][i])
					jacobian[i][i] = 2.0 * voltages[i] * g
				} else if j != slackBus {
					g := real(c.YBus[i][j])
					jacobian[i][j] = voltages[j] * g
				}
			}
		}

		deltaV := solveLinearSystem(jacobian, mismatches, n, slackBus)
		for i := 0; i < n; i++ {
			if i != slackBus {
				voltages[i] += deltaV[i]
				if voltages[i] < 0.5 {
					voltages[i] = 0.5
				}
				if voltages[i] > 1.2 {
					voltages[i] = 1.2
				}
			}
		}
	}

	nodeVoltages := make(map[string]float64)
	for _, s := range c.Substations {
		idx := c.NodeIndex[s.ID]
		nodeVoltages[s.ID] = voltages[idx] * c.BaseVoltage
	}

	branchPowers := c.CalculateBranchPower(voltages)
	losses := c.CalculateLosses(voltages)

	return &model.PowerFlowResult{
		Timestamp:    time.Now(),
		Converged:    converged,
		Iterations:   iterations,
		NodeVoltages: nodeVoltages,
		BranchPowers: branchPowers,
		Losses:       losses,
	}, nil
}

func (c *PowerFlowCalculator) CalculateBranchPower(voltages []float64) map[string]float64 {
	branchPowers := make(map[string]float64)
	for _, f := range c.Feeders {
		i, oki := c.NodeIndex[f.SourceID]
		j, okj := c.NodeIndex[f.TargetID]
		if !oki || !okj {
			continue
		}
		z := complex(f.ImpedanceR, f.ImpedanceX)
		y := 1.0 / z
		vDiff := complex(voltages[i]-voltages[j], 0)
		current := y * vDiff
		power := voltages[i] * real(current) * c.BaseVoltage * c.BaseVoltage
		branchPowers[f.ID] = math.Abs(power)
	}
	return branchPowers
}

func (c *PowerFlowCalculator) CalculateLosses(voltages []float64) float64 {
	totalLoss := 0.0
	for _, f := range c.Feeders {
		i, oki := c.NodeIndex[f.SourceID]
		j, okj := c.NodeIndex[f.TargetID]
		if !oki || !okj {
			continue
		}
		vDiff := voltages[i] - voltages[j]
		if f.ImpedanceR > 0 {
			current := vDiff * c.BaseVoltage / (f.ImpedanceR * c.BaseVoltage)
			totalLoss += current * current * f.ImpedanceR
		}
	}
	return totalLoss * c.BaseVoltage
}

func solveLinearSystem(A [][]float64, b []float64, n int, slackBus int) []float64 {
	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = 0
	}
	if n <= 1 {
		return x
	}

	for i := 0; i < n; i++ {
		if i == slackBus {
			continue
		}
		maxRow := i
		maxVal := math.Abs(A[i][i])
		for k := i + 1; k < n; k++ {
			if math.Abs(A[k][i]) > maxVal {
				maxVal = math.Abs(A[k][i])
				maxRow = k
			}
		}
		if maxVal < 1e-12 {
			continue
		}
		if maxRow != i {
			A[i], A[maxRow] = A[maxRow], A[i]
			b[i], b[maxRow] = b[maxRow], b[i]
		}
		for k := i + 1; k < n; k++ {
			factor := A[k][i] / A[i][i]
			for j := i; j < n; j++ {
				A[k][j] -= factor * A[i][j]
			}
			b[k] -= factor * b[i]
		}
	}

	for i := n - 1; i >= 0; i-- {
		if i == slackBus {
			continue
		}
		if math.Abs(A[i][i]) < 1e-12 {
			continue
		}
		x[i] = b[i]
		for j := i + 1; j < n; j++ {
			x[i] -= A[i][j] * x[j]
		}
		x[i] /= A[i][i]
	}

	return x
}


