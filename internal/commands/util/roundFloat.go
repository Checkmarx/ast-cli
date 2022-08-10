package util

import "math"

func RoundFloat(val float64, precision uint) float64 {
	const div = 10
	ratio := math.Pow(div, float64(precision))
	return math.Round(val*ratio) / ratio
}
