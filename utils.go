package main

import (
	"github.com/MicahParks/go-rsi/v2"
	"github.com/adshao/go-binance/v2"
)

// Parameters

func calculateMovingAverage(prices []float64, period int) float64 {
	var totalPrices float64
	prices = prices[len(prices)-(period+1):]
	for i := 0; i < len(prices)-1; i++ {
		totalPrices += prices[i]
	}
	return totalPrices / float64(len(prices)-1)
}

func calculateRelativeStrengthIndex(prices []float64, period int) float64 {
	initial := prices[:period]
	r, result := rsi.New(initial)
	remaining := prices[period : len(prices)-1]
	for _, next := range remaining {
		result = r.Calculate(next)
	}
	return result
}

func calculateVolumdeDiff(volumes []float64) float64 {
	before := sumSliceFloat64(volumes[len(volumes)-7 : len(volumes)-4])
	after := sumSliceFloat64(volumes[len(volumes)-4 : len(volumes)-1])
	return ((after - before) / before) * 100
}

func isGulfingCandles(klines []*binance.Kline) bool {
	currentPrice, previous1Price, previous2Price := klines[len(klines)-1], klines[len(klines)-2], klines[len(klines)-3]
	if previous2Price.Close > previous2Price.Open {
		return false
	}
	if previous1Price.Close < previous1Price.Open {
		return false
	}
	if (previous1Price.Open < previous2Price.Close) && (previous1Price.Close > previous2Price.Open) && (currentPrice.Close > previous1Price.Close) {
		return true
	}
	return false
}

// Others

func sumSliceFloat64(lists []float64) (total float64) {
	for _, value := range lists {
		total += value
	}
	return total
}

func mapKeyToString(maps map[string]Parameters) (result string) {
	for key, _ := range maps {
		result += key + " "
	}
	return result
}
