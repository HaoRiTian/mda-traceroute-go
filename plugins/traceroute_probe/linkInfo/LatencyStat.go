package linkInfo

import (
	"math"
	"sync"
)

var ONLINE_STATS_SYNC_POOL *sync.Pool

type LatencyStat struct {
	Cnt  float64 `json:"cnt"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Mean float64 `json:"mean"`
	M2   float64 `json:"m2"`
	M3   float64 `json:"m3"`
	M4   float64 `json:"m4"`
}

func init() {
	ONLINE_STATS_SYNC_POOL = &sync.Pool{New: func() interface{} {
		return &LatencyStat{
			Cnt:  0,
			Min:  0,
			Max:  0,
			Mean: 0,
			M2:   0,
			M3:   0,
			M4:   0,
		}
	}}
}

func NewLatencyStat() *LatencyStat {
	return ONLINE_STATS_SYNC_POOL.Get().(*LatencyStat)
}

// Append is used to store new data for stats, highOrder is in range 2~4 for X^2~4 stats
func (ls *LatencyStat) Append(x float64, highOrder uint8) {
	n1 := ls.Cnt
	if n1 == 0 {
		ls.Min = x
		ls.Max = x
	}
	if x < ls.Min {
		ls.Min = x
	}
	if x > ls.Max {
		ls.Max = x
	}
	ls.Cnt = ls.Cnt + 1
	delta := x - ls.Mean
	cmpt := delta / ls.Cnt
	cmpt2 := cmpt * cmpt
	term1 := delta * cmpt * n1
	ls.Mean = ls.Mean + cmpt

	switch highOrder {
	case 4:
		ls.M4 = ls.M4 + term1*cmpt2*(ls.Cnt*ls.Cnt-3*ls.Cnt+3) + 6*cmpt2*ls.M2 - 4*cmpt*ls.M3
		ls.M3 = ls.M3 + term1*cmpt*(ls.Cnt-2) - 3*cmpt*ls.M2
		ls.M2 = ls.M2 + term1
	case 3:
		ls.M3 = ls.M3 + term1*cmpt*(ls.Cnt-2) - 3*cmpt*ls.M2
		ls.M2 = ls.M2 + term1
	case 2:
		ls.M2 = ls.M2 + term1
	default:
	}
}

// Len is return data count
func (ls *LatencyStat) Len() float64 {
	return ls.Cnt
}

// Sum is return data sum
func (ls *LatencyStat) Sum() float64 {
	return ls.Mean * ls.Cnt
}

// Variance is return the variance
func (ls *LatencyStat) Variance() float64 {
	if ls.Cnt < 2 {
		return float64(0.0)
	} else {
		return ls.M2 / ls.Cnt
	}

}

// Std is return the std
func (ls *LatencyStat) Std() float64 {
	if ls.Cnt < 2 {
		return float64(0.0)
	} else {
		return math.Sqrt(ls.M2 / ls.Cnt)
	}
}

// Skewness is return skewness...
func (ls *LatencyStat) Skewness() float64 {
	if ls.M2 < 1e-14 || ls.Cnt <= 3 {
		return float64(0.0)
	} else {
		return math.Sqrt(ls.Cnt) * ls.M3 / ls.M2 / math.Sqrt(ls.M2)
	}
}

// Kurtosis is return kurtosis
func (ls *LatencyStat) Kurtosis() float64 {
	if ls.M2 < 1e-14 || ls.Cnt <= 4 {
		return float64(0.0)
	} else {
		return (ls.Cnt*ls.M4)/(ls.M2*ls.M2) - 3
	}
}
