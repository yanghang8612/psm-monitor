package monitor

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"psm-monitor/misc"
	"psm-monitor/net"
	"psm-monitor/slack"
)

type Record struct {
	ID                 uint `gorm:"primaryKey"`
	TrackedAt          time.Time
	TronLowPrice       float64
	TronHighPrice      float64
	EthLowPrice        float64
	EthHighPrice       float64
	BscLowPrice        float64
	BscHighPrice       float64
	PolygonLowPrice    float64
	PolygonHighPrice   float64
	AvalancheLowPrice  float64
	AvalancheHighPrice float64
	SolanaLowPrice     float64
	SolanaHighPrice    float64
}

var appDB *gorm.DB

func StartTrackFee(c *cron.Cron) {
	_ = c.AddFunc("0 */1 * * * ?", misc.WrapLog(track))
	_ = c.AddFunc("30 0 2 * * ?", misc.WrapLog(report))

	db, err := gorm.Open(sqlite.Open("monitor.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&Record{})

	appDB = db
}

func ReportFee() {
	report()
}

func track() {
	trxPrice := net.GetPrice("TRX")
	energyPrice, factor := net.GetEnergyPriceAndFactor()

	ethPrice := net.GetPrice("ETH")
	ethGasPrice := net.GetGasPrice("Ethereum")

	bnbPrice := net.GetPrice("BNB")
	bscGasPrice := net.GetGasPrice("BSC")

	polPrice := net.GetPrice("POL")
	polGasPrice := net.GetGasPrice("Polygon")

	avaxPrice := net.GetPrice("AVAX")
	avaxGasPrice := net.GetAvalanchePrice()

	solPrice := net.GetSolPrice()

	tronLowPrice := trxPrice * energyPrice * (1 + factor/1e4) * 14650 / 1e6
	tronHighPrice := trxPrice * energyPrice * (1 + factor/1e4) * 29650 / 1e6

	ethLowPrice := ethPrice * ethGasPrice * 41309 / 1e9
	ethHighPrice := ethPrice * ethGasPrice * 63209 / 1e9

	bscLowPrice := bnbPrice * bscGasPrice * 34515 / 1e9
	bscHighPrice := bnbPrice * bscGasPrice * 51627 / 1e9

	polygonLowPrice := polPrice * polGasPrice * 35394 / 1e9
	polygonHighPrice := polPrice * polGasPrice * 57306 / 1e9

	avalancheLowPrice := avaxPrice * avaxGasPrice * 44038 / 1e9
	avalancheHighPrice := avaxPrice * avaxGasPrice * 61138 / 1e9

	solanaLowPrice := solPrice * 10 * 17000 / 1e9
	solanaHighPrice := solPrice * 10 * 40000 / 1e9

	appDB.Create(&Record{TrackedAt: time.Now(),
		TronLowPrice: tronLowPrice, TronHighPrice: tronHighPrice,
		EthLowPrice: ethLowPrice, EthHighPrice: ethHighPrice,
		BscLowPrice: bscLowPrice, BscHighPrice: bscHighPrice,
		PolygonLowPrice: polygonLowPrice, PolygonHighPrice: polygonHighPrice,
		AvalancheLowPrice: avalancheLowPrice, AvalancheHighPrice: avalancheHighPrice,
		SolanaLowPrice: solanaLowPrice, SolanaHighPrice: solanaHighPrice})
}

func report() {
	now := time.Now()

	var dayAvgRecord Record
	preDay := now.AddDate(0, 0, -1)
	appDB.Model(&Record{}).
		Select("AVG(tron_low_price) as tron_low_price, AVG(tron_high_price) as tron_high_price, "+
			"AVG(eth_low_price) as eth_low_price, AVG(eth_high_price) as eth_high_price, "+
			"AVG(bsc_low_price) as bsc_low_price, AVG(bsc_high_price) as bsc_high_price, "+
			"AVG(polygon_low_price) as polygon_low_price, AVG(polygon_high_price) as polygon_high_price, "+
			"AVG(avalanche_low_price) as avalanche_low_price, AVG(avalanche_high_price) as avalanche_high_price, "+
			"AVG(solana_low_price) as solana_low_price, AVG(solana_high_price) as solana_high_price").
		Where("tracked_at BETWEEN ? AND ?", preDay, now).Find(&dayAvgRecord)

	var weekAvgRecord Record
	preWeek := now.AddDate(0, 0, -7)
	appDB.Model(&Record{}).
		Select("AVG(tron_low_price) as tron_low_price, AVG(tron_high_price) as tron_high_price, "+
			"AVG(eth_low_price) as eth_low_price, AVG(eth_high_price) as eth_high_price, "+
			"AVG(bsc_low_price) as bsc_low_price, AVG(bsc_high_price) as bsc_high_price, "+
			"AVG(polygon_low_price) as polygon_low_price, AVG(polygon_high_price) as polygon_high_price, "+
			"AVG(avalanche_low_price) as avalanche_low_price, AVG(avalanche_high_price) as avalanche_high_price, "+
			"AVG(solana_low_price) as solana_low_price, AVG(solana_high_price) as solana_high_price").
		Where("tracked_at BETWEEN ? AND ?", preWeek, now).Find(&weekAvgRecord)

	slackMessage := ""
	slackMessage += fmt.Sprintf("USDT 日均手续费:\n"+
		"> TRON: `%.5f$` - `%.5f$`\n"+
		"> ETH: `%.5f$` - `%.5f$`\n"+
		"> BSC: `%.5f$` - `%.5f$`\n"+
		"> Polgon: `%.5f$` - `%.5f$`\n"+
		"> Avalanche: `%.5f$` - `%.5f$`\n"+
		"> Solana: `%.5f$` - `%.5f$`\n",
		dayAvgRecord.TronLowPrice, dayAvgRecord.TronHighPrice,
		dayAvgRecord.EthLowPrice, dayAvgRecord.EthHighPrice,
		dayAvgRecord.BscLowPrice, dayAvgRecord.BscHighPrice,
		dayAvgRecord.PolygonLowPrice, dayAvgRecord.PolygonHighPrice,
		dayAvgRecord.AvalancheLowPrice, dayAvgRecord.AvalancheHighPrice,
		dayAvgRecord.SolanaLowPrice, dayAvgRecord.SolanaHighPrice)

	slackMessage += fmt.Sprintf("USDT 周均手续费:\n"+
		"> TRON: `%.5f$` - `%.5f$`\n"+
		"> ETH: `%.5f$` - `%.5f$`\n"+
		"> BSC: `%.5f$` - `%.5f$`\n"+
		"> Polgon: `%.5f$` - `%.5f$`\n"+
		"> Avalanche: `%.5f$` - `%.5f$`\n"+
		"> Solana: `%.5f$` - `%.5f$`\n",
		weekAvgRecord.TronLowPrice, weekAvgRecord.TronHighPrice,
		weekAvgRecord.EthLowPrice, weekAvgRecord.EthHighPrice,
		weekAvgRecord.BscLowPrice, weekAvgRecord.BscHighPrice,
		weekAvgRecord.PolygonLowPrice, weekAvgRecord.PolygonHighPrice,
		weekAvgRecord.AvalancheLowPrice, weekAvgRecord.AvalancheHighPrice,
		weekAvgRecord.SolanaLowPrice, weekAvgRecord.SolanaHighPrice)

	slack.ReportFee(slackMessage)
}
