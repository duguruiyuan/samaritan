package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/miaolz123/conver"
	"github.com/miaolz123/samaritan/log"
)

// Huobi : the exchange struct of okcoin.cn
type Huobi struct {
	stockMap     map[string]string
	orderTypeMap map[string]int
	periodMap    map[string]string
	records      map[string][]Record
	host         string
	log          log.Logger
	option       Option
}

// NewHuobi : create an exchange struct of okcoin.cn
func NewHuobi(opt Option) *Huobi {
	e := Huobi{
		stockMap:     map[string]string{"BTC": "1", "LTC": "2"},
		orderTypeMap: map[string]int{"1": 1, "2": -1, "3": 2, "4": -2},
		periodMap:    map[string]string{"M": "001", "M5": "005", "M15": "015", "M30": "030", "H": "060", "D": "100", "W": "200"},
		records:      make(map[string][]Record),
		host:         "https://api.huobi.com/apiv3",
		log:          log.New(opt.Type),
		option:       opt,
	}
	if _, ok := e.stockMap[e.option.MainStock]; !ok {
		e.option.MainStock = "BTC"
	}
	return &e
}

// Log : print something to console
func (e *Huobi) Log(msgs ...interface{}) {
	e.log.Do("info", 0.0, 0.0, msgs...)
}

// GetMainStock : get the MainStock of this exchange
func (e *Huobi) GetMainStock() string {
	return e.option.MainStock
}

// SetMainStock : set the MainStock of this exchange
func (e *Huobi) SetMainStock(stock string) string {
	if _, ok := e.stockMap[stock]; ok {
		e.option.MainStock = stock
	}
	return e.option.MainStock
}

func (e *Huobi) getAuthJSON(url string, params []string, optionals ...string) (json *simplejson.Json, err error) {
	params = append(params, []string{
		"access_key=" + e.option.AccessKey,
		"secret_key=" + e.option.SecretKey,
		fmt.Sprint("created=", time.Now().Unix()),
	}...)
	sort.Strings(params)
	params = append(params, "sign="+signMd5(params))
	resp, err := post(url, append(params, optionals...))
	if err != nil {
		return
	}
	return simplejson.NewJson(resp)
}

// GetAccount : get the account detail of this exchange
func (e *Huobi) GetAccount() interface{} {
	params := []string{
		"method=get_account_info",
	}
	json, err := e.getAuthJSON(e.host, params, "market=cny")
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "GetAccount() error, ", err)
		return nil
	}
	if code := conver.IntMust(json.Get("code").Interface()); code > 0 {
		err = fmt.Errorf("GetAccount() error, the error number is %v", code)
		e.log.Do("error", 0.0, 0.0, err)
		return nil
	}
	account := Account{
		Total:         conver.Float64Must(json.Get("total").Interface()),
		Net:           conver.Float64Must(json.Get("net_asset").Interface()),
		Balance:       conver.Float64Must(json.Get("available_cny_display").Interface()),
		FrozenBalance: conver.Float64Must(json.Get("frozen_cny_display").Interface()),
		BTC:           conver.Float64Must(json.Get("available_btc_display").Interface()),
		FrozenBTC:     conver.Float64Must(json.Get("frozen_btc_display").Interface()),
		LTC:           conver.Float64Must(json.Get("available_ltc_display").Interface()),
		FrozenLTC:     conver.Float64Must(json.Get("frozen_ltc_display").Interface()),
	}
	switch e.option.MainStock {
	case "BTC":
		account.Stock = account.BTC
		account.FrozenStock = account.FrozenBTC
	case "LTC":
		account.Stock = account.LTC
		account.FrozenStock = account.FrozenLTC
	}
	return account
}

// Buy : buy stocks
func (e *Huobi) Buy(stockType string, price, amount float64, msgs ...interface{}) (id string) {
	if _, ok := e.stockMap[stockType]; !ok {
		e.log.Do("error", 0.0, 0.0, "Buy() error, unrecognized stockType: ", stockType)
		return
	}
	params := []string{
		"coin_type=" + e.stockMap[stockType],
		fmt.Sprint("amount=", amount),
	}
	methodParam := "method=buy_market"
	if price > 0 {
		methodParam = "method=buy"
		params = append(params, fmt.Sprint("price=", price))
	}
	params = append(params, methodParam)
	json, err := e.getAuthJSON(e.host, params, "market=cny")
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "Buy() error, ", err)
		return
	}
	if code := conver.IntMust(json.Get("code").Interface()); code > 0 {
		err = fmt.Errorf("Buy() error, the error number is %v", code)
		e.log.Do("error", 0.0, 0.0, err)
		return
	}
	e.log.Do("buy", price, amount, msgs...)
	id = fmt.Sprint(json.Get("id").Interface())
	return
}

