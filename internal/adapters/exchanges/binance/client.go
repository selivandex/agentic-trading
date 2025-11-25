package binance

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
	spotBaseURL         = "https://api.binance.com"
	spotTestnetBaseURL  = "https://testnet.binance.vision"
	futuresBaseURL      = "https://fapi.binance.com"
	futuresTestnetURL   = "https://testnet.binancefuture.com"
	defaultRecvWindowMs = 5000
	defaultHTTPTimeout  = 10 * time.Second
)

// Config configures the Binance client.
type Config struct {
	APIKey    string
	SecretKey string
	Market    exchanges.MarketType
	Testnet   bool

	HTTPClient *http.Client
	RecvWindow time.Duration
}

// NewClient creates a new Binance adapter.
func NewClient(cfg Config) (exchanges.Exchange, error) {
	if cfg.APIKey == "" && cfg.SecretKey != "" {
		return nil, fmt.Errorf("api key required when secret key provided")
	}
	if cfg.SecretKey == "" && cfg.APIKey != "" {
		return nil, fmt.Errorf("secret key required when api key provided")
	}
	if cfg.Market == "" {
		cfg.Market = exchanges.MarketTypeSpot
	}
	if cfg.RecvWindow == 0 {
		cfg.RecvWindow = defaultRecvWindowMs * time.Millisecond
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &client{
		cfg:        cfg,
		httpClient: httpClient,
	}, nil
}

type client struct {
	cfg        Config
	httpClient *http.Client
}

func (c *client) Name() string {
	return "binance"
}

func (c *client) GetTicker(ctx context.Context, symbol string) (*exchanges.Ticker, error) {
	endpoint := c.apiPath("/api/v3/ticker/24hr", "/fapi/v1/ticker/24hr")
	payload := url.Values{"symbol": []string{normalizeSymbol(symbol)}}

	data, err := c.get(ctx, endpoint, payload)
	if err != nil {
		return nil, err
	}

	var res struct {
		Symbol             string `json:"symbol"`
		LastPrice          string `json:"lastPrice"`
		BidPrice           string `json:"bidPrice"`
		AskPrice           string `json:"askPrice"`
		HighPrice          string `json:"highPrice"`
		LowPrice           string `json:"lowPrice"`
		Volume             string `json:"volume"`
		QuoteVolume        string `json:"quoteVolume"`
		PriceChangePercent string `json:"priceChangePercent"`
		CloseTime          int64  `json:"closeTime"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	return &exchanges.Ticker{
		Symbol:       res.Symbol,
		LastPrice:    parseDecimal(res.LastPrice),
		BidPrice:     parseDecimal(res.BidPrice),
		AskPrice:     parseDecimal(res.AskPrice),
		High24h:      parseDecimal(res.HighPrice),
		Low24h:       parseDecimal(res.LowPrice),
		VolumeBase:   parseDecimal(res.Volume),
		VolumeQuote:  parseDecimal(res.QuoteVolume),
		Change24hPct: parseDecimal(res.PriceChangePercent),
		Timestamp:    time.UnixMilli(res.CloseTime),
	}, nil
}

func (c *client) GetOrderBook(ctx context.Context, symbol string, depth int) (*exchanges.OrderBook, error) {
	if depth <= 0 {
		depth = 50
	}
	endpoint := c.apiPath("/api/v3/depth", "/fapi/v1/depth")
	params := url.Values{
		"symbol": []string{normalizeSymbol(symbol)},
		"limit":  []string{strconv.Itoa(depth)},
	}

	data, err := c.get(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var res struct {
		LastUpdateID int64           `json:"lastUpdateId"`
		Bids         [][]interface{} `json:"bids"`
		Asks         [][]interface{} `json:"asks"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	book := &exchanges.OrderBook{
		Symbol:    normalizeSymbol(symbol),
		Timestamp: time.Unix(0, res.LastUpdateID),
	}

	for _, bid := range res.Bids {
		book.Bids = append(book.Bids, parseOrderBookEntry(bid))
	}

	for _, ask := range res.Asks {
		book.Asks = append(book.Asks, parseOrderBookEntry(ask))
	}

	return book, nil
}

func (c *client) GetOHLCV(ctx context.Context, symbol string, timeframe string, limit int) ([]exchanges.OHLCV, error) {
	if limit <= 0 {
		limit = 100
	}

	endpoint := c.apiPath("/api/v3/klines", "/fapi/v1/klines")
	params := url.Values{
		"symbol":   []string{normalizeSymbol(symbol)},
		"interval": []string{timeframe},
		"limit":    []string{strconv.Itoa(limit)},
	}

	data, err := c.get(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var raw [][]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	candles := make([]exchanges.OHLCV, 0, len(raw))
	for _, row := range raw {
		openTime := toInt64(row[0])
		closeTime := toInt64(row[6])
		candles = append(candles, exchanges.OHLCV{
			Symbol:    normalizeSymbol(symbol),
			Timeframe: timeframe,
			OpenTime:  time.UnixMilli(openTime),
			CloseTime: time.UnixMilli(closeTime),
			Open:      parseDecimal(fmt.Sprint(row[1])),
			High:      parseDecimal(fmt.Sprint(row[2])),
			Low:       parseDecimal(fmt.Sprint(row[3])),
			Close:     parseDecimal(fmt.Sprint(row[4])),
			Volume:    parseDecimal(fmt.Sprint(row[5])),
		})
	}

	return candles, nil
}

func (c *client) GetTrades(ctx context.Context, symbol string, limit int) ([]exchanges.Trade, error) {
	if limit <= 0 {
		limit = 100
	}

	endpoint := c.apiPath("/api/v3/trades", "/fapi/v1/trades")
	params := url.Values{
		"symbol": []string{normalizeSymbol(symbol)},
		"limit":  []string{strconv.Itoa(limit)},
	}

	data, err := c.get(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}

	var raw []struct {
		ID           int64  `json:"id"`
		Price        string `json:"price"`
		Qty          string `json:"qty"`
		QuoteQty     string `json:"quoteQty"`
		Time         int64  `json:"time"`
		IsBuyerMaker bool   `json:"isBuyerMaker"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	trades := make([]exchanges.Trade, 0, len(raw))
	for _, t := range raw {
		side := exchanges.OrderSideBuy
		if t.IsBuyerMaker {
			side = exchanges.OrderSideSell
		}
		trades = append(trades, exchanges.Trade{
			ID:        strconv.FormatInt(t.ID, 10),
			Symbol:    normalizeSymbol(symbol),
			Price:     parseDecimal(t.Price),
			Amount:    parseDecimal(t.Qty),
			Side:      side,
			Timestamp: time.UnixMilli(t.Time),
		})
	}

	return trades, nil
}

func (c *client) GetFundingRate(ctx context.Context, symbol string) (*exchanges.FundingRate, error) {
	if !c.isFutures() {
		return nil, exchanges.ErrNotSupported
	}

	params := url.Values{"symbol": []string{normalizeSymbol(symbol)}}
	data, err := c.get(ctx, "/fapi/v1/premiumIndex", params)
	if err != nil {
		return nil, err
	}

	var res struct {
		Symbol      string `json:"symbol"`
		FundingRate string `json:"lastFundingRate"`
		NextFunding int64  `json:"nextFundingTime"`
		Time        int64  `json:"time"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	return &exchanges.FundingRate{
		Symbol:    res.Symbol,
		Rate:      parseDecimal(res.FundingRate),
		NextTime:  time.UnixMilli(res.NextFunding),
		Timestamp: time.UnixMilli(res.Time),
	}, nil
}

func (c *client) GetOpenInterest(ctx context.Context, symbol string) (*exchanges.OpenInterest, error) {
	if !c.isFutures() {
		return nil, exchanges.ErrNotSupported
	}

	params := url.Values{"symbol": []string{normalizeSymbol(symbol)}}
	data, err := c.get(ctx, "/fapi/v1/openInterest", params)
	if err != nil {
		return nil, err
	}

	var res struct {
		OpenInterest string `json:"openInterest"`
		Time         int64  `json:"time"`
		Symbol       string `json:"symbol"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	return &exchanges.OpenInterest{
		Symbol:    res.Symbol,
		Amount:    parseDecimal(res.OpenInterest),
		Timestamp: time.UnixMilli(res.Time),
	}, nil
}

func (c *client) GetBalance(ctx context.Context) (*exchanges.Balance, error) {
	if c.isFutures() {
		return c.getFuturesBalance(ctx)
	}

	data, err := c.signed(ctx, http.MethodGet, "/api/v3/account", url.Values{})
	if err != nil {
		return nil, err
	}

	var res struct {
		Balances []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		} `json:"balances"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	detail := make([]exchanges.BalanceDetail, 0, len(res.Balances))
	total := decimal.Zero
	available := decimal.Zero

	for _, b := range res.Balances {
		free := parseDecimal(b.Free)
		locked := parseDecimal(b.Locked)
		detail = append(detail, exchanges.BalanceDetail{
			Currency:  b.Asset,
			Total:     free.Add(locked),
			Available: free,
			Borrowed:  decimal.Zero,
		})
		total = total.Add(free.Add(locked))
		available = available.Add(free)
	}

	return &exchanges.Balance{
		Total:     total,
		Available: available,
		Details:   detail,
		Currency:  "USD",
	}, nil
}

func (c *client) getFuturesBalance(ctx context.Context) (*exchanges.Balance, error) {
	data, err := c.signed(ctx, http.MethodGet, "/fapi/v2/balance", url.Values{})
	if err != nil {
		return nil, err
	}

	var res []struct {
		AccountAlias string `json:"accountAlias"`
		Asset        string `json:"asset"`
		Balance      string `json:"balance"`
		Available    string `json:"availableBalance"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	detail := make([]exchanges.BalanceDetail, 0, len(res))
	total := decimal.Zero
	available := decimal.Zero

	for _, b := range res {
		totalAmt := parseDecimal(b.Balance)
		avail := parseDecimal(b.Available)
		detail = append(detail, exchanges.BalanceDetail{
			Currency:  b.Asset,
			Total:     totalAmt,
			Available: avail,
		})
		total = total.Add(totalAmt)
		available = available.Add(avail)
	}

	return &exchanges.Balance{
		Total:     total,
		Available: available,
		Details:   detail,
		Currency:  "USD",
	}, nil
}

func (c *client) GetPositions(ctx context.Context) ([]exchanges.Position, error) {
	if !c.isFutures() {
		return []exchanges.Position{}, nil
	}

	data, err := c.signed(ctx, http.MethodGet, "/fapi/v2/positionRisk", url.Values{})
	if err != nil {
		return nil, err
	}

	var res []struct {
		Symbol           string `json:"symbol"`
		PositionAmt      string `json:"positionAmt"`
		EntryPrice       string `json:"entryPrice"`
		MarkPrice        string `json:"markPrice"`
		UnRealizedProfit string `json:"unRealizedProfit"`
		LiqPrice         string `json:"liquidationPrice"`
		Leverage         string `json:"leverage"`
		MarginType       string `json:"marginType"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	positions := make([]exchanges.Position, 0, len(res))
	for _, p := range res {
		size := parseDecimal(p.PositionAmt)
		if size.IsZero() {
			continue
		}

		side := exchanges.PositionSideLong
		if size.LessThan(decimal.Zero) {
			side = exchanges.PositionSideShort
		}

		positions = append(positions, exchanges.Position{
			Symbol:           p.Symbol,
			Market:           exchanges.MarketTypeLinearPerp,
			Side:             side,
			Size:             size.Abs(),
			EntryPrice:       parseDecimal(p.EntryPrice),
			MarkPrice:        parseDecimal(p.MarkPrice),
			LiquidationPrice: parseDecimal(p.LiqPrice),
			MarginMode:       marginModeFromString(p.MarginType),
			Leverage:         parseDecimal(p.Leverage),
			UnrealizedPnL:    parseDecimal(p.UnRealizedProfit),
			UpdatedAt:        time.Now(),
		})
	}

	return positions, nil
}

func (c *client) GetOpenOrders(ctx context.Context, symbol string) ([]exchanges.Order, error) {
	endpoint := c.apiPath("/api/v3/openOrders", "/fapi/v1/openOrders")
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", normalizeSymbol(symbol))
	}

	data, err := c.signed(ctx, http.MethodGet, endpoint, params)
	if err != nil {
		return nil, err
	}

	var res []struct {
		OrderID    int64  `json:"orderId"`
		ClientID   string `json:"clientOrderId"`
		Symbol     string `json:"symbol"`
		Side       string `json:"side"`
		Type       string `json:"type"`
		Status     string `json:"status"`
		TimeInForce string `json:"timeInForce"`
		Price      string `json:"price"`
		OrigQty    string `json:"origQty"`
		ExecutedQty string `json:"executedQty"`
		UpdateTime int64  `json:"updateTime"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	orders := make([]exchanges.Order, 0, len(res))
	for _, o := range res {
		orders = append(orders, exchanges.Order{
			ID:           strconv.FormatInt(o.OrderID, 10),
			ClientOrderID: o.ClientID,
			Symbol:       o.Symbol,
			Market:       c.cfg.Market,
			Type:         orderTypeFromString(o.Type),
			Side:         orderSideFromString(o.Side),
			Status:       orderStatusFromString(o.Status),
			TimeInForce:  timeInForceFromString(o.TimeInForce),
			Price:        parseDecimal(o.Price),
			Quantity:     parseDecimal(o.OrigQty),
			Filled:       parseDecimal(o.ExecutedQty),
			CreatedAt:    time.UnixMilli(o.UpdateTime),
			UpdatedAt:    time.UnixMilli(o.UpdateTime),
		})
	}

	return orders, nil
}

func (c *client) PlaceOrder(ctx context.Context, req *exchanges.OrderRequest) (*exchanges.Order, error) {
	if req == nil {
		return nil, exchanges.ErrInvalidRequest
	}

	params := url.Values{
		"symbol": []string{normalizeSymbol(req.Symbol)},
		"side":   []string{strings.ToUpper(string(req.Side))},
		"type":   []string{mapOrderType(req.Type)},
	}

	if !req.Quantity.IsZero() {
		params.Set("quantity", req.Quantity.String())
	}
	if !req.Price.IsZero() {
		params.Set("price", req.Price.String())
	}
	if req.TimeInForce != "" {
		params.Set("timeInForce", string(req.TimeInForce))
	}
	if !req.StopPrice.IsZero() {
		params.Set("stopPrice", req.StopPrice.String())
	}
	if req.ClientOrderID != "" {
		params.Set("newClientOrderId", req.ClientOrderID)
	} else if req.Tag != "" {
		params.Set("newClientOrderId", req.Tag)
	}
	if req.ReduceOnly {
		params.Set("reduceOnly", "true")
	}

	endpoint := c.apiPath("/api/v3/order", "/fapi/v1/order")
	data, err := c.signed(ctx, http.MethodPost, endpoint, params)
	if err != nil {
		return nil, err
	}

	var res struct {
		OrderID      int64  `json:"orderId"`
		ClientOrderID string `json:"clientOrderId"`
		Symbol       string `json:"symbol"`
		Side         string `json:"side"`
		Type         string `json:"type"`
		TransactTime int64  `json:"transactTime"`
		Price        string `json:"price"`
		OrigQty      string `json:"origQty"`
		CummulativeQty string `json:"cummulativeQuoteQty"`
		Status       string `json:"status"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	return &exchanges.Order{
		ID:            strconv.FormatInt(res.OrderID, 10),
		ClientOrderID: res.ClientOrderID,
		Symbol:        res.Symbol,
		Market:        req.Market,
		Type:          orderTypeFromString(res.Type),
		Side:          orderSideFromString(res.Side),
		Status:        orderStatusFromString(res.Status),
		Price:         parseDecimal(res.Price),
		Quantity:      parseDecimal(res.OrigQty),
		Filled:        parseDecimal(res.CummulativeQty),
		TimeInForce:   req.TimeInForce,
		CreatedAt:     time.UnixMilli(res.TransactTime),
		UpdatedAt:     time.UnixMilli(res.TransactTime),
		ReduceOnly:    req.ReduceOnly,
	}, nil
}

func (c *client) CancelOrder(ctx context.Context, symbol, orderID string) error {
	params := url.Values{
		"symbol": []string{normalizeSymbol(symbol)},
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}

	endpoint := c.apiPath("/api/v3/order", "/fapi/v1/order")
	_, err := c.signed(ctx, http.MethodDelete, endpoint, params)
	return err
}

func (c *client) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if !c.isFutures() {
		return exchanges.ErrNotSupported
	}

	params := url.Values{
		"symbol":   []string{normalizeSymbol(symbol)},
		"leverage": []string{strconv.Itoa(leverage)},
	}

	_, err := c.signed(ctx, http.MethodPost, "/fapi/v1/leverage", params)
	return err
}

func (c *client) SetMarginMode(ctx context.Context, symbol string, mode exchanges.MarginMode) error {
	if !c.isFutures() {
		return exchanges.ErrNotSupported
	}

	params := url.Values{
		"symbol":     []string{normalizeSymbol(symbol)},
		"marginType": []string{strings.ToUpper(string(mode))},
	}

	_, err := c.signed(ctx, http.MethodPost, "/fapi/v1/marginType", params)
	return err
}

func (c *client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, params, false)
}

func (c *client) signed(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	return c.doRequest(ctx, method, path, params, true)
}

func (c *client) doRequest(ctx context.Context, method, path string, params url.Values, signed bool) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}

	var body io.Reader
	query := params.Encode()

	if signed {
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		params.Set("recvWindow", strconv.FormatInt(c.cfg.RecvWindow.Milliseconds(), 10))
		signature := c.sign(params.Encode())
		params.Set("signature", signature)
		query = params.Encode()
	}

	reqURL := c.baseURL() + path

	switch method {
	case http.MethodGet, http.MethodDelete:
		if query != "" {
			reqURL = reqURL + "?" + query
		}
	default:
		body = strings.NewReader(query)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	if signed {
		req.Header.Set("X-MBX-APIKEY", c.cfg.APIKey)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, payload)
	}

	return payload, nil
}

func (c *client) baseURL() string {
	if c.isFutures() {
		if c.cfg.Testnet {
			return futuresTestnetURL
		}
		return futuresBaseURL
	}

	if c.cfg.Testnet {
		return spotTestnetBaseURL
	}
	return spotBaseURL
}

func (c *client) apiPath(spotPath, futuresPath string) string {
	if c.isFutures() {
		return futuresPath
	}
	return spotPath
}

func (c *client) isFutures() bool {
	return c.cfg.Market != exchanges.MarketTypeSpot
}

func (c *client) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(c.cfg.SecretKey))
	_, _ = mac.Write([]byte(payload))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func parseAPIError(status int, payload []byte) error {
	var apiErr struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(payload, &apiErr); err == nil && apiErr.Code != 0 {
		if apiErr.Code == -1003 {
			return fmt.Errorf("%w: %s", exchanges.ErrRateLimited, apiErr.Msg)
		}
		return fmt.Errorf("binance error %d: %s", apiErr.Code, apiErr.Msg)
	}
	return fmt.Errorf("binance http %d: %s", status, string(payload))
}

func parseOrderBookEntry(values []interface{}) exchanges.OrderBookEntry {
	if len(values) < 2 {
		return exchanges.OrderBookEntry{}
	}
	return exchanges.OrderBookEntry{
		Price:  parseDecimal(fmt.Sprint(values[0])),
		Amount: parseDecimal(fmt.Sprint(values[1])),
	}
}

func parseDecimal(v string) decimal.Decimal {
	d, err := decimal.NewFromString(v)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case json.Number:
		i, _ := val.Int64()
		return i
	default:
		num, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return num
	}
}

