package utils

import "math"

// roundFloat rounds a float to the nearest n integer
func RoundFloat(f float64, n int) float64 {
	pow := math.Pow10(n)
	return math.Round(f*pow) / pow
}

// Remove removes a specific element from a slice
func Remove(slice []string, s string) []string {
	for i, v := range slice {
		if v == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func AdaptTotalUSD(totalETH float64, ethPriceUSD float64) float64 {
	v := totalETH * ethPriceUSD
	// Round to 2 decimal places
	v = math.Round(v*100) / 100

	return v
}
