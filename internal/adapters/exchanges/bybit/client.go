package bybit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/exchanges"
)

const (
	baseURL        = "https://api.bybit.com"
	testnetURL     = "https://api-testnet.bybit.com"
	defaultTimeout = 10 * time.Second
	defaultRecvWin = 5 * time.Second
)

// Config configures the Bybit client.
type Config struct {
	APIKey    string
	SecretKey string
	Market    exchanges.MarketType
	Testnet   bool

	HTTPClient *http.Client
	RecvWindow time.Duration
}

// NewClient creates a new Bybit adapter instance.
func NewClient(cfg Config) (exchanges.Exchange, error) {
	if cfg.RecvWindow == 0 {
		cfg.RecvWindow = defaultRecvWin
	}
	if cfg.Market == "" {
		cfg.Market = exchanges.MarketTypeLinearPerp
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}

	return &client{
		cfg: cfg,
	}, nil
}

type client struct {
	cfg Config
}

func (c *client) Name() string {
	return "bybit"
}

func (c *client) GetTicker(ctx context.Context, symbol string) (*exchanges.Ticker, error) {
	var res struct {
		List []struct {
			Symbol        string `json:"symbol"`
			LastPrice     string `json:"lastPrice"`
			Bid1Price     string `json:"bid1Price"`
			Ask1Price     string `json:"ask1Price"`
			HighPrice24h  string `json:"highPrice24h"`
			LowPrice24h   string `json:"lowPrice24h"`
			Volume24h     string `json:"volume24h"`
			Turnover24h   string `json:"turnover24h"`
			Price24hPcnt  string `json:"price24hPcnt"`
			CloseTime     int64  `json:"closeTime"`
			PrevPrice24h  string `json:"prevPrice24h"`
			NextFundingTs int64  `json:"nextFundingTime"`
		} `json:"list"`
	}

	params := url.Values{
		"category": []string{c.category()},
		"symbol":   []string{normalizeSymbol(symbol)},
	}
	if err := c.publicGet(ctx, "/v5/market/tickers", params, &res); err != nil {
		return nil, err
	}
	if len(res.List) == 0 {
		return nil, fmt.Errorf("ticker not found for %s", symbol)
	}
	item := res.List[0]
	return &exchanges.Ticker{
		Symbol:       item.Symbol,
		LastPrice:    dec(item.LastPrice),
		BidPrice:     dec(item.Bid1Price),
		AskPrice:     dec(item.Ask1Price),
		High24h:      dec(item.HighPrice24h),
		Low24h:       dec(item.LowPrice24h),
		VolumeBase:   dec(item.Volume24h),
		VolumeQuote:  dec(item.Turnover24h),
		Change24hPct: dec(item.Price24hPcnt).Mul(decimal.NewFromInt(100)),
		Timestamp:    time.UnixMilli(item.CloseTime),
	}, nil
}

func (c *client) GetOrderBook(ctx context.Context, symbol string, depth int) (*exchanges.OrderBook, error) {
	if depth <= 0 {
		depth = 50
	}
	var res struct {
		Bids [][]string `json:"b"`
		Asks [][]string `json:"a"`
		Ts   string     `json:"ts"`
	}

	params := url.Values{
		"category": []string{c.category()},
		"symbol":   []string{normalizeSymbol(symbol)},
		"limit":    []string{strconv.Itoa(depth)},
	}
	if err := c.publicGet(ctx, "/v5/market/orderbook", params, &res); err != nil {
		return nil, err
	}

	book := &exchanges.OrderBook{
		Symbol:    normalizeSymbol(symbol),
		Timestamp: time.UnixMilli(parseInt64(res.Ts)),
	}
	for _, bid := range res.Bids {
		book.Bids = append(book.Bids, exchanges.OrderBookEntry{
			Price:  decIdx(bid, 0),
			Amount: decIdx(bid, 1),
		})
	}
	for _, ask := range res.Asks {
		book.Asks = append(book.Asks, exchanges.OrderBookEntry{
			Price:  decIdx(ask, 0),
			Amount: decIdx(ask, 1),
		})
	}
	return book, nil
}