// Sell : sell stocks
func (e *Huobi) Sell(stockType string, price, amount float64, msgs ...interface{}) (id string) {
	if _, ok := e.stockMap[stockType]; !ok {
		e.log.Do("error", 0.0, 0.0, "Sell() error, unrecognized stockType: ", stockType)
		return
	}
	params := []string{
		"coin_type=" + e.stockMap[stockType],
		fmt.Sprint("amount=", amount),
	}
	methodParam := "method=sell_market"
	if price > 0 {
		methodParam = "method=sell"
		params = append(params, fmt.Sprint("price=", price))
	}
	params = append(params, methodParam)
	json, err := e.getAuthJSON(e.host, params, "market=cny")
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "Sell() error, ", err)
		return
	}
	if code := conver.IntMust(json.Get("code").Interface()); code > 0 {
		err = fmt.Errorf("Sell() error, the error number is %v", code)
		e.log.Do("error", 0.0, 0.0, err)
		return
	}
	e.log.Do("sell", price, amount, msgs...)
	id = fmt.Sprint(json.Get("id").Interface())
	return
}

// GetOrder : get details of an order
func (e *Huobi) GetOrder(stockType, id string) interface{} {
	params := []string{
		"method=order_info",
		"coin_type=" + e.stockMap[stockType],
		"id=" + id,
	}
	json, err := e.getAuthJSON(e.host, params, "market=cny")
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "GetOrders() error, ", err)
		return nil
	}
	if code := conver.IntMust(json.Get("code").Interface()); code > 0 {
		err = fmt.Errorf("GetOrders() error, the error number is %v", code)
		e.log.Do("error", 0.0, 0.0, err)
		return nil
	}
	return Order{
		ID:         fmt.Sprint(json.Get("id").Interface()),
		Price:      json.Get("order_price").MustFloat64(),
		Amount:     json.Get("order_amount").MustFloat64(),
		DealAmount: json.Get("processed_amount").MustFloat64(),
		OrderType:  e.orderTypeMap[json.Get("type").MustString()],
		StockType:  stockType,
	}
}

// CancelOrder : cancel an order
func (e *Huobi) CancelOrder(order Order) bool {
	params := []string{
		"method=cancel_order",
		"coin_type=" + e.stockMap[order.StockType],
		"id=" + order.ID,
	}
	json, err := e.getAuthJSON(e.host, params, "market=cny")
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "CancelOrder() error, ", err)
		return false
	}
	if code := conver.IntMust(json.Get("code").Interface()); code > 0 {
		err = fmt.Errorf("CancelOrder() error, the error number is %v", code)
		e.log.Do("error", 0.0, 0.0, err)
		return false
	}
	if json.Get("result").MustString() == "success" {
		e.log.Do("cancel", 0.0, 0.0, fmt.Sprintf("%+v", order))
		return true
	}
	e.log.Do("error", 0.0, 0.0, "CancelOrder() error, ", json.Get("msg").Interface())
	return false
}

// GetOrders : get all unfilled orders
func (e *Huobi) GetOrders(stockType string) (orders []Order) {
	if _, ok := e.stockMap[stockType]; !ok {
		e.log.Do("error", 0.0, 0.0, "GetOrders() error, unrecognized stockType: ", stockType)
		return
	}
	params := []string{
		"method=get_orders",
		"coin_type=" + e.stockMap[stockType],
	}
	json, err := e.getAuthJSON(e.host+"order_info.do", params)
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "GetOrders() error, ", err)
		return
	}
	if code := conver.IntMust(json.Get("code").Interface()); code > 0 {
		err = fmt.Errorf("GetOrders() error, the error number is %v", code)
		e.log.Do("error", 0.0, 0.0, err)
		return
	}
	count := len(json.MustArray())
	for i := 0; i < count; i++ {
		orderJSON := json.GetIndex(i)
		orders = append(orders, Order{
			ID:         fmt.Sprint(orderJSON.Get("id").Interface()),
			Price:      orderJSON.Get("order_price").MustFloat64(),
			Amount:     orderJSON.Get("order_amount").MustFloat64(),
			DealAmount: orderJSON.Get("processed_amount").MustFloat64(),
			OrderType:  e.orderTypeMap[orderJSON.Get("type").MustString()],
			StockType:  stockType,
		})
	}
	return orders
}

// GetTrades : get all filled orders recently
func (e *Huobi) GetTrades(stockType string) (orders []Order) {
	if _, ok := e.stockMap[stockType]; !ok {
		e.log.Do("error", 0.0, 0.0, "GetTrades() error, unrecognized stockType: ", stockType)
		return
	}
	params := []string{
		"method=get_new_deal_orders",
		"coin_type=" + e.stockMap[stockType],
	}
	json, err := e.getAuthJSON(e.host+"order_history.do", params)
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "GetTrades() error, ", err)
		return
	}
	if code := conver.IntMust(json.Get("code").Interface()); code > 0 {
		err = fmt.Errorf("GetTrades() error, the error number is %v", code)
		e.log.Do("error", 0.0, 0.0, err)
		return
	}
	count := len(json.MustArray())
	for i := 0; i < count; i++ {
		orderJSON := json.GetIndex(i)
		orders = append(orders, Order{
			ID:         fmt.Sprint(orderJSON.Get("id").Interface()),
			Price:      orderJSON.Get("order_price").MustFloat64(),
			Amount:     orderJSON.Get("order_amount").MustFloat64(),
			DealAmount: orderJSON.Get("processed_amount").MustFloat64(),
			OrderType:  e.orderTypeMap[orderJSON.Get("type").MustString()],
			StockType:  stockType,
		})
	}
	return orders
}

