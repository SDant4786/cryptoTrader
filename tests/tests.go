package tests

import (
	"fmt"
	"github.com/preichenberger/go-coinbasepro/v2"
	"github.com/yourusername/yourprojectname/indicators"
	"log"
	"strings"
	"time"
)
func TestsToRun(client *coinbasepro.Client) {
	//Test(client)
}
func Test(client *coinbasepro.Client) {
	var buyableCoins []string
	coinsWorthTrading := []coinbasepro.Product{}
	//Get Coins
	products, _ := client.GetProducts()

	for _, p := range products {
		//If not bought, pull candles
		if strings.Contains(p.ID, "-USD") &&
			!strings.Contains(p.ID, "-USDT") &&
			!strings.Contains(p.ID, "-USDC") &&
			!strings.Contains(p.ID, "-UST") {
			buy := true
			if buy == true {
				coinsWorthTrading = append(coinsWorthTrading, p)
			}
		}
	}
	//Get coin data
	for _, p := range coinsWorthTrading {
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
		if len(candles) < 28 {
			continue
		}
		//Trim most recent candle since its not completed
		candles = candles[: len(candles) - 3]
		//Analysis of coin
		hmas := indicators.HMA(candles, 14)
		hma1 := hmas[len(hmas)-1] - hmas[len(hmas)-2]
		hma2 := hmas[len(hmas)-2] - hmas[len(hmas)-3]
		rsi := indicators.RSI(candles, 6)
		//Six Hour
		candles, err = client.GetHistoricRates(p.ID, coinbasepro.GetHistoricRatesParams{
			Start:       time.Now().AddDate(0, -1, 0),
			End:         time.Now(),
			Granularity: 21600,
		})
		if err != nil {
			log.Println(err)
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
		//candles = candles[: len(candles) - 1]
		sixHourAC := indicators.CalcAccumulationDistribution(candles[: len(candles) - 1])
		sixHourHMA := indicators.CalcSMA(candles, 14)
		sixHourRsi := indicators.RSI(candles, 14)
		//Add to buy list
		if
			sixHourHMA[len(sixHourHMA)-1] - sixHourHMA[len(sixHourHMA)-2]  > 0 &&
			sixHourHMA[len(sixHourHMA)-1] - sixHourHMA[len(sixHourHMA)-2]  > sixHourHMA[len(sixHourHMA)-2] - sixHourHMA[len(sixHourHMA)-3] &&
			sixHourAC[len(sixHourAC)-1] - sixHourAC[len(sixHourAC)-2]  > sixHourAC[len(sixHourAC)-2] - sixHourAC[len(sixHourAC)-3] &&
			sixHourRsi[len(sixHourRsi)-1] < 55 &&
			rsi[len(rsi) -1] < 50 &&
			hma1 < 0 && hma2 > 0 {
			buyableCoins = append(buyableCoins, p.ID)
		}
	}
		fmt.Println(buyableCoins)
}
func bitcoinTest(client *coinbasepro.Client) {
	//One Hour
	candles, err := client.GetHistoricRates("BTC-USD", coinbasepro.GetHistoricRatesParams{
		Start:       time.Now().AddDate(0, 0, -3),
		End:         time.Now(),
		Granularity: 3600,
	})
	if err != nil {
		log.Println(err)
	}
	//Flip candles
	for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
		candles[i], candles[j] = candles[j], candles[i]
	}
	//Analysis of coin
	oneHourhmas := indicators.HMA(candles, 6)
	btcHourSlope1 := oneHourhmas[len(oneHourhmas)-1] - oneHourhmas[len(oneHourhmas)-2]
	btcHourSlope2 := oneHourhmas[len(oneHourhmas)-2] - oneHourhmas[len(oneHourhmas)-3]
	fmt.Println(btcHourSlope2, btcHourSlope1)

}
func AccountTest (client *coinbasepro.Client){
	accounts, _ := client.GetOrder("6274330")
	fmt.Println(accounts)
}