func (c *client) GetOHLCV(ctx context.Context, symbol string, timeframe string, limit int) ([]exchanges.OHLCV, error) {
	if limit <= 0 {
		limit = 200
	}
	var res struct {
		List [][]string `json:"list"`
	}

	params := url.Values{
		"category": []string{c.category()},
		"symbol":   []string{normalizeSymbol(symbol)},
		"interval": []string{mapInterval(timeframe)},
		"limit":    []string{strconv.Itoa(limit)},
	}
	if err := c.publicGet(ctx, "/v5/market/kline", params, &res); err != nil {
		return nil, err
	}

	candles := make([]exchanges.OHLCV, 0, len(res.List))
	for _, row := range res.List {
		if len(row) < 7 {
			continue
		}
		openTime := parseInt64(row[0])
		candles = append(candles, exchanges.OHLCV{
			Symbol:    normalizeSymbol(symbol),
			Timeframe: timeframe,
			OpenTime:  time.UnixMilli(openTime),
			CloseTime: time.UnixMilli(openTime + intervalDurationMs(timeframe)),
			Open:      dec(row[1]),
			High:      dec(row[2]),
			Low:       dec(row[3]),
			Close:     dec(row[4]),
			Volume:    dec(row[5]),
		})
	}
	return candles, nil
}

func (c *client) GetTrades(ctx context.Context, symbol string, limit int) ([]exchanges.Trade, error) {
	if limit <= 0 {
		limit = 100
	}
	var res struct {
		List []struct {
			ExecID    string `json:"execId"`
			Side      string `json:"side"`
			Price     string `json:"price"`
			Size      string `json:"size"`
			Timestamp string `json:"time"`
		} `json:"list"`
	}

	params := url.Values{
		"category": []string{c.category()},
		"symbol":   []string{normalizeSymbol(symbol)},
		"limit":    []string{strconv.Itoa(limit)},
	}
	if err := c.publicGet(ctx, "/v5/market/recent-trade", params, &res); err != nil {
		return nil, err
	}

	trades := make([]exchanges.Trade, 0, len(res.List))
	for _, t := range res.List {
		trades = append(trades, exchanges.Trade{
			ID:        t.ExecID,
			Symbol:    normalizeSymbol(symbol),
			Price:     dec(t.Price),
			Amount:    dec(t.Size),
			Side:      sideFromString(t.Side),
			Timestamp: time.UnixMilli(parseInt64(t.Timestamp)),
		})
	}
	return trades, nil
}

func (c *client) GetFundingRate(ctx context.Context, symbol string) (*exchanges.FundingRate, error) {
	if c.category() != "linear" && c.category() != "inverse" {
		return nil, exchanges.ErrNotSupported
	}
	var res struct {
		List []struct {
			Symbol    string `json:"symbol"`
			Funding   string `json:"fundingRate"`
			Timestamp string `json:"fundingRateTimestamp"`
			NextTime  string `json:"nextFundingTime"`
		} `json:"list"`
	}

	params := url.Values{
		"category": []string{c.category()},
		"symbol":   []string{normalizeSymbol(symbol)},
	}
	if err := c.publicGet(ctx, "/v5/market/funding/history", params, &res); err != nil {
		return nil, err
	}
	if len(res.List) == 0 {
		return nil, fmt.Errorf("funding rate not found")
	}
	item := res.List[0]
	return &exchanges.FundingRate{
		Symbol:    item.Symbol,
		Rate:      dec(item.Funding),
		Timestamp: time.UnixMilli(parseInt64(item.Timestamp)),
		NextTime:  time.UnixMilli(parseInt64(item.NextTime)),
	}, nil
}

func (c *client) GetOpenInterest(ctx context.Context, symbol string) (*exchanges.OpenInterest, error) {
	if c.category() != "linear" && c.category() != "inverse" {
		return nil, exchanges.ErrNotSupported
	}
	var res struct {
		List []struct {
			OpenInterest string `json:"openInterest"`
			Timestamp    string `json:"timestamp"`
		} `json:"list"`
	}

	params := url.Values{
		"category":     []string{c.category()},
		"symbol":       []string{normalizeSymbol(symbol)},
		"intervalTime": []string{"5"},
	}
	if err := c.publicGet(ctx, "/v5/market/open-interest", params, &res); err != nil {
		return nil, err
	}
	if len(res.List) == 0 {
		return nil, fmt.Errorf("open interest not found")
	}
	item := res.List[0]
	return &exchanges.OpenInterest{
		Symbol:    normalizeSymbol(symbol),
		Amount:    dec(item.OpenInterest),
		Timestamp: time.UnixMilli(parseInt64(item.Timestamp)),
	}, nil
}

