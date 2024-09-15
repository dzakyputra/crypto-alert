package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"
)

func getDataFromGoogleSheets(service *sheets.Service) (*sheets.ValueRange, error) {
	ctx := context.Background()

	writeRange := "data!A1:B4"
	resp, err := service.Spreadsheets.Values.Get(SPREADSHEET_ID, writeRange).Context(ctx).Do()
	if err != nil {
		fmt.Println("[Read Trading Information Data] Unable to retrieve data from sheet: ", err)
		return resp, err
	}

	return resp, err
}

func getTradingDetailsFromGoogleSheets(service *sheets.Service) (*sheets.ValueRange, error) {
	ctx := context.Background()

	writeRange := "trading_details!A2:ZZ"
	resp, err := service.Spreadsheets.Values.Get(SPREADSHEET_ID, writeRange).Context(ctx).Do()
	if err != nil {
		fmt.Println("[Read Trading Details] Unable to retrieve data from sheet: ", err)
		return resp, err
	}

	return resp, err
}

func getAllTradingFromGoogleSheets(service *sheets.Service) (*sheets.ValueRange, error) {
	ctx := context.Background()

	writeRange := "all_trading!A2:ZZ"
	resp, err := service.Spreadsheets.Values.Get(SPREADSHEET_ID, writeRange).Context(ctx).Do()
	if err != nil {
		fmt.Println("[Read Trading Details] Unable to retrieve data from sheet: ", err)
		return resp, err
	}

	return resp, err
}

func writeAllTradingToGoogleSheets(service *sheets.Service, tradingDetails []TradingDetails) {
	ctx := context.Background()

	writeRange := "all_trading!A1"
	resp, err := service.Spreadsheets.Values.Get(SPREADSHEET_ID, writeRange).Context(ctx).Do()
	if err != nil {
		fmt.Println("[Write All Trading] Unable to retrieve data from sheet: ", err)
	}

	values := [][]interface{}{}
	for _, detail := range tradingDetails {
		values = append(values, []interface{}{
			detail.Timestamp,
			detail.Pair,
			detail.Quantity,
			detail.BuyPrice,
			detail.SellPrice,
			detail.OrderID,
			"NEW",
		})
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	appendRange := fmt.Sprintf("all_trading!A%d", len(resp.Values)+1)

	_, err = service.Spreadsheets.Values.Append(SPREADSHEET_ID, appendRange, valueRange).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		fmt.Println("[Write All Trading] Unable to append data: %v")
	}

	fmt.Println("[Write All Trading] Data appended successfully...")
}

func writeDummyTradeDataToGoogleSheets(service *sheets.Service, parameters map[string]Parameters) {
	ctx := context.Background()

	writeRange := "dummy_trade!A1"
	resp, err := service.Spreadsheets.Values.Get(SPREADSHEET_ID, writeRange).Context(ctx).Do()
	if err != nil {
		fmt.Println("[Write Dummy Trade] Unable to retrieve data from sheet: ", err)
	}

	values := [][]interface{}{}

	for _, param := range parameters {
		values = append(values, []interface{}{
			param.Symbol,
			param.DateTime,
			param.MovingAverage,
			param.RelativeStrengthIndex,
			param.IsGulfingCandles,
			param.Volume,
			param.VolumeDiff,
			param.CurrentPrice,
			param.Param1,
			param.Param2,
		})
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	appendRange := fmt.Sprintf("dummy_trade!A%d", len(resp.Values)+1)

	_, err = service.Spreadsheets.Values.Append(SPREADSHEET_ID, appendRange, valueRange).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		fmt.Println("[Write Dummy Trade] Unable to append data: %v")
	}

	fmt.Println("[Write Dummy Trade] Data appended successfully...")
}

