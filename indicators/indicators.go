package indicators

import (
	"math"

	"github.com/preichenberger/go-coinbasepro/v2"
)

type AdxResult struct {
	Pdi float64
	Mdi float64
	Adx float64
}
type AtrResult struct {
	Tr   float64
	Atr  float64
	Atrp float64
}

func HMA(candles []coinbasepro.HistoricRate, period int) []float64 {
	var hmas []float64
	wmasShort := WMA(candles, period/2)
	wmasLong := WMA(candles, period)
	pSqrt := math.Round(math.Sqrt(float64(period)))

	remove := len(wmasShort) - len(wmasLong)
	wmasShort = wmasShort[remove:]

	var rawHMAs []float64
	for i := 0; i < len(wmasLong); i++ {
		rawHMA := (2 * wmasShort[i]) - wmasLong[i]
		rawHMAs = append(rawHMAs, rawHMA)
	}

	hmas = WMAFloatSlice(rawHMAs, int(pSqrt))
	return hmas
}
func WMA(candles []coinbasepro.HistoricRate, period int) []float64 {
	var wmas []float64
	k := float64((period * (period + 1)) / 2.0)
	for i := 0; i < len(candles)-period+1; i++ {
		product := 0.0
		for j := 0; j < period; j++ {
			product += candles[j+i].Close * float64(j+1)
		}
		wma := product / k
		wmas = append(wmas, wma)
	}
	return wmas
}
func WMAFloatSlice(hma []float64, period int) []float64 {
	var wmas []float64
	k := float64((period * (period + 1)) / 2.0)
	for i := 0; i < len(hma)-period+1; i++ {
		product := 0.0
		for j := 0; j < period; j++ {
			product += hma[j+i] * float64(j+1)
		}
		wma := product / k
		wmas = append(wmas, wma)
	}
	return wmas
}
func StochasticRSI(rsi []float64, period int) []float64 {
	var sRSI []float64
	for i := period; i < len(rsi)-1; i++ {
		min := minOverPeriod(rsi[i-period : i])
		max := maxOverPeriod(rsi[i-period : i])

		s := (rsi[i-1] - min) / (max - min)
		sRSI = append(sRSI, s*100)
	}
	return sRSISmoothing(sRSI, 3)
}
func minOverPeriod(data []float64) float64 {
	min := 100.0
	for _, d := range data {
		if d < min {
			min = d
		}
	}
	return min
}
func maxOverPeriod(data []float64) float64 {
	max := 0.0
	for _, d := range data {
		if d > max {
			max = d
		}
	}
	return max
}
func sRSISmoothing(data []float64, period int) []float64 {
	var smaSlice []float64

	for i := period; i <= len(data); i++ {
		smaSlice = append(smaSlice, SumFloat(data[i-period:i])/float64(period))
	}

	return smaSlice
}
func SumFloat(data []float64) float64 {

	var sum float64

	for _, x := range data {
		sum += x
	}

	return sum
}
func RSI(candles []coinbasepro.HistoricRate, period int) []float64 {
	var RSIs []float64

	for i := 1; i < len(candles)-period+1; i++ {
		rsi := 0.0
		totalGain := 0.0
		totalLoss := 0.0
		for j := 0; j < period; j++ {

			previous := candles[j+i-1].Close
			current := candles[j+i].Close

			difference := current - previous

			if difference >= 0 {
				totalGain += difference
			} else {
				totalLoss -= difference
			}
		}
		rs := totalGain / math.Abs(totalLoss)
		rsi = 100 - (100 / (1 + rs))
		RSIs = append(RSIs, rsi)
	}
	return RSIs
}

func EMAOverSMA(candles []coinbasepro.HistoricRate, period int) bool {
	ema := CalcEMA(candles, period)
	sma := CalcSMA(candles, period)
	return ema[len(ema)-2] > sma[len(sma)-2]
}
func IncreasingEmaSlope(candles []coinbasepro.HistoricRate, period int) float64 {
	ema := CalcEMA(candles, period)

	a := ema[len(ema)-4]
	b := ema[len(ema)-3]
	c := ema[len(ema)-2]

	return (a - b) - (b - c)

}
func EMASlopeOver0(candles []coinbasepro.HistoricRate, period int) bool {
	ema := CalcEMA(candles, period)
	a := ema[len(ema)-3]
	b := ema[len(ema)-2]
	return b-a > 0
}
func CalcEMA(candles []coinbasepro.HistoricRate, period int) []float64 {
	var ema []float64

	l := float64(period) + 1
	m := 2 / l
	k := m

	ema = append(ema, candles[0].Close)

	for i := 1; i < len(candles); i++ {
		ema = append(ema, (candles[i].Close*k)+(ema[i-1]*(1-k)))
	}
	return ema
}
func CalcSMA(candles []coinbasepro.HistoricRate, period int) []float64 {
	var smaSlice []float64

	for i := period; i <= len(candles); i++ {
		smaSlice = append(smaSlice, Sum(candles[i-14:i])/float64(14))
	}

	return smaSlice
}