func (c *client) GetBalance(ctx context.Context) (*exchanges.Balance, error) {
	if err := c.ensureCredentials(); err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"accountType": "UNIFIED",
	}
	var res struct {
		List []struct {
			TotalEquity string `json:"totalEquity"`
			TotalAvail  string `json:"totalAvailableBalance"`
			Coin        []struct {
				Coin                string `json:"coin"`
				WalletBalance       string `json:"walletBalance"`
				AvailableToWithdraw string `json:"availableToWithdraw"`
				BorrowAmount        string `json:"borrowAmount"`
			} `json:"coin"`
		} `json:"list"`
	}
	if err := c.privateRequest(ctx, http.MethodPost, "/v5/account/wallet-balance", nil, body, &res); err != nil {
		return nil, err
	}
	if len(res.List) == 0 {
		return nil, fmt.Errorf("wallet response empty")
	}
	entry := res.List[0]
	detail := make([]exchanges.BalanceDetail, 0, len(entry.Coin))
	for _, coin := range entry.Coin {
		detail = append(detail, exchanges.BalanceDetail{
			Currency:  coin.Coin,
			Total:     dec(coin.WalletBalance),
			Available: dec(coin.AvailableToWithdraw),
			Borrowed:  dec(coin.BorrowAmount),
		})
	}
	return &exchanges.Balance{
		Total:     dec(entry.TotalEquity),
		Available: dec(entry.TotalAvail),
		Currency:  "USD",
		Details:   detail,
	}, nil
}

func (c *client) GetPositions(ctx context.Context) ([]exchanges.Position, error) {
	if err := c.ensureCredentials(); err != nil {
		return nil, err
	}
	params := url.Values{
		"category": []string{c.category()},
	}
	var res struct {
		List []struct {
			Symbol          string `json:"symbol"`
			Side            string `json:"side"`
			Size            string `json:"size"`
			EntryPrice      string `json:"entryPrice"`
			MarkPrice       string `json:"markPrice"`
			LiqPrice        string `json:"liqPrice"`
			PositionValue   string `json:"positionValue"`
			Leverage        string `json:"leverage"`
			UnrealizedPnL   string `json:"unrealizedPnl"`
			TakeProfit      string `json:"takeProfit"`
			StopLoss        string `json:"stopLoss"`
			TrailingStop    string `json:"trailingStop"`
			PositionIdx     int    `json:"positionIdx"`
			Timestamp       string `json:"updatedTime"`
			MarginMode      string `json:"tradeMode"`
			PositionBalance string `json:"positionBalance"`
		} `json:"list"`
	}
	if err := c.privateRequest(ctx, http.MethodGet, "/v5/position/list", params, nil, &res); err != nil {
		return nil, err
	}

	positions := make([]exchanges.Position, 0, len(res.List))
	for _, p := range res.List {
		size := dec(p.Size)
		if size.IsZero() {
			continue
		}
		side := positionSideFromString(p.Side)
		positions = append(positions, exchanges.Position{
			Symbol:           p.Symbol,
			Market:           c.marketType(),
			Side:             side,
			Size:             size.Abs(),
			EntryPrice:       dec(p.EntryPrice),
			MarkPrice:        dec(p.MarkPrice),
			LiquidationPrice: dec(p.LiqPrice),
			MarginMode:       marginModeFromInt(p.MarginMode),
			Leverage:         dec(p.Leverage),
			UnrealizedPnL:    dec(p.UnrealizedPnL),
			UpdatedAt:        time.UnixMilli(parseInt64(p.Timestamp)),
		})
	}
	return positions, nil
}

