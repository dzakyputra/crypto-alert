package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"
)

func getUserAsset(binanceClient *binance.Client, symbol string) (result binance.UserAssetRecord, err error) {
	assets, err := binanceClient.NewGetUserAsset().Do(context.Background())
	if err != nil {
		return result, err
	}
	for _, asset := range assets {
		if asset.Asset == symbol {
			return asset, nil
		}
	}
	return result, errors.New("asset not found")
}

func getActivePairs(binanceClient *binance.Client) (symbols []binance.Symbol, err error) {
	res, err := binanceClient.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return symbols, err
	}
	for _, symbol := range res.Symbols {
		if symbol.QuoteAsset != "USDT" {
			continue
		}
		if symbol.Status != "TRADING" {
			continue
		}
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}

func getParametersPerPairs(binanceClient *binance.Client, symbols []binance.Symbol) (map[string]Parameters, map[string]Parameters) {
	var wg sync.WaitGroup
	var m1 sync.Mutex
	var m2 sync.Mutex
	upperParameters := make(map[string]Parameters)
	lowerParameters := make(map[string]Parameters)
	maxWorkers := 20
	semaphore := make(chan struct{}, maxWorkers)

	for _, symbol := range symbols {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(symbol binance.Symbol) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			klines, err := getKlines(binanceClient, symbol.Symbol, 300)
			if err != nil {
				return
			}

			parameter, isValid := generateParameters(klines, symbol.Symbol, symbol.Filters)
			if !isValid {
				return
			}

			// UPPER PARAMETERS
			param1 := (parameter.IsGulfingCandles && (parameter.CurrentPrice >= (parameter.MovingAverage * 1.02)) && (parameter.IsUpperTrend))
			param2 := (parameter.CurrentPrice >= (parameter.MovingAverage * 1.02)) && parameter.IsBreakResistance
			parameter.Param1 = param1
			parameter.Param2 = param2

			if param1 || param2 {
				m1.Lock()
				defer m1.Unlock()
				upperParameters[symbol.Symbol] = parameter
			}

			// LOWER PARAMETERS
			if ((parameter.CurrentPrice <= (parameter.MovingAverage * 0.98)) && !parameter.IsUpperTrend) || parameter.IsBreakSupport {
				m2.Lock()
				defer m2.Unlock()
				lowerParameters[symbol.Symbol] = parameter
			}

		}(symbol)
	}

	wg.Wait()

	return upperParameters, lowerParameters
}

func getKlines(client *binance.Client, symbol string, limit int) ([]*binance.Kline, error) {
	klines, err := client.NewKlinesService().
		Symbol(symbol).
		Interval("15m").
		Limit(limit).
		Do(context.Background())

	if err != nil {
		return klines, err
	}

	return klines, nil
}

func generateParameters(klines []*binance.Kline, symbol string, filters []map[string]interface{}) (parameters Parameters, isValid bool) {
	var closePrices, volumes []float64
	var maxPrice, minPrice float64
	var startMA float64
	var endMA float64

	if len(klines) < 300 {
		return parameters, false
	}

	var tickSize string
	var tickSizeInt int
	for _, filter := range filters {
		if filter["filterType"] == "PRICE_FILTER" {
			tickSize = filter["tickSize"].(string)

			tickSizes := strings.Split(tickSize, ".")

			if len(tickSizes) > 1 {

				if tickSizes[0] == "1" {
					tickSizeInt = 0
					break
				}

				for _, v := range tickSizes[1] {
					tickSizeInt += 1
					if string(v) == "1" {
						break
					}
				}
			}
		}
	}

	for i, k := range klines {
		closePrice, _ := strconv.ParseFloat(k.Close, 64)
		openPrice, _ := strconv.ParseFloat(k.Open, 64)
		volume, _ := strconv.ParseFloat(k.QuoteAssetVolume, 64)
		closePrices = append(closePrices, closePrice)
		volumes = append(volumes, volume)

		if i >= 0 && i < 100 {
			startMA += closePrice
		}

		if i >= 200 && i < 300 {
			endMA += closePrice
		}

		if closePrice > maxPrice {
			maxPrice = closePrice
		}

		if openPrice > maxPrice {
			maxPrice = openPrice
		}

		if closePrice < minPrice {
			minPrice = closePrice
		}

		if openPrice < minPrice {
			minPrice = openPrice
		}
	}

	currentPrice, _ := strconv.ParseFloat(klines[len(klines)-1].Close, 64)

	return Parameters{
		Symbol:                symbol,
		DateTime:              time.UnixMilli(klines[len(klines)-1].CloseTime),
		MovingAverage:         calculateMovingAverage(closePrices, 20),
		RelativeStrengthIndex: calculateRelativeStrengthIndex(closePrices, 14),
		IsGulfingCandles:      isGulfingCandles(klines),
		Volume:                volumes[len(volumes)-1],
		VolumeDiff:            calculateVolumdeDiff(volumes),
		IsUpperTrend:          (startMA / float64(100)) < (endMA / float64(100)),
		IsBreakResistance:     currentPrice >= maxPrice,
		IsBreakSupport:        currentPrice <= minPrice,
		CurrentPrice:          closePrices[len(closePrices)-1],
		TickSize:              tickSizeInt,
	}, true
}
