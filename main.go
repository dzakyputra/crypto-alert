package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func main() {

	port := os.Getenv("PORT")
	// port := "8081"

	http.HandleFunc("/", welcome)
	http.HandleFunc("/automate-screening", automateScreening)
	http.HandleFunc("/check-order-status", checkOrderStatus)
	http.HandleFunc("/check-stop-loss", checkStopLoss)
	http.HandleFunc("/test", test)
	http.ListenAndServe(":"+port, nil)
}

func initBinanceClient() *binance.Client {
	return binance.NewClient(BINANCE_API_KEY, BINANCE_SECRET_KEY)
}

func initGoogleSheetClient() *sheets.Service {
	ctx := context.Background()
	creds := "credentials.json"

	service, err := sheets.NewService(ctx, option.WithCredentialsFile(creds))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	return service
}

func welcome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!") // Write the response to the client
}

func test(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!") // Write the response to the client
}

func checkStopLoss(w http.ResponseWriter, r *http.Request) {

	// Initialization
	var wgInit sync.WaitGroup
	var binanceClient *binance.Client
	var sheetsClient *sheets.Service

	wgInit.Add(1)
	go func() {
		defer wgInit.Done()

		binanceClient = initBinanceClient()
	}()

	wgInit.Add(1)
	go func() {
		defer wgInit.Done()

		sheetsClient = initGoogleSheetClient()
	}()
	wgInit.Wait()

	// Get All Trading Data
	data, _ := getAllTradingFromGoogleSheets(sheetsClient)

	// Get All Symbols
	symbols := make(map[string]bool)
	for _, pairs := range data.Values {
		if len(pairs) >= 7 {
			orgClientOrderID := pairs[5].(string)
			status := pairs[6].(string)
			symbol := pairs[1].(string)

			if status == "NEW" && orgClientOrderID != "error" {
				symbols[symbol] = true
			}
		}
	}

	// Get The Latest Price
	var wg sync.WaitGroup
	var m sync.Mutex
	maxWorkers := 20
	semaphore := make(chan struct{}, maxWorkers)
	prices := make(map[string]float64)

	for symbol, _ := range symbols {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(symbol string) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			klines, err := getKlines(binanceClient, symbol, 1)
			if err != nil {
				return
			}

			currentPrice, err := strconv.ParseFloat(klines[0].Close, 64)
			if err != nil {
				return
			}

			m.Lock()
			defer m.Unlock()
			prices[symbol] = currentPrice

		}(symbol)
	}

	wg.Wait()

	// Stop Loss and Sell Order
	for i, pairs := range data.Values {
		if len(pairs) >= 7 {
			orgClientOrderID := pairs[5].(string)
			status := pairs[6].(string)
			symbol := pairs[1].(string)
			buyPrice := pairs[3].(string)
			quantity := pairs[2].(string)
			buyPriceFloat64, _ := strconv.ParseFloat(buyPrice, 64)

			if status == "NEW" && orgClientOrderID != "error" {
				if prices[symbol] < 0.98*buyPriceFloat64 {

					fmt.Println("SELL", symbol, buyPriceFloat64, 0.98*buyPriceFloat64, prices[symbol])

					if quantity == "a" {
						continue
					}

					_, err := binanceClient.NewCancelOrderService().Symbol(symbol).OrigClientOrderID(orgClientOrderID).Do(context.Background())
					if err != nil {
						fmt.Println("ERROR CANCEL ORDER", err)
						continue
					}

					sellMarketResponse, err := binanceClient.NewCreateOrderService().Symbol(symbol).
						Side(binance.SideTypeSell).
						Type(binance.OrderTypeMarket).
						Quantity(quantity).
						Do(context.Background())
					if err != nil {
						fmt.Println(err)
						continue
					}

					var totalPrices float64
					for _, fill := range sellMarketResponse.Fills {
						price, _ := strconv.ParseFloat(fill.Price, 64)
						totalPrices += price
					}
					averageSellMarketPrice := totalPrices / float64(len(sellMarketResponse.Fills))

					editAllTradingDataToGoogleSheets(sheetsClient, "E", i+2, fmt.Sprint(averageSellMarketPrice))

				}
			}
		}
	}

	fmt.Println("REFRESHED!!")

	fmt.Fprintf(w, "hai!")
}