func (c *client) GetOpenOrders(ctx context.Context, symbol string) ([]exchanges.Order, error) {
	if err := c.ensureCredentials(); err != nil {
		return nil, err
	}
	params := url.Values{
		"category": []string{c.category()},
	}
	if symbol != "" {
		params.Set("symbol", normalizeSymbol(symbol))
	}
	var res struct {
		List []struct {
			OrderID     string `json:"orderId"`
			OrderLinkID string `json:"orderLinkId"`
			Symbol      string `json:"symbol"`
			Side        string `json:"side"`
			OrderType   string `json:"orderType"`
			Qty         string `json:"qty"`
			Price       string `json:"price"`
			AvgPrice    string `json:"avgPrice"`
			TimeInForce string `json:"timeInForce"`
			Status      string `json:"orderStatus"`
			ReduceOnly  bool   `json:"reduceOnly"`
			RemainQty   string `json:"leavesQty"`
			CumExecQty  string `json:"cumExecQty"`
			CreatedTime string `json:"createdTime"`
			UpdatedTime string `json:"updatedTime"`
		} `json:"list"`
	}
	if err := c.privateRequest(ctx, http.MethodGet, "/v5/order/realtime", params, nil, &res); err != nil {
		return nil, err
	}

	orders := make([]exchanges.Order, 0, len(res.List))
	for _, o := range res.List {
		orders = append(orders, exchanges.Order{
			ID:            o.OrderID,
			ClientOrderID: o.OrderLinkID,
			Symbol:        o.Symbol,
			Market:        c.marketType(),
			Type:          orderTypeFromString(o.OrderType),
			Side:          sideFromString(o.Side),
			Status:        orderStatusFromString(o.Status),
			Price:         dec(o.Price),
			Quantity:      dec(o.Qty),
			Filled:        dec(o.CumExecQty),
			AvgFillPrice:  dec(o.AvgPrice),
			TimeInForce:   tifFromString(o.TimeInForce),
			CreatedAt:     time.UnixMilli(parseInt64(o.CreatedTime)),
			UpdatedAt:     time.UnixMilli(parseInt64(o.UpdatedTime)),
			ReduceOnly:    o.ReduceOnly,
		})
	}
	return orders, nil
}

func (c *client) PlaceOrder(ctx context.Context, req *exchanges.OrderRequest) (*exchanges.Order, error) {
	if err := c.ensureCredentials(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, exchanges.ErrInvalidRequest
	}

	payload := map[string]interface{}{
		"category":    c.category(),
		"symbol":      normalizeSymbol(req.Symbol),
		"side":        strings.ToUpper(string(req.Side)),
		"orderType":   mapOrderType(req.Type),
		"qty":         req.Quantity.String(),
		"timeInForce": string(fallbackTIF(req.TimeInForce, exchanges.TimeInForceGTC)),
		"reduceOnly":  req.ReduceOnly,
	}
	if !req.Price.IsZero() {
		payload["price"] = req.Price.String()
	}
	if !req.StopPrice.IsZero() {
		payload["triggerPrice"] = req.StopPrice.String()
		payload["triggerBy"] = "LastPrice"
	}
	if req.ClientOrderID != "" {
		payload["orderLinkId"] = req.ClientOrderID
	} else if req.Tag != "" {
		payload["orderLinkId"] = req.Tag
	}

	var res struct {
		OrderID     string `json:"orderId"`
		OrderLinkID string `json:"orderLinkId"`
	}

	if err := c.privateRequest(ctx, http.MethodPost, "/v5/order/create", nil, payload, &res); err != nil {
		return nil, err
	}

	return &exchanges.Order{
		ID:            res.OrderID,
		ClientOrderID: res.OrderLinkID,
		Symbol:        normalizeSymbol(req.Symbol),
		Market:        c.marketType(),
		Type:          req.Type,
		Side:          req.Side,
		Status:        exchanges.OrderStatusNew,
		Price:         req.Price,
		Quantity:      req.Quantity,
		TimeInForce:   fallbackTIF(req.TimeInForce, exchanges.TimeInForceGTC),
		ReduceOnly:    req.ReduceOnly,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (c *client) CancelOrder(ctx context.Context, symbol, orderID string) error {
	if err := c.ensureCredentials(); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"category": c.category(),
		"symbol":   normalizeSymbol(symbol),
	}
	if orderID != "" {
		payload["orderId"] = orderID
	}

	return c.privateRequest(ctx, http.MethodPost, "/v5/order/cancel", nil, payload, nil)
}

func (c *client) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if err := c.ensureCredentials(); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"category":     c.category(),
		"symbol":       normalizeSymbol(symbol),
		"buyLeverage":  strconv.Itoa(leverage),
		"sellLeverage": strconv.Itoa(leverage),
	}
	return c.privateRequest(ctx, http.MethodPost, "/v5/position/set-leverage", nil, payload, nil)
}

