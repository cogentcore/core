// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glm

import (
	"fmt"
	"math"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// todo: add tests

// GLM contains results and parameters for running a general
// linear model, which is a general form of multivariate linear
// regression, supporting multiple independent and dependent
// variables.  Make a NewGLM and then do Run() on a tensor
// table.IndexView with the relevant data in columns of the table.
// Batch-mode gradient descent is used and the relevant parameters
// can be altered from defaults before calling Run as needed.
type GLM struct {
	// Coeff are the coefficients to map from input independent variables
	// to the dependent variables.  The first, outer dimension is number of
	// dependent variables, and the second, inner dimension is number of
	// independent variables plus one for the offset (b) (last element).
	Coeff tensor.Float64

	// mean squared error of the fitted values relative to data
	MSE float64

	// R2 is the r^2 total variance accounted for by the linear model,
	// for each dependent variable = 1 - (ErrVariance / ObsVariance)
	R2 []float64

	// Observed variance of each of the dependent variables to be predicted.
	ObsVariance []float64

	// Variance of the error residuals per dependent variables
	ErrVariance []float64

	//	optional names of the independent variables, for reporting results
	IndepNames []string

	//	optional names of the dependent variables, for reporting results
	DepNames []string

	///////////////////////////////////////////
	// Parameters for the GLM model fitting:

	// ZeroOffset restricts the offset of the linear function to 0,
	// forcing it to pass through the origin.  Otherwise, a constant offset "b"
	// is fit during the model fitting process.
	ZeroOffset bool

	// learning rate parameter, which can be adjusted to reduce iterations based on
	// specific properties of the data, but the default is reasonable for most "typical" data.
	LRate float64 `default:"0.1"`

	// tolerance on difference in mean squared error (MSE) across iterations to stop
	// iterating and consider the result to be converged.
	StopTolerance float64 `default:"0.0001"`

	// Constant cost factor subtracted from weights, for the L1 norm or "Lasso"
	// regression.  This is good for producing sparse results but can arbitrarily
	// select one of multiple correlated independent variables.
	L1Cost float64

	// Cost factor proportional to the coefficient value, for the L2 norm or "Ridge"
	// regression.  This is good for generally keeping weights small and equally
	// penalizes correlated independent variables.
	L2Cost float64

	// CostStartIter is the iteration when we start applying the L1, L2 Cost factors.
	// It is often a good idea to have a few unconstrained iterations prior to
	// applying the cost factors.
	CostStartIter int `default:"5"`

	// maximum number of iterations to perform
	MaxIters int `default:"50"`

	///////////////////////////////////////////
	// Cached values from the table

	// Table of data
	Table *table.IndexView

	// tensor columns from table with the respective variables
	IndepVars, DepVars, PredVars, ErrVars tensor.Tensor

	// Number of independent and dependent variables
	NIndepVars, NDepVars int
}

func NewGLM() *GLM {
	glm := &GLM{}
	glm.Defaults()
	return glm
}

func (glm *GLM) Defaults() {
	glm.LRate = 0.1
	glm.StopTolerance = 0.001
	glm.MaxIters = 50
	glm.CostStartIter = 5
}

func (glm *GLM) init(nIv, nDv int) {
	glm.NIndepVars = nIv
	glm.NDepVars = nDv
	glm.Coeff.SetShape([]int{nDv, nIv + 1}, "DepVars", "IndepVars")
	glm.R2 = make([]float64, nDv)
	glm.ObsVariance = make([]float64, nDv)
	glm.ErrVariance = make([]float64, nDv)
	glm.IndepNames = make([]string, nIv)
	glm.DepNames = make([]string, nDv)
}

// SetTable sets the data to use from given indexview of table, where
// each of the Vars args specifies a column in the table, which can have either a
// single scalar value for each row, or a tensor cell with multiple values.
// predVars and errVars (predicted values and error values) are optional.
func (glm *GLM) SetTable(ix *table.IndexView, indepVars, depVars, predVars, errVars string) error {
	dt := ix.Table
	iv, err := dt.ColumnByName(indepVars)
	if err != nil {
		return err
	}
	dv, err := dt.ColumnByName(depVars)
	if err != nil {
		return err
	}
	var pv, ev tensor.Tensor
	if predVars != "" {
		pv, err = dt.ColumnByName(predVars)
		if err != nil {
			return err
		}
	}
	if errVars != "" {
		ev, err = dt.ColumnByName(errVars)
		if err != nil {
			return err
		}
	}
	if pv != nil && !pv.Shape().IsEqual(dv.Shape()) {
		return fmt.Errorf("predVars must have same shape as depVars")
	}
	if ev != nil && !ev.Shape().IsEqual(dv.Shape()) {
		return fmt.Errorf("errVars must have same shape as depVars")
	}
	_, nIv := iv.RowCellSize()
	_, nDv := dv.RowCellSize()
	glm.init(nIv, nDv)
	glm.Table = ix
	glm.IndepVars = iv
	glm.DepVars = dv
	glm.PredVars = pv
	glm.ErrVars = ev
	return nil
}

// Run performs the multi-variate linear regression using data SetTable function,
// learning linear coefficients and an overall static offset that best
// fits the observed dependent variables as a function of the independent variables.
// Initial values of the coefficients, and other parameters for the regression,
// should be set prior to running.
func (glm *GLM) Run() {
	ix := glm.Table
	iv := glm.IndepVars
	dv := glm.DepVars
	pv := glm.PredVars
	ev := glm.ErrVars

	if pv == nil {
		pv = dv.Clone()
	}
	if ev == nil {
		ev = dv.Clone()
	}

	nDv := glm.NDepVars
	nIv := glm.NIndepVars
	nCi := nIv + 1

	dc := glm.Coeff.Clone().(*tensor.Float64)

	lastItr := false
	sse := 0.0
	prevmse := 0.0
	n := ix.Len()
	norm := 1.0 / float64(n)
	lrate := norm * glm.LRate
	for itr := 0; itr < glm.MaxIters; itr++ {
		for i := range dc.Values {
			dc.Values[i] = 0
		}
		sse = 0
		if (itr+1)%10 == 0 {
			lrate *= 0.5
		}
		for i := 0; i < n; i++ {
			row := ix.Indexes[i]
			for di := 0; di < nDv; di++ {
				pred := 0.0
				for ii := 0; ii < nIv; ii++ {
					pred += glm.Coeff.Value([]int{di, ii}) * iv.FloatRowCell(row, ii)
				}
				if !glm.ZeroOffset {
					pred += glm.Coeff.Value([]int{di, nIv})
				}
				targ := dv.FloatRowCell(row, di)
				err := targ - pred
				sse += err * err
				for ii := 0; ii < nIv; ii++ {
					dc.Values[di*nCi+ii] += err * iv.FloatRowCell(row, ii)
				}
				if !glm.ZeroOffset {
					dc.Values[di*nCi+nIv] += err
				}
				if lastItr {
					pv.SetFloatRowCell(row, di, pred)
					if ev != nil {
						ev.SetFloatRowCell(row, di, err)
					}
				}
			}
		}
		for di := 0; di < nDv; di++ {
			for ii := 0; ii <= nIv; ii++ {
				if glm.ZeroOffset && ii == nIv {
					continue
				}
				idx := di*(nCi+1) + ii
				w := glm.Coeff.Values[idx]
				d := dc.Values[idx]
				sgn := 1.0
				if w < 0 {
					sgn = -1.0
				} else if w == 0 {
					sgn = 0
				}
				glm.Coeff.Values[idx] += lrate * (d - glm.L1Cost*sgn - glm.L2Cost*w)
			}
		}
		glm.MSE = norm * sse
		if lastItr {
			break
		}
		if itr > 0 {
			dmse := glm.MSE - prevmse
			if math.Abs(dmse) < glm.StopTolerance || itr == glm.MaxIters-2 {
				lastItr = true
			}
		}
		fmt.Println(itr, glm.MSE)
		prevmse = glm.MSE
	}

	obsMeans := make([]float64, nDv)
	errMeans := make([]float64, nDv)
	for i := 0; i < n; i++ {
		row := ix.Indexes[i]
		for di := 0; di < nDv; di++ {
			obsMeans[di] += dv.FloatRowCell(row, di)
			errMeans[di] += ev.FloatRowCell(row, di)
		}
	}
	for di := 0; di < nDv; di++ {
		obsMeans[di] *= norm
		errMeans[di] *= norm
		glm.ObsVariance[di] = 0
		glm.ErrVariance[di] = 0
	}
	for i := 0; i < n; i++ {
		row := ix.Indexes[i]
		for di := 0; di < nDv; di++ {
			o := dv.FloatRowCell(row, di) - obsMeans[di]
			glm.ObsVariance[di] += o * o
			e := ev.FloatRowCell(row, di) - errMeans[di]
			glm.ErrVariance[di] += e * e
		}
	}
	for di := 0; di < nDv; di++ {
		glm.ObsVariance[di] *= norm
		glm.ErrVariance[di] *= norm
		glm.R2[di] = 1.0 - (glm.ErrVariance[di] / glm.ObsVariance[di])
	}
}

// Variance returns a description of the variance accounted for by the regression
// equation, R^2, for each dependent variable, along with the variances of
// observed and errors (residuals), which are used to compute it.
func (glm *GLM) Variance() string {
	str := ""
	for di := range glm.R2 {
		if len(glm.DepNames) > di && glm.DepNames[di] != "" {
			str += glm.DepNames[di]
		} else {
			str += fmt.Sprintf("DV %d", di)
		}
		str += fmt.Sprintf("\tR^2: %8.6g\tR: %8.6g\tVar Err: %8.4g\t Obs: %8.4g\n", glm.R2[di], math.Sqrt(glm.R2[di]), glm.ErrVariance[di], glm.ObsVariance[di])
	}
	return str
}

// Coeffs returns a string describing the coefficients
func (glm *GLM) Coeffs() string {
	str := ""
	for di := range glm.NDepVars {
		if len(glm.DepNames) > di && glm.DepNames[di] != "" {
			str += glm.DepNames[di]
		} else {
			str += fmt.Sprintf("DV %d", di)
		}
		str += " = "
		for ii := 0; ii <= glm.NIndepVars; ii++ {
			str += fmt.Sprintf("\t%8.6g", glm.Coeff.Value([]int{di, ii}))
			if ii < glm.NIndepVars {
				str += " * "
				if len(glm.IndepNames) > ii && glm.IndepNames[di] != "" {
					str += glm.IndepNames[di]
				} else {
					str += fmt.Sprintf("IV_%d", ii)
				}
				str += " + "
			}
		}
		str += "\n"
	}
	return str
}