func checkOrderStatus(w http.ResponseWriter, r *http.Request) {

	// Initialization
	var wgInit sync.WaitGroup
	var binanceClient *binance.Client
	var sheetsClient *sheets.Service

	wgInit.Add(1)
	go func() {
		defer wgInit.Done()

		binanceClient = initBinanceClient()
	}()

	wgInit.Add(1)
	go func() {
		defer wgInit.Done()

		sheetsClient = initGoogleSheetClient()
	}()
	wgInit.Wait()

	// Get All Trading Data
	data, _ := getAllTradingFromGoogleSheets(sheetsClient)

	// Check the Order Status
	for i, pairs := range data.Values {
		if len(pairs) >= 7 {
			orgClientOrderID := pairs[5].(string)
			symbol := pairs[1].(string)
			status := pairs[6].(string)

			if (status == "NEW" || status == "PARTIALLY_FILLED") && orgClientOrderID != "error" {
				result, err := binanceClient.NewGetOrderService().Symbol(symbol).OrigClientOrderID(orgClientOrderID).Do(context.Background())
				if err != nil {
					fmt.Println("[ERROR]", err)
					status = "NEW"
				} else {
					status = string(result.Status)
				}

				editAllTradingDataToGoogleSheets(sheetsClient, "G", i+2, status)
			}
		}
	}

	fmt.Println("REFRESHED!!")

	fmt.Fprintf(w, "hai!")
}

func automateScreening(w http.ResponseWriter, r *http.Request) {

	// Initialization
	var wgInit sync.WaitGroup
	var binanceClient *binance.Client
	var sheetsClient *sheets.Service

	wgInit.Add(1)
	go func() {
		defer wgInit.Done()

		binanceClient = initBinanceClient()
	}()

	wgInit.Add(1)
	go func() {
		defer wgInit.Done()

		sheetsClient = initGoogleSheetClient()
	}()
	wgInit.Wait()

	// Get initial data
	var err error
	var wgGetData sync.WaitGroup
	var asset binance.UserAssetRecord
	var symbols []binance.Symbol
	var tradingIndormationData TradingIndormationData
	var tradingDetails []TradingDetails
	var blacklistAssets map[string]int

	wgGetData.Add(1)
	go func() {
		defer wgGetData.Done()

		asset, err = getUserAsset(binanceClient, "USDT")
		if err != nil {
			fmt.Println(err)
		}
	}()

	wgGetData.Add(1)
	go func() {
		defer wgGetData.Done()

		symbols, err = getActivePairs(binanceClient)
		if err != nil || len(symbols) <= 0 {
			fmt.Println(err, len(symbols))
		}
	}()

	wgGetData.Add(1)
	go func() {
		defer wgGetData.Done()

		data, err := getDataFromGoogleSheets(sheetsClient)
		if err != nil {
			fmt.Println(err)
		}
		tradingIndormationData = getTradingInformation(data)
	}()

	wgGetData.Add(1)
	go func() {
		defer wgGetData.Done()

		data, err := getTradingDetailsFromGoogleSheets(sheetsClient)
		if err != nil {
			fmt.Println(err)
		}
		blacklistAssets, tradingDetails = getTradingDetails(data)
	}()

	wgGetData.Wait()

	// Trading Logic
	upperParameters, lowerParameters := getParametersPerPairs(binanceClient, symbols)
	_, _, resultTrading := tradingLogic(binanceClient, asset, tradingIndormationData, blacklistAssets, upperParameters)

	// Write data to the Google Sheets
	var wgWriteData sync.WaitGroup

	wgWriteData.Add(1)
	go func() {
		defer wgWriteData.Done()

		writeTradingInformationDataToGoogleSheets(sheetsClient, upperParameters, tradingIndormationData)
	}()

	wgWriteData.Add(1)
	go func() {
		defer wgWriteData.Done()

		overwriteTradingDetailsToGoogleSheets(sheetsClient, append(tradingDetails, resultTrading...))
	}()

	wgWriteData.Add(1)
	go func() {
		defer wgWriteData.Done()

		writeAllTradingToGoogleSheets(sheetsClient, resultTrading)
	}()

	// wgWriteData.Add(1)
	// go func() {
	// 	defer wgWriteData.Done()

	// 	if isEligibleToTrade {
	// 		sendTelegramMessage("TRADE DATA", newParameters)
	// 		writeDummyTradeDataToGoogleSheets(sheetsClient, newParameters)
	// 	}
	// }()

	wgWriteData.Add(1)
	go func() {
		defer wgWriteData.Done()

		sendTelegramMessage("PARAMETER DATA", upperParameters, lowerParameters)
	}()

	wgWriteData.Wait()
}