func (c *client) SetMarginMode(ctx context.Context, symbol string, mode exchanges.MarginMode) error {
	if err := c.ensureCredentials(); err != nil {
		return err
	}
	tradeMode := 0
	if mode == exchanges.MarginIsolated {
		tradeMode = 1
	}
	payload := map[string]interface{}{
		"category":  c.category(),
		"symbol":    normalizeSymbol(symbol),
		"tradeMode": tradeMode,
	}
	return c.privateRequest(ctx, http.MethodPost, "/v5/position/switch-isolated", nil, payload, nil)
}

func (c *client) publicGet(ctx context.Context, path string, params url.Values, target interface{}) error {
	_, body, err := c.doRequest(ctx, http.MethodGet, path, params, nil, false)
	if err != nil {
		return err
	}
	return decodeResponse(body, target)
}

func (c *client) privateRequest(ctx context.Context, method, path string, params url.Values, payload map[string]interface{}, target interface{}) error {
	c.ensureCredentials()
	_, body, err := c.doRequest(ctx, method, path, params, payload, true)
	if err != nil {
		return err
	}
	return decodeResponse(body, target)
}

func (c *client) doRequest(ctx context.Context, method, path string, params url.Values, payload map[string]interface{}, signed bool) (*http.Response, []byte, error) {
	var body io.Reader
	var bodyString string
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
		bodyString = string(raw)
		body = strings.NewReader(bodyString)
	}

	reqURL := c.baseURL() + path
	if method == http.MethodGet && params != nil && len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if signed {
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		recv := strconv.FormatInt(c.cfg.RecvWindow.Milliseconds(), 10)
		signature := c.sign(ts, recv, bodyString)

		if params != nil && len(params) > 0 && method != http.MethodGet {
			req.URL.RawQuery = params.Encode()
		}

		req.Header.Set("X-BAPI-API-KEY", c.cfg.APIKey)
		req.Header.Set("X-BAPI-SIGN", signature)
		req.Header.Set("X-BAPI-TIMESTAMP", ts)
		req.Header.Set("X-BAPI-RECV-WINDOW", recv)
	}

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode >= 400 {
		return resp, nil, fmt.Errorf("bybit http %d: %s", resp.StatusCode, string(respBody))
	}

	return resp, respBody, nil
}