// Accumulation Distribution
func CalcAccumulationDistribution(candles []coinbasepro.HistoricRate) []float64 {
	ad := make([]float64, len(candles))

	for i := 0; i < len(candles); i++ {
		if i > 0 {
			ad[i] = ad[i-1]
		}

		ad[i] += candles[i].Volume * (((candles[i].Close - candles[i].Low) - (candles[i].High - candles[i].Close)) / (candles[i].High - candles[i].Low))
	}

	return ad
}

// Sum returns the sum of all elements of 'data'.
func Sum(candles []coinbasepro.HistoricRate) float64 {

	var sum float64

	for _, x := range candles {
		sum += x.Close
	}

	return sum
}
func CalcAdx(bars []coinbasepro.HistoricRate) []AdxResult {
	var start = len(bars) - 30
	if start < 0 {
		return []AdxResult{
			AdxResult{
				Pdi: 0,
				Mdi: 0,
				Adx: 0,
			},
		}
	}
	bars = bars[start:]
	var prevHigh float64
	var prevLow float64
	var prevTrs float64
	var prevPdm float64
	var prevMdm float64
	var prevAdx float64
	var sumTr float64
	var sumPdm float64
	var sumMdm float64
	var sumDx float64
	var lookBackPeriod = 14
	var results []AdxResult
	var atrResults = CalcTrueRangeHr(bars)

	for i := 0; i < len(bars); i++ {
		b := bars[i]
		index := i + 1
		result := AdxResult{}

		if index == 1 {
			results = append(results, result)
			prevHigh = b.High
			prevLow = b.Low
			continue
		}

		var tr = atrResults[i].Tr

		var pdm1 float64
		var mdm1 float64
		if (b.High - prevHigh) > (prevLow - b.Low) {
			pdm1 = math.Max(b.High-prevHigh, 0)
		} else {
			pdm1 = 0
		}

		if (prevLow - b.Low) > (b.High - prevHigh) {
			mdm1 = math.Max(prevLow-b.Low, 0)
		} else {
			mdm1 = 0
		}

		prevHigh = b.High
		prevLow = b.Low

		if index <= lookBackPeriod+1 {
			sumTr += tr
			sumPdm += pdm1
			sumMdm += mdm1
		}

		if index <= lookBackPeriod {
			results = append(results, result)
			continue
		}

		var trs float64
		var pdm float64
		var mdm float64

		if index == lookBackPeriod+1 {
			trs = sumTr
			pdm = sumPdm
			mdm = sumMdm
		} else {
			trs = prevTrs - (prevTrs / float64(lookBackPeriod)) + tr
			pdm = prevPdm - (prevPdm / float64(lookBackPeriod)) + pdm1
			mdm = prevMdm - (prevMdm / float64(lookBackPeriod)) + mdm1
		}

		prevTrs = trs
		prevPdm = pdm
		prevMdm = mdm

		var pdi = 100 * pdm / trs
		var mdi = 100 * mdm / trs
		var dx = 100 * math.Abs((pdi-mdi)/(pdi+mdi))

		result.Pdi = pdi
		result.Mdi = mdi

		var adx float64

		if index > 2*lookBackPeriod {
			adx = (prevAdx*(float64(lookBackPeriod)-1) + dx) / float64(lookBackPeriod)
			result.Adx = adx
			prevAdx = adx
		} else if index == 2*lookBackPeriod {
			sumDx += dx
			adx = sumDx / float64(lookBackPeriod)
			result.Adx = adx
			prevAdx = adx
		} else {
			sumDx += dx
		}
		if math.IsNaN(adx) {
			return []AdxResult{
				AdxResult{
					Pdi: 0,
					Mdi: 0,
					Adx: 0,
				},
			}
		}
		results = append(results, result)
	}

	return results
}
func CalcTrueRangeHr(bars []coinbasepro.HistoricRate) []AtrResult {
	var prevAtr float64
	var prevClose float64
	var highMinusPrevClose float64
	var lowMinusPrevClose float64
	var sumTr float64
	var lookbackPeriod float64 = 14
	var results []AtrResult

	for i := 0; i < len(bars); i++ {
		h := bars[i]
		var index float64 = float64(i) + 1

		var result AtrResult

		if index > 1 {
			highMinusPrevClose = math.Abs(h.High - prevClose)
			lowMinusPrevClose = math.Abs(h.Low - prevClose)
		}

		tr := math.Max(h.High-h.Low, math.Max(highMinusPrevClose, lowMinusPrevClose))
		result.Tr = tr

		if index > lookbackPeriod {
			// calculate ATR
			result.Atr = (prevAtr*(lookbackPeriod-1) + tr) / lookbackPeriod
			if h.Close == 0 {
				result.Atrp = 0
			} else {
				result.Atrp = (result.Atr / h.Close) * 100
			}
			prevAtr = result.Atr
		} else if index == lookbackPeriod {
			// initialize ATR
			sumTr += tr
			result.Atr = sumTr / lookbackPeriod
			if h.Close == 0 {
				result.Atrp = 0
			} else {
				result.Atrp = (result.Atr / h.Close) * 100
			}
			prevAtr = result.Atr
		} else {
			// only used for periods before ATR initialization
			sumTr += tr
		}

		results = append(results, result)
		prevClose = h.Close
	}
	return results
}