func tradingLogic(client *binance.Client, asset binance.UserAssetRecord, tradingInformationData TradingIndormationData, blacklistAssets map[string]int, parameters map[string]Parameters) (bool, map[string]Parameters, []TradingDetails) {

	result := make(map[string]Parameters)
	resultTrading := []TradingDetails{}

	balance, _ := strconv.ParseFloat(asset.Free, 64)
	if balance <= MINIMUM_BALANCE {
		return false, result, resultTrading
	}

	for _, pair := range tradingInformationData.LastAlertCoin {
		if _, exists := parameters[pair]; exists {
			result[pair] = parameters[pair]
		}
	}

	if len(result) <= 30 {
		return false, result, resultTrading
	}

	var maxDivider int
	if len(result) >= 4 {
		maxDivider = 4
	} else if len(result) == 1 {
		maxDivider = 2
	} else {
		maxDivider = len(result)
	}

	balanceToTrade := balance - MINIMUM_BALANCE
	balancePerTrade := balanceToTrade / float64(maxDivider)

	if balancePerTrade <= 10 {
		return false, result, resultTrading
	}

	if balancePerTrade > 10 {
		balancePerTrade = 10
	}

	fmt.Println("[TRADE] Length:", len(result), " | Divider:", maxDivider, " | Balance Per Trade:", balancePerTrade)
	fmt.Println(mapKeyToString(result))

	var i int
	for pair, _ := range result {

		if blacklistAssets[pair] >= 2 {
			continue
		}

		if parameters[pair].TickSize <= 0 {
			continue
		}

		i++

		fmt.Println("\n", pair)

		orderResponse, err := client.NewCreateOrderService().Symbol(pair).
			Side(binance.SideTypeBuy).
			Type(binance.OrderTypeMarket).
			QuoteOrderQty(fmt.Sprintf("%f", balancePerTrade)).
			Do(context.Background())
		if err != nil {
			fmt.Println(err)
			continue
		}

		var totalPrices float64
		for _, fill := range orderResponse.Fills {
			price, _ := strconv.ParseFloat(fill.Price, 64)
			totalPrices += price
		}
		averageBuyPrice := totalPrices / float64(len(orderResponse.Fills))
		sellPrice := averageBuyPrice * 1.02

		fmt.Println("[BUY] ", pair, averageBuyPrice, parameters[pair].TickSize)

		var sellPriceStr string
		switch parameters[pair].TickSize {
		case 1:
			sellPriceStr = fmt.Sprintf("%.1f", sellPrice)
		case 2:
			sellPriceStr = fmt.Sprintf("%.2f", sellPrice)
		case 3:
			sellPriceStr = fmt.Sprintf("%.3f", sellPrice)
		case 4:
			sellPriceStr = fmt.Sprintf("%.4f", sellPrice)
		case 5:
			sellPriceStr = fmt.Sprintf("%.5f", sellPrice)
		case 6:
			sellPriceStr = fmt.Sprintf("%.6f", sellPrice)
		case 7:
			sellPriceStr = fmt.Sprintf("%.7f", sellPrice)
		case 8:
			sellPriceStr = fmt.Sprintf("%.8f", sellPrice)
		default:
			sellPriceStr = fmt.Sprintf("%.9f", sellPrice)
		}

		fmt.Println("[TRY TO SELL] ", pair, " SELL PRICE: ", sellPrice, sellPriceStr, orderResponse.ExecutedQuantity)

		sellResponse, err := client.NewCreateOrderService().Symbol(pair).
			Side(binance.SideTypeSell).
			Type(binance.OrderTypeLimit).
			TimeInForce(binance.TimeInForceTypeGTC).
			Quantity(orderResponse.ExecutedQuantity).
			Price(sellPriceStr).
			Do(context.Background())
		if err != nil {
			fmt.Println(err)
		}

		clientOID := "error"
		if err == nil {
			clientOID = sellResponse.ClientOrderID
		}

		resultTrading = append(resultTrading, TradingDetails{
			OrderID:   clientOID,
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			Pair:      pair,
			Quantity:  orderResponse.ExecutedQuantity,
			BuyPrice:  averageBuyPrice,
			SellPrice: sellPriceStr,
		})

		fmt.Println("[SUCCESS] ", pair, " SELL PRICE: ", sellPrice, sellPriceStr)

		if i >= maxDivider {
			break
		}
	}

	fmt.Println("\n")

	return true, result, resultTrading

}
