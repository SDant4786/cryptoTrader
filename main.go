package main

import (
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/preichenberger/go-coinbasepro/v2"
	"github.com/yourusername/yourprojectname/indicators"
	"github.com/yourusername/yourprojectname/orders"
	"github.com/yourusername/yourprojectname/tests"
)

var dontBuy = []string{
	"RAI-USD",
	"PAX-USD",
	"DAI-USD",
	"QUICK-USD",
	"BTC-USD",
	"WBTC-USD",
	"ETH-USD",
	"MKR-USD",
	"LINK-USD",
	"YFI-USD",
	"YFII-USD",
	"IOTX-USD",
}
var lastSellTime time.Time = time.Now()
var lastResetBalance float64 = 0
var usdAccount = ""
var coinsWorthTrading []coinbasepro.Product
var purchasedCoins []orders.BoughtCoin
var accountBalance float64
var client = coinbasepro.NewClient()

func main() {
	log.Println("Starting")
	client.UpdateConfig(&coinbasepro.ClientConfig{
		BaseURL:    "",
		Key:        "",
		Passphrase: "",
		Secret:     "",
	})
	client.HTTPClient = &http.Client{
		Timeout: 15 * time.Second,
	}
	ran := false
	buyAtStart := false
	coinsSold := false
	tests.TestsToRun(client)
	reset()
	getStartingBalance()
	getCoinsWorthTrading()
	//Loop forever
	for {
		//Coin buying loops
		if buyCoinCheck(ran, buyAtStart) {
			buyAtStart = false
			ran = true
			go buyCoinsControl()
		}
		if resetBuyFlagCheck(ran) {
			ran = false
			go resetBalanceControl()
		}
		if time.Now().Minute()%5 == 0 && coinsSold == false {
			coinsSold = true
			go sellCoins()
		}
		if time.Now().Minute()%5 != 0 && coinsSold == true {
			coinsSold = false
		}
	}
}
func buyCoinsControl() {
	time.Sleep(time.Minute * 1)
	buyCoins()
	time.Sleep(time.Minute * 2)
	setStopLoss()
}
func resetBalanceControl() {
	resetCheck()
}
func resetCheck() {
	newBalance := 0.0
	//Get accounts
	accounts, err := client.GetAccounts()
	if err != nil {
		return
	}
	//Filter through accounts
	for _, account := range accounts {
		//Get available amount and current balance
		balance, _ := strconv.ParseFloat(account.Balance, 64)
		ticker, _ := client.GetTicker(account.Currency + "-USD")
		price, _ := strconv.ParseFloat(ticker.Price, 64)
		if balance > 0 && account.Currency != "USD" {
			newBalance += price * balance
		}
		if account.Currency == "USD" {
			usdBalance, _ := strconv.ParseFloat(account.Balance, 64)
			newBalance += usdBalance
		}
	}
	accountBalance = newBalance
	log.Println("Balance: $", math.Floor(accountBalance*100)/100)
}
func sellCoins() {
	time.Sleep(time.Minute * 1)
	var coinsToRemove []string
	var printString string = ""

	if accountBalance > lastResetBalance*1.016 {
		reset()
		getStartingBalance()
		lastResetBalance = accountBalance
		return
	}
	//Filter through bought coins
	for _, pCoin := range purchasedCoins {
		candles, err := client.GetHistoricRates(pCoin.Symbol, coinbasepro.GetHistoricRatesParams{
			Start:       time.Now().Add(time.Duration(-5) * time.Hour),
			End:         time.Now(),
			Granularity: 300,
		})
		if err != nil {
			log.Println(err)
			continue
		}
		time.Sleep(time.Millisecond * 100)
		//Flip candles
		for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
			candles[i], candles[j] = candles[j], candles[i]
		}
		//Trim most recent candle since its not completed
		candles = candles[:len(candles)-1]
		if len(candles) < 28 {
			continue
		}
		//15 min
		candles15, err := client.GetHistoricRates(pCoin.Symbol, coinbasepro.GetHistoricRatesParams{
			Start:       time.Now().AddDate(0, 0, -1),
			End:         time.Now(),
			Granularity: 900,
		})
		if err != nil {
			log.Println(err)
			continue
		}
		time.Sleep(time.Millisecond * 100)
		//Flip candles
		for i, j := 0, len(candles15)-1; i < j; i, j = i+1, j-1 {
			candles15[i], candles15[j] = candles15[j], candles15[i]
		}
		if len(candles15) < 28 {
			continue
		}
		//Trim most recent candle since its not completed
		candles15 = candles15[:len(candles15)-1]
		//Analysis of coin
		hmas := indicators.HMA(candles15, 14)
		hma1 := hmas[len(hmas)-1] - hmas[len(hmas)-2]
		hma2 := hmas[len(hmas)-2] - hmas[len(hmas)-3]
		hma3 := hmas[len(hmas)-3] - hmas[len(hmas)-4]

		ticker, _ := client.GetTicker(pCoin.Symbol)
		price, _ := strconv.ParseFloat(ticker.Price, 64)
		percentIncrease := ((price - pCoin.HighestPrice) / pCoin.HighestPrice) * 100
		if (percentIncrease >= 1.6 &&
			candles[len(candles)-1].Close < candles[len(candles)-2].Close &&
			hma1-hma2 < hma2-hma3) ||
			(time.Now().Sub(pCoin.TimeBought).Hours() >= 24 && price*1.006 > pCoin.HighestPrice) {
			//Cancel stop limit order
			_, err := client.CancelAllOrders(coinbasepro.CancelAllOrdersParams{
				ProductID: pCoin.Symbol},
			)
			if err != nil {
				log.Println(err.Error())
			}
			//Wait a bit to ensure its canceled
			time.Sleep(time.Second * 5)
			//Sell coin
			err = orders.MarketSellOrder(client, pCoin, 100000)
			if err != nil {
				log.Println("Error selling: ", pCoin.Symbol, " Error | ", err.Error())
			} else {
				printString += "Sold by algorithm: " + pCoin.Symbol + " | "
				coinsToRemove = append(coinsToRemove, pCoin.Symbol)
			}
			time.Sleep(time.Millisecond * 100)
		}
	}

	//Remove coin from stored data that have been sold
	for _, pCoin := range purchasedCoins {
		order, _ := client.GetOrder(pCoin.OrderID)
		time.Sleep(time.Millisecond * 100)
		if order.Settled == true && order.Side == "sell" {
			coinsToRemove = append(coinsToRemove, pCoin.Symbol)
			printString += "Sold by stop loss: " + pCoin.Symbol + " | "
		}
	}
	//Remove coin from stored data that have been sold
	for _, ctr := range coinsToRemove {
		purchasedCoins = removeCoins(ctr)
	}
	if printString != "" {
		log.Println(printString)
	}
	if printString != "" {
		lastSellTime = time.Now()
	}
}
func buyCoins() {
	var buyableCoins []string
	var printString string = ""
	//Get coin data
	for _, p := range coinsWorthTrading {
		//If not bought, pull candles
		if checkToBuy(p) == true {
			//15 min
			candles, err := client.GetHistoricRates(p.ID, coinbasepro.GetHistoricRatesParams{
				Start:       time.Now().AddDate(0, 0, -1),
				End:         time.Now(),
				Granularity: 900,
			})
			if err != nil {
				log.Println(err)
				continue
			}
			time.Sleep(time.Millisecond * 100)
			//Flip candles
			for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
				candles[i], candles[j] = candles[j], candles[i]
			}
			if len(candles) < 30 {
				continue
			}
			//Trim most recent candle since its not completed
			candles = candles[:len(candles)-1]
			//Analysis of coin
			hmas := indicators.HMA(candles, 14)
			hma1 := hmas[len(hmas)-1] - hmas[len(hmas)-2]
			hma2 := hmas[len(hmas)-2] - hmas[len(hmas)-3]
			hma3 := hmas[len(hmas)-3] - hmas[len(hmas)-4]
			hma4 := hmas[len(hmas)-4] - hmas[len(hmas)-5]
			rsi := indicators.RSI(candles, 6)

			//Six Hour
			candles, err = client.GetHistoricRates(p.ID, coinbasepro.GetHistoricRatesParams{
				Start:       time.Now().AddDate(0, -2, 0),
				End:         time.Now(),
				Granularity: 86400,
			})
			if err != nil {
				log.Println(err)
				continue
			}
			time.Sleep(time.Millisecond * 100)
			//Flip candles
			for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
				candles[i], candles[j] = candles[j], candles[i]
			}
			if len(candles) < 28 {
				continue
			}
			//Trim most recent candle since its not completed
			candles = candles[:len(candles)-1]
			sixHourHMA := indicators.HMA(candles, 14)
			sixHourRsi := indicators.RSI(candles, 14)
			//Add to buy list
			if rsi[len(rsi)-1] < 25 &&
				hma1-hma2 < hma2-hma3 &&
				hma2-hma3 < hma3-hma4 &&
				sixHourHMA[len(sixHourHMA)-1]-sixHourHMA[len(sixHourHMA)-2] > 0 &&
				sixHourHMA[len(sixHourHMA)-1]-sixHourHMA[len(sixHourHMA)-2] > sixHourHMA[len(sixHourHMA)-2]-sixHourHMA[len(sixHourHMA)-3] &&
				sixHourRsi[len(sixHourRsi)-1] < 60 {
				buyableCoins = append(buyableCoins, p.ID)
			}
		}
	}
	//Only buy 2 coins at a time to prevent heavy losses
	if len(buyableCoins) > 2 {
		buyableCoins = buyableCoins[:2]
	}
	//Get account data
	USDAccount, _ := client.GetAccount(usdAccount)
	time.Sleep(time.Millisecond * 100)
	//Split funds based on how many coins to buy
	availableToBuy, _ := strconv.ParseFloat(USDAccount.Available, 64)
	availableToBuy *= .9975
	amountForEachBuy := accountBalance * .1
	if availableToBuy > accountBalance*.8 && amountForEachBuy > availableToBuy {
		amountForEachBuy = availableToBuy
	}
	if amountForEachBuy > availableToBuy {
		return
	}
	//Buy Coins
	for _, coinToBuy := range buyableCoins {
		newCoin, err := orders.MarketBuyOrder(client, coinToBuy, amountForEachBuy, 100000)
		if err != nil {
			log.Println("Error buying coin: ", coinToBuy, " Error: ", err.Error())
		} else {
			printString += "Placed order for: " + coinToBuy + " | "
			purchasedCoins = append(purchasedCoins, newCoin)
			break
		}
		time.Sleep(time.Second * 1)
	}
	if printString != "" {
		log.Println(printString)
	}
}
func setStopLoss() {
	var coinsToRemove []string
	var printString string = ""
	//Filter through stored coin data
	for i, pCoin := range purchasedCoins {
		if pCoin.StopLimitSet == false {
			order, _ := client.GetOrder(pCoin.OrderID)
			time.Sleep(time.Millisecond * 100)
			filledSize, err := strconv.ParseFloat(order.FilledSize, 64)
			if err != nil {
				log.Println("Error getting filled size. Error: ", err.Error())
				continue
			}
			executedValue, err := strconv.ParseFloat(order.ExecutedValue, 64)
			if err != nil {
				log.Println("Error getting executed value. Error: ", err.Error())
				continue
			}
			filledPrice := executedValue / filledSize
			if order.Status != "active" && order.Side == "buy" {
				if filledSize != 0.0 && executedValue != 0.0 && filledPrice != 0.0 {
					purchasedCoins[i].BuyComplete = true
					purchasedCoins[i].Balance = filledSize
					purchasedCoins[i].HighestPrice = filledPrice
					purchasedCoins[i].SellPoint = filledPrice * .85
				}
				//Check if purchase data has been added
				if purchasedCoins[i].BuyComplete == true {
					//Set stop loss based on purchased price
					var err error
					purchasedCoins[i], err = orders.StopLimitSellOrder(client, purchasedCoins[i], 100000, 100000)
					if err != nil {
						log.Println("Error setting stop loss: ", purchasedCoins[i].Symbol, " | Error: ", err.Error())
					} else {
						//Save the order id so I can cancel it later if price increases
						printString += "Set 10% stop loss for: " + purchasedCoins[i].Symbol + " | "
						purchasedCoins[i].StopLimitSet = true
						time.Sleep(time.Second * 1)
						continue
					}
				}
			}
		}
	}
	//Remove coins with with failed buy orders
	for _, ctr := range coinsToRemove {
		purchasedCoins = removeCoins(ctr)
	}
	if printString != "" {
		log.Println(printString)
	}
}
func reset() {
	log.Println("Resetting")
	accountBalance = 0
	//Get accounts
	accounts, _ := client.GetAccounts()
	time.Sleep(time.Millisecond * 100)
	//Filter through accounts
	for _, account := range accounts {
		//Get available amount and current balance
		balance, _ := strconv.ParseFloat(account.Balance, 64)
		//Sell all code
		if balance > 0 {
			_, err := client.CancelAllOrders(coinbasepro.CancelAllOrdersParams{
				ProductID: account.Currency + "-USD"},
			)
			if err != nil {
				log.Println(err.Error())
			}
			time.Sleep(time.Second * 2)
			err = orders.LimitSellOrder(client, account.Currency+"-USD", balance, 100000, 100000)
			if err != nil {
				//log.Println("Error in reset selling:", account.Currency, "-USD. | Error: ", err.Error())
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
	purchasedCoins = []orders.BoughtCoin{}
	time.Sleep(time.Second * 30)
}
func removeCoins(coinToRemove string) []orders.BoughtCoin {
	ind := -1
	for i, pCoin := range purchasedCoins {
		if pCoin.Symbol == coinToRemove {
			ind = i
		}
	}
	if ind != -1 {
		return append(purchasedCoins[:ind], purchasedCoins[ind+1:]...)
	} else {
		return purchasedCoins
	}
}
func checkToBuy(product coinbasepro.Product) bool {
	buy := true
	for _, pc := range purchasedCoins {
		if pc.Symbol == product.ID {
			buy = false
			break
		}
	}
	return buy
}
func buyCoinCheck(ran bool, buyAtStart bool) bool {
	return (((time.Now().Minute() < 5 && time.Now().Minute() >= 0) ||
		(time.Now().Minute() < 20 && time.Now().Minute() >= 15) ||
		(time.Now().Minute() < 35 && time.Now().Minute() >= 30) ||
		(time.Now().Minute() < 50 && time.Now().Minute() >= 45)) &&
		ran == false) || buyAtStart == true
}
func resetBuyFlagCheck(ran bool) bool {
	return ((time.Now().Minute() > 6 && time.Now().Minute() < 15) ||
		(time.Now().Minute() > 21 && time.Now().Minute() < 30) ||
		(time.Now().Minute() > 36 && time.Now().Minute() < 45) ||
		(time.Now().Minute() > 51 && time.Now().Minute() < 60)) &&
		ran == true
}
func getStartingBalance() {
	accountBalance = 0.0
	accounts, _ := client.GetAccounts()
	time.Sleep(time.Millisecond * 100)
	for _, account := range accounts {
		//Get available amount and current balance
		balance, _ := strconv.ParseFloat(account.Balance, 64)
		ticker, _ := client.GetTicker(account.Currency + "-USD")
		time.Sleep(time.Millisecond * 100)
		price, _ := strconv.ParseFloat(ticker.Price, 64)
		if balance > 0 && account.Currency != "USD" {
			accountBalance += price * balance
		}
		if account.Currency == "USD" {
			availableToBuy, _ := strconv.ParseFloat(account.Available, 64)
			accountBalance += availableToBuy
		}
	}
	lastResetBalance = accountBalance
	log.Println("Starting Balance: ", accountBalance)
}
func getCoinsWorthTrading() {
	coinsWorthTrading = []coinbasepro.Product{}
	//Get Coins
	products, _ := client.GetProducts()
	time.Sleep(time.Millisecond * 100)

	for _, p := range products {
		//If not bought, pull candles
		if strings.Contains(p.ID, "-USD") &&
			!strings.Contains(p.ID, "USDT") &&
			!strings.Contains(p.ID, "USDC") &&
			!strings.Contains(p.ID, "UST") {
			buy := true
			for _, db := range dontBuy {
				if db == p.ID {
					buy = false
					break
				}
			}
			if buy == true {
				coinsWorthTrading = append(coinsWorthTrading, p)
			}
		}
	}
}