// GetTicker : get market ticker & depth
func (e *Huobi) GetTicker(stockType string, sizes ...int) interface{} {
	if _, ok := e.stockMap[stockType]; !ok {
		e.log.Do("error", 0.0, 0.0, "GetTicker() error, unrecognized stockType: ", stockType)
		return nil
	}
	size := 20
	if len(sizes) > 0 && sizes[0] > 20 {
		size = sizes[0]
	}

	resp, err := get(fmt.Sprint("http://api.huobi.com/staticmarket/depth_", strings.ToLower(stockType), "_", size, ".js"))
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "GetTicker() error, ", err)
		return nil
	}
	json, err := simplejson.NewJson(resp)
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "GetTicker() error, ", err)
		return nil
	}
	ticker := Ticker{}
	depthsJSON := json.Get("bids")
	for i := 0; i < len(depthsJSON.MustArray()); i++ {
		depthJSON := depthsJSON.GetIndex(i)
		ticker.Bids = append(ticker.Bids, MarketOrder{
			Price:  depthJSON.GetIndex(0).MustFloat64(),
			Amount: depthJSON.GetIndex(1).MustFloat64(),
		})
	}
	depthsJSON = json.Get("asks")
	for i := 0; i < len(depthsJSON.MustArray()); i++ {
		depthJSON := depthsJSON.GetIndex(i)
		ticker.Asks = append(ticker.Asks, MarketOrder{
			Price:  depthJSON.GetIndex(0).MustFloat64(),
			Amount: depthJSON.GetIndex(1).MustFloat64(),
		})
	}
	if len(ticker.Bids) < 1 || len(ticker.Asks) < 1 {
		e.log.Do("error", 0.0, 0.0, "GetTicker() error, can not get enough Bids or Asks")
		return nil
	}
	ticker.Buy = ticker.Bids[0].Price
	ticker.Sell = ticker.Asks[0].Price
	ticker.Mid = (ticker.Buy + ticker.Sell) / 2
	return ticker
}

// GetRecords : get candlestick data
func (e *Huobi) GetRecords(stockType, period string, sizes ...int) (records []Record) {
	if _, ok := e.stockMap[stockType]; !ok {
		e.log.Do("error", 0.0, 0.0, "GetRecords() error, unrecognized stockType: ", stockType)
		return
	}
	if _, ok := e.periodMap[period]; !ok {
		e.log.Do("error", 0.0, 0.0, "GetRecords() error, unrecognized period: ", period)
		return
	}
	size := 200
	if len(sizes) > 0 {
		size = sizes[0]
	}
	resp, err := get(fmt.Sprint("http://api.huobi.com/staticmarket/", strings.ToLower(stockType), "_kline_", e.periodMap[period], "_json.js"))
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "GetRecords() error, ", err)
		return
	}
	json, err := simplejson.NewJson(resp)
	if err != nil {
		e.log.Do("error", 0.0, 0.0, "GetRecords() error, ", err)
		return
	}
	timeLast := int64(0)
	if len(e.records[period]) > 0 {
		timeLast = e.records[period][len(e.records[period])-1].Time
	}
	recordsNew := []Record{}
	for i := len(json.MustArray()); i > 0; i-- {
		recordJSON := json.GetIndex(i - 1)
		recordTime := conver.Int64Must(recordJSON.GetIndex(0).MustString("19700101000000000")[:12])
		if recordTime > timeLast {
			recordsNew = append(recordsNew, Record{
				Time:   recordTime,
				Open:   recordJSON.GetIndex(1).MustFloat64(),
				High:   recordJSON.GetIndex(2).MustFloat64(),
				Low:    recordJSON.GetIndex(3).MustFloat64(),
				Close:  recordJSON.GetIndex(4).MustFloat64(),
				Volume: recordJSON.GetIndex(5).MustFloat64(),
			})
		} else if recordTime == timeLast {
			e.records[period][len(e.records[period])-1] = Record{
				Time:   recordTime,
				Open:   recordJSON.GetIndex(1).MustFloat64(),
				High:   recordJSON.GetIndex(2).MustFloat64(),
				Low:    recordJSON.GetIndex(3).MustFloat64(),
				Close:  recordJSON.GetIndex(4).MustFloat64(),
				Volume: recordJSON.GetIndex(5).MustFloat64(),
			}
		} else {
			break
		}
	}
	e.records[period] = append(e.records[period], recordsNew...)
	if len(e.records[period]) > size {
		e.records[period] = e.records[period][:size]
	}
	fmt.Println(len(e.records[period]))
	return e.records[period]
}