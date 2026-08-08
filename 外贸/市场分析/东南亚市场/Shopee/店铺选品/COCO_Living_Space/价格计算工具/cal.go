package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
)

const (
	// 1 CNY = 0.61 MYR
	MarketExchangeRate = 0.61

	// 佣金费率：18.36%
	CommissionRate = 0.1836

	// 活动服务费率 + 交易手续费率：共计 2%
	OtherFeeRate = 0.02
)

// CalculateStorePrice 计算马来西亚站点商品价格。
//
// 公式：
// 店铺商品价格 =
// (全球商品价格 × 市场汇率 × 站点调价比例 + 跨境物流成本)
// ÷ (1 - 佣金费率 - 活动服务费率 - 交易手续费率)
func CalculateStorePrice(
	globalPriceCNY float64,
	siteAdjustmentRate float64,
	logisticsCostMYR float64,
) (float64, error) {
	if globalPriceCNY < 0 {
		return 0, fmt.Errorf("全球商品价格不能小于 0")
	}

	if siteAdjustmentRate <= 0 {
		return 0, fmt.Errorf("站点调价比例必须大于 0")
	}

	if logisticsCostMYR < 0 {
		return 0, fmt.Errorf("跨境物流成本不能小于 0")
	}

	feeDenominator := 1 - CommissionRate - OtherFeeRate
	if feeDenominator <= 0 {
		return 0, fmt.Errorf("费率设置错误：总费率必须小于 100%%")
	}

	convertedGoodsPrice :=
		globalPriceCNY *
			MarketExchangeRate *
			siteAdjustmentRate

	storePrice :=
		(convertedGoodsPrice + logisticsCostMYR) /
			feeDenominator

	return storePrice, nil
}

// RoundUpToTwoDecimals 向上保留两位小数，防止四舍五入造成少收款。
func RoundUpToTwoDecimals(value float64) float64 {
	return math.Ceil(value*100) / 100
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	var globalPriceCNY float64
	var adjustmentPercent float64
	var logisticsCostMYR float64

	fmt.Println("Shopee 马来西亚站商品价格计算器")
	fmt.Println("--------------------------------")

	fmt.Print("请输入全球商品价格（CNY）：")
	if _, err := fmt.Fscan(reader, &globalPriceCNY); err != nil {
		fmt.Println("输入错误：全球商品价格必须是数字")
		return
	}

	fmt.Print("请输入站点调价比例（例如 120 表示 120%）：")
	if _, err := fmt.Fscan(reader, &adjustmentPercent); err != nil {
		fmt.Println("输入错误：站点调价比例必须是数字")
		return
	}

	fmt.Print("请输入跨境物流成本/藏价（MYR）：")
	if _, err := fmt.Fscan(reader, &logisticsCostMYR); err != nil {
		fmt.Println("输入错误：跨境物流成本必须是数字")
		return
	}

	siteAdjustmentRate := adjustmentPercent / 100

	storePrice, err := CalculateStorePrice(
		globalPriceCNY,
		siteAdjustmentRate,
		logisticsCostMYR,
	)
	if err != nil {
		fmt.Println("计算失败：", err)
		return
	}

	adjustedGoodsPrice :=
		globalPriceCNY *
			MarketExchangeRate *
			siteAdjustmentRate

	suggestedPrice := RoundUpToTwoDecimals(storePrice)

	fmt.Println("\n计算结果")
	fmt.Println("--------------------------------")
	fmt.Printf("市场汇率：1 CNY = %.2f MYR\n", MarketExchangeRate)
	fmt.Printf("站点调价比例：%.2f%%\n", adjustmentPercent)
	fmt.Printf("佣金费率：%.2f%%\n", CommissionRate*100)
	fmt.Printf("其他手续费率：%.2f%%\n", OtherFeeRate*100)
	fmt.Printf("换算及调价后的商品金额：%.2f MYR\n", adjustedGoodsPrice)
	fmt.Printf("跨境物流藏价：%.2f MYR\n", logisticsCostMYR)
	fmt.Printf("计算价格：%.4f MYR\n", storePrice)
	fmt.Printf("建议店铺售价：%.2f MYR\n", suggestedPrice)
}
