package dataStruct

type LatencyStat struct {
	Count float64 `json:"count"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Mean  float64 `json:"mean"`
	Std   float64 `json:"std"`  // variance 方差
	Skew  float64 `json:"skew"` // skewness 偏度系数
	Kurt  float64 `json:"kurt"` // kurtosis 峰度系数
}