func mapOrderType(t exchanges.OrderType) string {
	switch t {
	case exchanges.OrderTypeMarket:
		return "MARKET"
	case exchanges.OrderTypeStopMarket:
		return "STOP_MARKET"
	case exchanges.OrderTypeStopLimit:
		return "STOP"
	default:
		return "LIMIT"
	}
}

func orderTypeFromString(s string) exchanges.OrderType {
	switch strings.ToUpper(s) {
	case "MARKET":
		return exchanges.OrderTypeMarket
	case "STOP_MARKET":
		return exchanges.OrderTypeStopMarket
	case "STOP", "STOP_LIMIT":
		return exchanges.OrderTypeStopLimit
	default:
		return exchanges.OrderTypeLimit
	}
}

func orderSideFromString(s string) exchanges.OrderSide {
	if strings.ToUpper(s) == "SELL" {
		return exchanges.OrderSideSell
	}
	return exchanges.OrderSideBuy
}

func timeInForceFromString(s string) exchanges.TimeInForce {
	switch strings.ToUpper(s) {
	case "IOC":
		return exchanges.TimeInForceIOC
	case "FOK":
		return exchanges.TimeInForceFOK
	default:
		return exchanges.TimeInForceGTC
	}
}

func orderStatusFromString(s string) exchanges.OrderStatus {
	switch strings.ToUpper(s) {
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

func marginModeFromString(v string) exchanges.MarginMode {
	if strings.ToLower(v) == "isolated" {
		return exchanges.MarginIsolated
	}
	return exchanges.MarginCross
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.ReplaceAll(symbol, "-", ""))
}