func overwriteTradingDetailsToGoogleSheets(service *sheets.Service, tradingDetails []TradingDetails) {
	writeRange := "trading_details!A2:ZZ"
	values := [][]interface{}{}
	for _, param := range tradingDetails {
		values = append(values, []interface{}{
			param.Timestamp,
			param.Pair,
			param.BuyPrice,
			param.SellPrice,
		})
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	clearReq := sheets.ClearValuesRequest{}
	_, err := service.Spreadsheets.Values.Clear(SPREADSHEET_ID, writeRange, &clearReq).Do()
	if err != nil {
		fmt.Printf("[Overwrite Trading Details] Unable to clear values in range %s: %v", writeRange, err)
	}

	_, err = service.Spreadsheets.Values.Update(SPREADSHEET_ID, writeRange, valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		fmt.Printf("[Overwrite Trading Details] Unable to update data in sheet: %v", err)
	}

	fmt.Printf("[Overwrite Trading Details] Updated cell %s with value %v\n", writeRange, valueRange.Values)
}

func overwriteAllTradingGoogleSheets(service *sheets.Service, tradingDetails []TradingDetails) {
	writeRange := "all_trading!A2:ZZ"
	values := [][]interface{}{}
	for _, param := range tradingDetails {
		values = append(values, []interface{}{
			param.Timestamp,
			param.Pair,
			param.Quantity,
			param.BuyPrice,
			param.SellPrice,
			param.OrderID,
			param.Status,
		})
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	clearReq := sheets.ClearValuesRequest{}
	_, err := service.Spreadsheets.Values.Clear(SPREADSHEET_ID, writeRange, &clearReq).Do()
	if err != nil {
		fmt.Printf("[Overwrite All Trading] Unable to clear values in range %s: %v", writeRange, err)
	}

	_, err = service.Spreadsheets.Values.Update(SPREADSHEET_ID, writeRange, valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		fmt.Printf("[Overwrite All Trading] Unable to update data in sheet: %v", err)
	}

	fmt.Printf("[Overwrite All Trading] Updated cell %s with value %v\n", writeRange, valueRange.Values)
}

func writeTradingInformationDataToGoogleSheets(service *sheets.Service, parameters map[string]Parameters, tradingIndormationData TradingIndormationData) {
	writeRange := "data!B2:B4"
	var values string
	var total int
	for key, _ := range parameters {
		total++
		if values == "" {
			values += key
			continue
		}
		values += "," + key
	}

	if values == "" {
		values = "a"
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{
			{values},
			{total},
			{tradingIndormationData.CurrentTotalAlertCoin},
		},
	}

	_, err := service.Spreadsheets.Values.Update(SPREADSHEET_ID, writeRange, valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		fmt.Printf("[Write Trading Information Data] Unable to update data in sheet: %v", err)
	}

	fmt.Printf("[Write Trading Information Data] Updated cell %s with value %v\n", writeRange, valueRange.Values)
}

func editAllTradingDataToGoogleSheets(service *sheets.Service, column string, index int, value string) {
	writeRange := fmt.Sprintf("all_trading!%v%d", column, index)

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{
			{value},
		},
	}

	_, err := service.Spreadsheets.Values.Update(SPREADSHEET_ID, writeRange, valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		fmt.Printf("[Edit All Trading Data] Unable to update data in sheet: %v", err)
	}

	fmt.Printf("[Edit All Trading Data] Updated cell %s with value %v\n", writeRange, valueRange.Values)
}

func getTradingInformation(data *sheets.ValueRange) TradingIndormationData {
	tradingIndormationData := TradingIndormationData{}
	for _, v := range data.Values {
		if v[0].(string) == "lastAlertCoin" {
			tradingIndormationData.LastAlertCoin = strings.Split(v[1].(string), ",")
		} else if v[0].(string) == "currentTotalAlertCoin" {
			tradingIndormationData.CurrentTotalAlertCoin = v[1].(string)
		} else if v[0].(string) == "previousTotalAlertCoin" {
			tradingIndormationData.PreviousTotalAlertCoin = v[1].(string)
		}
	}
	return tradingIndormationData
}

func getTradingDetails(data *sheets.ValueRange) (map[string]int, []TradingDetails) {
	blacklistAssets := make(map[string]int)
	result := []TradingDetails{}
	for _, d := range data.Values {
		if len(d) >= 4 {

			tradingTime, err := time.Parse("2006-01-02 15:04:05", d[0].(string))
			if err != nil {
				continue
			}

			duration := time.Now().Sub(tradingTime).Minutes()

			if duration < 480 {
				blacklistAssets[d[1].(string)] += 1

				buyPrice := d[2].(string)
				buyPriceFloat, _ := strconv.ParseFloat(buyPrice, 64)
				result = append(result, TradingDetails{
					Timestamp: d[0].(string),
					Pair:      d[1].(string),
					BuyPrice:  buyPriceFloat,
					SellPrice: d[3].(string),
				})
			}
		}
	}

	return blacklistAssets, result
}
