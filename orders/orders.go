package orders

import (
	"fmt"
	"github.com/preichenberger/go-coinbasepro/v2"
	"math"
	"strconv"
	"strings"
	"time"
)

type BoughtCoin struct {
	Symbol string
	Available float64
	Balance float64
	SellPoint float64
	HighestPrice float64
	PriceIncrement float64
	SizeIncrement float64
	OrderID string
	BuyComplete bool
	StopLimitSet bool
	TimeBought time.Time
}

func LimitBuyOrder (client *coinbasepro.Client, symbol string, pInc float64, sInc float64, fundsAvailable float64) (BoughtCoin, error) {
	ticker, _ := client.GetTicker(symbol)
	price, _ := strconv.ParseFloat(ticker.Price, 64)
	price *= 1.0025
	orderSize := fundsAvailable / price
	order := coinbasepro.Order{
		Type:           "limit",
		Side:           "buy",
		ProductID:      symbol,
		Price: 			fmt.Sprintf("%f", math.Floor(price*pInc)/pInc),
		Size:           fmt.Sprintf("%f", math.Floor(orderSize*sInc)/sInc),
		TimeInForce:    "GTT",
		CancelAfter: 	"min",
	}
	savedOrder, err := client.CreateOrder(&order)
	newCoin := BoughtCoin {
		Symbol:         symbol,
		Available:      math.Floor(orderSize*sInc)/sInc,
		Balance:        math.Floor(orderSize*sInc)/sInc,
		SellPoint:      (math.Floor(price*pInc)/pInc) * .9,
		HighestPrice:   math.Floor(price*pInc)/pInc,
		PriceIncrement: pInc,
		SizeIncrement:  sInc,
		OrderID:        savedOrder.ID,
		BuyComplete: false,
		StopLimitSet: false,
		TimeBought: time.Now(),
	}
	if err != nil {
		fmt.Println(err.Error())
		if strings.Contains(err.Error(), "Minimum size") {
			return BoughtCoin{}, err
		} else if strings.Contains(err.Error(), "size is too accurate") {
			newCoin, err = LimitBuyOrder(client,symbol, pInc, sInc / 10, fundsAvailable)
		} else if strings.Contains(err.Error(), "price is too accurate") {
			newCoin, err = LimitBuyOrder(client, symbol, pInc / 10, sInc, fundsAvailable)
		}
	}

	return newCoin, err
}
func StopLimitSellOrder (client *coinbasepro.Client, coin BoughtCoin, pInc float64, sInc float64) (BoughtCoin, error) {
	stopPriceString := fmt.Sprintf("%f", math.Floor(coin.SellPoint*pInc) / pInc)
	price := coin.HighestPrice * .8475
	price = math.Floor(coin.SellPoint*pInc) / pInc
	priceString := fmt.Sprintf("%f", price)
	sizeString := fmt.Sprintf("%f", math.Floor(coin.Balance*sInc) / sInc)
	//Set stop market at 5%
	order := coinbasepro.Order{
		Type:           "limit",
		Side:           "sell",
		Stop: 			"loss",
		StopPrice:      stopPriceString,
		Price: 			priceString,
		ProductID:      coin.Symbol,
		Size:          sizeString,
	}
	savedOrder, err := client.CreateOrder(&order)
	if err != nil {
		if strings.Contains(err.Error(), "Minimum size") {
			return coin, err
		} else if strings.Contains(err.Error(), "size is too accurate") {
			coin.SizeIncrement = coin.SizeIncrement / 10
			coin, err = StopLimitSellOrder(client, coin, pInc, sInc / 10)
		} else if strings.Contains(err.Error(), "price is too accurate") {
			coin.PriceIncrement = coin.PriceIncrement / 10
			coin, err = StopLimitSellOrder(client, coin, pInc / 10, sInc)
		}
	} else {
		coin.OrderID = savedOrder.ID
	}

	return coin, err
}
func LimitSellOrder (client *coinbasepro.Client, symbol string, balance float64, pInc float64, sInc float64) error{
	ticker, _ := client.GetTicker(symbol)
	price, _ := strconv.ParseFloat(ticker.Price, 64)
	price = price * .9975
	order := coinbasepro.Order{
		Type:      "limit",
		Side:      "sell",
		ProductID: symbol,
		Size:      fmt.Sprintf("%f", math.Floor(balance*sInc)/sInc),
		Price: 	   fmt.Sprintf("%f", math.Floor(price*pInc)/pInc),
	}
	_, err := client.CreateOrder(&order)
	if err != nil {
		if strings.Contains(err.Error(), "Minimum size") {
			return err
		} else if strings.Contains(err.Error(), "size is too accurate") {
			err = LimitSellOrder(client,symbol, balance, pInc, sInc / 10)
		} else if strings.Contains(err.Error(), "price is too accurate") {
			err = LimitSellOrder(client, symbol, balance, pInc / 10, sInc)
		}
	}

	return err
}
func MarketBuyOrder (client *coinbasepro.Client, symbol string, fundsAvailable float64, inc float64) (BoughtCoin, error) {
	order := coinbasepro.Order{
		Type:      "market",
		Side:      "buy",
		ProductID: symbol,
		Funds:     fmt.Sprintf("%f", math.Floor(fundsAvailable*inc)/inc),
	}
	savedOrder, err := client.CreateOrder(&order)
	newCoin := BoughtCoin{
		Symbol:       symbol,
		Available:    0,
		Balance:      0,
		SellPoint:    0,
		HighestPrice: 0,
		OrderID:      savedOrder.ID,
		PriceIncrement: 100000,
		SizeIncrement:  100000,
		BuyComplete:  false,
		StopLimitSet: false,
		TimeBought:   time.Now(),
	}
	if err != nil {
		if strings.Contains(err.Error(), "funds is too accurate.") {
			newCoin, err = MarketBuyOrder(client, symbol, fundsAvailable, inc / 10)
		}
	}
	return newCoin, err
}
func MarketSellOrder (client *coinbasepro.Client, coin BoughtCoin, inc float64) error {
	order := coinbasepro.Order{
		Type:      "market",
		Side:      "sell",
		ProductID: coin.Symbol,
		Size:     fmt.Sprintf("%f", math.Floor(coin.Balance*inc)/inc),
	}
	_, err := client.CreateOrder(&order)
	if err != nil {
		if strings.Contains(err.Error(), "Minimum size") || strings.Contains(err.Error(), "size is too accurate") {
			coin.Balance /= 10
			err = MarketSellOrder(client, coin, inc / 10)
		}
	}
	return err
}