func (c *client) sign(timestamp, recvWindow, body string) string {
	payload := timestamp + c.cfg.APIKey + recvWindow + body
	mac := hmac.New(sha256.New, []byte(c.cfg.SecretKey))
	_, _ = mac.Write([]byte(payload))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (c *client) baseURL() string {
	if c.cfg.Testnet {
		return testnetURL
	}
	return baseURL
}

func (c *client) ensureCredentials() error {
	if c.cfg.APIKey == "" || c.cfg.SecretKey == "" {
		return fmt.Errorf("bybit private call requires api credentials")
	}
	return nil
}

func (c *client) category() string {
	switch c.cfg.Market {
	case exchanges.MarketTypeSpot:
		return "spot"
	case exchanges.MarketTypeInversePerp, exchanges.MarketTypeDeliveryFut:
		return "inverse"
	default:
		return "linear"
	}
}

func (c *client) marketType() exchanges.MarketType {
	return c.cfg.Market
}

type apiResponse struct {
	RetCode int             `json:"retCode"`
	RetMsg  string          `json:"retMsg"`
	Result  json.RawMessage `json:"result"`
	Time    int64           `json:"time"`
}

func decodeResponse(body []byte, target interface{}) error {
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if resp.RetCode != 0 {
		return fmt.Errorf("bybit error %d: %s", resp.RetCode, resp.RetMsg)
	}
	if target == nil || len(resp.Result) == 0 {
		return nil
	}
	return json.Unmarshal(resp.Result, target)
}

func dec(value string) decimal.Decimal {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func decIdx(values []string, idx int) decimal.Decimal {
	if idx >= len(values) {
		return decimal.Zero
	}
	return dec(values[idx])
}

func parseInt64(value string) int64 {
	i, _ := strconv.ParseInt(value, 10, 64)
	return i
}

func sideFromString(value string) exchanges.OrderSide {
	if strings.EqualFold(value, "sell") {
		return exchanges.OrderSideSell
	}
	return exchanges.OrderSideBuy
}

func positionSideFromString(value string) exchanges.PositionSide {
	if strings.EqualFold(value, "sell") || strings.EqualFold(value, "short") {
		return exchanges.PositionSideShort
	}
	return exchanges.PositionSideLong
}

func mapInterval(tf string) string {
	mapping := map[string]string{
		"1m":  "1",
		"3m":  "3",
		"5m":  "5",
		"15m": "15",
		"30m": "30",
		"1h":  "60",
		"2h":  "120",
		"4h":  "240",
		"6h":  "360",
		"12h": "720",
		"1d":  "D",
		"1w":  "W",
		"1M":  "M",
	}
	if v, ok := mapping[tf]; ok {
		return v
	}
	return tf
}

func intervalDurationMs(tf string) int64 {
	switch tf {
	case "3m":
		return int64(3 * time.Minute / time.Millisecond)
	case "5m":
		return int64(5 * time.Minute / time.Millisecond)
	case "15m":
		return int64(15 * time.Minute / time.Millisecond)
	case "30m":
		return int64(30 * time.Minute / time.Millisecond)
	case "1h":
		return int64(time.Hour / time.Millisecond)
	case "2h":
		return int64(2 * time.Hour / time.Millisecond)
	case "4h":
		return int64(4 * time.Hour / time.Millisecond)
	case "6h":
		return int64(6 * time.Hour / time.Millisecond)
	case "12h":
		return int64(12 * time.Hour / time.Millisecond)
	case "1d":
		return int64(24 * time.Hour / time.Millisecond)
	case "1w":
		return int64(7 * 24 * time.Hour / time.Millisecond)
	case "1M":
		return int64(30 * 24 * time.Hour / time.Millisecond)
	default:
		return int64(time.Minute / time.Millisecond)
	}
}

func mapOrderType(t exchanges.OrderType) string {
	switch t {
	case exchanges.OrderTypeMarket:
		return "Market"
	case exchanges.OrderTypeStopMarket:
		return "Market"
	case exchanges.OrderTypeStopLimit:
		return "Limit"
	default:
		return "Limit"
	}
}

func orderTypeFromString(value string) exchanges.OrderType {
	switch strings.ToUpper(value) {
	case "MARKET":
		return exchanges.OrderTypeMarket
	case "STOP", "STOP_LIMIT", "STOPLIMIT":
		return exchanges.OrderTypeStopLimit
	default:
		return exchanges.OrderTypeLimit
	}
}

func orderStatusFromString(value string) exchanges.OrderStatus {
	switch strings.ToUpper(value) {
	case "NEW":
		return exchanges.OrderStatusNew
	case "PARTIALLY_FILLED":
		return exchanges.OrderStatusPartial
	case "FILLED":
		return exchanges.OrderStatusFilled
	case "CANCELED":
		return exchanges.OrderStatusCanceled
	case "REJECTED":
		return exchanges.OrderStatusRejected
	case "EXPIRED":
		return exchanges.OrderStatusExpired
	default:
		return exchanges.OrderStatusUnknown
	}
}

func tifFromString(value string) exchanges.TimeInForce {
	switch strings.ToUpper(value) {
	case "IOC":
		return exchanges.TimeInForceIOC
	case "FOK":
		return exchanges.TimeInForceFOK
	default:
		return exchanges.TimeInForceGTC
	}
}

func marginModeFromInt(value string) exchanges.MarginMode {
	if value == "1" {
		return exchanges.MarginIsolated
	}
	return exchanges.MarginCross
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.ReplaceAll(symbol, "/", ""))
}

func fallbackTIF(value, def exchanges.TimeInForce) exchanges.TimeInForce {
	if value == "" {
		return def
	}
	return value
}
