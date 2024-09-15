package main

import "time"

type Parameters struct {
	Symbol                string
	DateTime              time.Time
	MovingAverage         float64
	RelativeStrengthIndex float64
	IsGulfingCandles      bool
	Volume                float64
	VolumeDiff            float64
	IsUpperTrend          bool
	IsBreakResistance     bool
	IsBreakSupport        bool
	CurrentPrice          float64

	// Combination 1 or more parameters
	Param1 bool
	Param2 bool

	// Others
	TickSize int
}

type TradingIndormationData struct {
	LastAlertCoin          []string
	CurrentTotalAlertCoin  string
	PreviousTotalAlertCoin string
}

type TradingDetails struct {
	OrderID   string
	Timestamp string
	Pair      string
	Quantity  string
	BuyPrice  float64
	SellPrice string
	Status    string
}
