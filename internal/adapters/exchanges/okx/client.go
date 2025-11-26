package okx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"prometheus/internal/adapters/exchanges"
	"prometheus/pkg/errors"
)

const (
	productionBaseURL = "https://www.okx.com"
	testBaseURL       = "https://www.okx.com" // public demo, still same domain with paper flag
	defaultTimeout    = 10 * time.Second
)

// Config configures the OKX client.
type Config struct {
	APIKey     string
	SecretKey  string
	Passphrase string
	Market     exchanges.MarketType
	Testnet    bool

	HTTPClient *http.Client
}

// NewClient constructs a new OKX adapter.
func NewClient(cfg Config) (exchanges.Exchange, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}
	if cfg.Market == "" {
		cfg.Market = exchanges.MarketTypeLinearPerp
	}
	return &client{cfg: cfg}, nil
}

type client struct {
	cfg Config
}

func (c *client) Name() string {
	return "okx"
}

func (c *client) GetTicker(ctx context.Context, symbol string) (*exchanges.Ticker, error) {
	params := url.Values{
		"instId": []string{normalizeSymbol(symbol)},
	}
	var res struct {
		Data []struct {
			InstID    string `json:"instId"`
			Last      string `json:"last"`
			BidPx     string `json:"bidPx"`
			AskPx     string `json:"askPx"`
			High24h   string `json:"high24h"`
			Low24h    string `json:"low24h"`
			VolCcy24h string `json:"volCcy24h"`
			Vol24h    string `json:"vol24h"`
			SodUtc0   string `json:"sodUtc0"`
			Open24h   string `json:"open24h"`
			Ts        string `json:"ts"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/v5/market/ticker", params, &res); err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, errors.ErrNotFound
	}
	item := res.Data[0]
	return &exchanges.Ticker{
		Symbol:       item.InstID,
		LastPrice:    dec(item.Last),
		BidPrice:     dec(item.BidPx),
		AskPrice:     dec(item.AskPx),
		High24h:      dec(item.High24h),
		Low24h:       dec(item.Low24h),
		VolumeBase:   dec(item.Vol24h),
		VolumeQuote:  dec(item.VolCcy24h),
		Change24hPct: percentage(dec(item.Last), dec(item.Open24h)),
		Timestamp:    time.UnixMilli(parseInt64(item.Ts)),
	}, nil
}

func (c *client) GetOrderBook(ctx context.Context, symbol string, depth int) (*exchanges.OrderBook, error) {
	if depth <= 0 {
		depth = 40
	}
	params := url.Values{
		"instId": []string{normalizeSymbol(symbol)},
		"sz":     []string{strconv.Itoa(depth)},
	}
	var res struct {
		Data []struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
			Ts   string     `json:"ts"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/v5/market/books", params, &res); err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, errors.ErrNotFound
	}
	data := res.Data[0]
	book := &exchanges.OrderBook{
		Symbol:    normalizeSymbol(symbol),
		Timestamp: time.UnixMilli(parseInt64(data.Ts)),
	}
	for _, ask := range data.Asks {
		book.Asks = append(book.Asks, exchanges.OrderBookEntry{
			Price:  decIdx(ask, 0),
			Amount: decIdx(ask, 1),
		})
	}
	for _, bid := range data.Bids {
		book.Bids = append(book.Bids, exchanges.OrderBookEntry{
			Price:  decIdx(bid, 0),
			Amount: decIdx(bid, 1),
		})
	}
	return book, nil
}

func (c *client) GetOHLCV(ctx context.Context, symbol, timeframe string, limit int) ([]exchanges.OHLCV, error) {
	if limit <= 0 {
		limit = 200
	}
	params := url.Values{
		"instId": []string{normalizeSymbol(symbol)},
		"bar":    []string{timeframe},
		"limit":  []string{strconv.Itoa(limit)},
	}
	var res struct {
		Data [][]string `json:"data"`
	}
	if err := c.get(ctx, "/api/v5/market/candles", params, &res); err != nil {
		return nil, err
	}
	candles := make([]exchanges.OHLCV, 0, len(res.Data))
	for _, row := range res.Data {
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
			Volume:    dec(row[6]),
		})
	}
	return candles, nil
}

func (c *client) GetTrades(ctx context.Context, symbol string, limit int) ([]exchanges.Trade, error) {
	if limit <= 0 {
		limit = 100
	}
	params := url.Values{
		"instId": []string{normalizeSymbol(symbol)},
		"limit":  []string{strconv.Itoa(limit)},
	}
	var res struct {
		Data []struct {
			TradeID string `json:"tradeId"`
			Px      string `json:"px"`
			Sz      string `json:"sz"`
			Side    string `json:"side"`
			Ts      string `json:"ts"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/v5/market/trades", params, &res); err != nil {
		return nil, err
	}
	trades := make([]exchanges.Trade, 0, len(res.Data))
	for _, t := range res.Data {
		trades = append(trades, exchanges.Trade{
			ID:        t.TradeID,
			Symbol:    normalizeSymbol(symbol),
			Price:     dec(t.Px),
			Amount:    dec(t.Sz),
			Side:      sideFromString(t.Side),
			Timestamp: time.UnixMilli(parseInt64(t.Ts)),
		})
	}
	return trades, nil
}

func (c *client) GetFundingRate(ctx context.Context, symbol string) (*exchanges.FundingRate, error) {
	if c.marketType() == exchanges.MarketTypeSpot {
		return nil, exchanges.ErrNotSupported
	}
	params := url.Values{
		"instId": []string{normalizeSymbol(symbol)},
	}
	var res struct {
		Data []struct {
			InstID    string `json:"instId"`
			Funding   string `json:"fundingRate"`
			NextRate  string `json:"nextFundingRate"`
			FundingTs string `json:"fundingTime"`
			Ts        string `json:"ts"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/v5/public/funding-rate", params, &res); err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, errors.ErrNotFound
	}
	item := res.Data[0]
	return &exchanges.FundingRate{
		Symbol:    item.InstID,
		Rate:      dec(item.Funding),
		NextTime:  time.UnixMilli(parseInt64(item.FundingTs)),
		Timestamp: time.UnixMilli(parseInt64(item.Ts)),
	}, nil
}

func (c *client) GetOpenInterest(ctx context.Context, symbol string) (*exchanges.OpenInterest, error) {
	if c.marketType() == exchanges.MarketTypeSpot {
		return nil, exchanges.ErrNotSupported
	}
	params := url.Values{
		"instId": []string{normalizeSymbol(symbol)},
	}
	var res struct {
		Data []struct {
			InstID string `json:"instId"`
			OI     string `json:"oi"`
			Ts     string `json:"ts"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/v5/public/open-interest", params, &res); err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, errors.ErrNotFound
	}
	item := res.Data[0]
	return &exchanges.OpenInterest{
		Symbol:    item.InstID,
		Amount:    dec(item.OI),
		Timestamp: time.UnixMilli(parseInt64(item.Ts)),
	}, nil
}

func (c *client) GetBalance(ctx context.Context) (*exchanges.Balance, error) {
	var res struct {
		Data []struct {
			TotalEq string `json:"totalEq"`
			Details []struct {
				Ccy      string `json:"ccy"`
				CashBal  string `json:"cashBal"`
				AvailBal string `json:"availBal"`
				DisEq    string `json:"disEq"`
				Borrowed string `json:"borrowFroz"`
			} `json:"details"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v5/account/balance", nil, nil, &res); err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, errors.ErrNotFound
	}
	entry := res.Data[0]
	detail := make([]exchanges.BalanceDetail, 0, len(entry.Details))
	for _, d := range entry.Details {
		detail = append(detail, exchanges.BalanceDetail{
			Currency:  d.Ccy,
			Total:     dec(d.CashBal),
			Available: dec(d.AvailBal),
			Borrowed:  dec(d.Borrowed),
		})
	}
	return &exchanges.Balance{
		Total:     dec(entry.TotalEq),
		Available: sumAvailable(detail),
		Currency:  "USD",
		Details:   detail,
	}, nil
}

func (c *client) GetPositions(ctx context.Context) ([]exchanges.Position, error) {
	params := url.Values{}
	if c.marketType() != exchanges.MarketTypeSpot {
		params.Set("instType", c.instType())
	}
	var res struct {
		Data []struct {
			InstID  string `json:"instId"`
			PosSide string `json:"posSide"`
			Pos     string `json:"pos"`
			AvgPx   string `json:"avgPx"`
			MarkPx  string `json:"markPx"`
			LiqPx   string `json:"liqPx"`
			Lever   string `json:"lever"`
			MgnMode string `json:"mgnMode"`
			Upl     string `json:"upl"`
			TS      string `json:"uTime"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v5/account/positions", params, nil, &res); err != nil {
		return nil, err
	}

	positions := make([]exchanges.Position, 0, len(res.Data))
	for _, p := range res.Data {
		size := dec(p.Pos)
		if size.IsZero() {
			continue
		}
		positions = append(positions, exchanges.Position{
			Symbol:           p.InstID,
			Market:           c.marketType(),
			Side:             positionSideFromString(p.PosSide),
			Size:             size.Abs(),
			EntryPrice:       dec(p.AvgPx),
			MarkPrice:        dec(p.MarkPx),
			LiquidationPrice: dec(p.LiqPx),
			MarginMode:       marginModeFromString(p.MgnMode),
			Leverage:         dec(p.Lever),
			UnrealizedPnL:    dec(p.Upl),
			UpdatedAt:        time.UnixMilli(parseInt64(p.TS)),
		})
	}
	return positions, nil
}

func (c *client) GetOpenOrders(ctx context.Context, symbol string) ([]exchanges.Order, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("instId", normalizeSymbol(symbol))
	}
	if c.marketType() != exchanges.MarketTypeSpot {
		params.Set("instType", c.instType())
	}

	var res struct {
		Data []struct {
			InstID      string `json:"instId"`
			OrdID       string `json:"ordId"`
			ClOrdID     string `json:"clOrdId"`
			Side        string `json:"side"`
			OrdType     string `json:"ordType"`
			Sz          string `json:"sz"`
			Px          string `json:"px"`
			AvgPx       string `json:"avgPx"`
			State       string `json:"state"`
			FillSz      string `json:"fillSz"`
			FeeCcy      string `json:"feeCcy"`
			TimeInForce string `json:"timeInForce"`
			ReduceOnly  bool   `json:"reduceOnly"`
			CTime       string `json:"cTime"`
			UTime       string `json:"uTime"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/v5/trade/orders-pending", params, nil, &res); err != nil {
		return nil, err
	}

	orders := make([]exchanges.Order, 0, len(res.Data))
	for _, o := range res.Data {
		orders = append(orders, exchanges.Order{
			ID:            o.OrdID,
			ClientOrderID: o.ClOrdID,
			Symbol:        o.InstID,
			Market:        c.marketType(),
			Type:          orderTypeFromString(o.OrdType),
			Side:          sideFromString(o.Side),
			Status:        stateToOrderStatus(o.State),
			Price:         dec(o.Px),
			Quantity:      dec(o.Sz),
			Filled:        dec(o.FillSz),
			AvgFillPrice:  dec(o.AvgPx),
			TimeInForce:   tifFromOKX(o.TimeInForce),
			ReduceOnly:    o.ReduceOnly,
			CreatedAt:     time.UnixMilli(parseInt64(o.CTime)),
			UpdatedAt:     time.UnixMilli(parseInt64(o.UTime)),
		})
	}
	return orders, nil
}

func (c *client) PlaceOrder(ctx context.Context, req *exchanges.OrderRequest) (*exchanges.Order, error) {
	if req == nil {
		return nil, exchanges.ErrInvalidRequest
	}
	payload := map[string]string{
		"instId":  normalizeSymbol(req.Symbol),
		"tdMode":  tdMode(req.MarginMode),
		"side":    strings.ToLower(string(req.Side)),
		"ordType": orderTypeToAPI(req.Type),
		"sz":      req.Quantity.String(),
	}
	if !req.Price.IsZero() {
		payload["px"] = req.Price.String()
	}
	if req.ClientOrderID != "" {
		payload["clOrdId"] = req.ClientOrderID
	} else if req.Tag != "" {
		payload["clOrdId"] = req.Tag
	}
	if req.ReduceOnly {
		payload["reduceOnly"] = "true"
	}

	var res struct {
		Data []struct {
			OrdID   string `json:"ordId"`
			ClOrdID string `json:"clOrdId"`
		} `json:"data"`
	}

	if err := c.request(ctx, http.MethodPost, "/api/v5/trade/order", nil, payload, &res); err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, errors.ErrNotFound
	}
	data := res.Data[0]
	return &exchanges.Order{
		ID:            data.OrdID,
		ClientOrderID: data.ClOrdID,
		Symbol:        normalizeSymbol(req.Symbol),
		Market:        c.marketType(),
		Type:          req.Type,
		Side:          req.Side,
		Status:        exchanges.OrderStatusNew,
		Price:         req.Price,
		Quantity:      req.Quantity,
		TimeInForce:   req.TimeInForce,
		ReduceOnly:    req.ReduceOnly,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

func (c *client) CancelOrder(ctx context.Context, symbol, orderID string) error {
	payload := map[string]string{
		"instId": normalizeSymbol(symbol),
		"ordId":  orderID,
	}
	return c.request(ctx, http.MethodPost, "/api/v5/trade/cancel-order", nil, payload, nil)
}

func (c *client) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	payload := map[string]string{
		"instId":  normalizeSymbol(symbol),
		"lever":   strconv.Itoa(leverage),
		"mgnMode": tdMode(""),
	}
	return c.request(ctx, http.MethodPost, "/api/v5/account/set-leverage", nil, payload, nil)
}

func (c *client) SetMarginMode(ctx context.Context, symbol string, mode exchanges.MarginMode) error {
	payload := map[string]string{
		"instId":  normalizeSymbol(symbol),
		"lever":   "1",
		"mgnMode": tdMode(mode),
	}
	return c.request(ctx, http.MethodPost, "/api/v5/account/set-leverage", nil, payload, nil)
}

func (c *client) get(ctx context.Context, path string, params url.Values, target interface{}) error {
	reqURL := c.baseURL() + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.Wrapf(errors.ErrInternal, "okx http %d: %s", resp.StatusCode, string(body))
	}
	return decodeEnvelope(body, target)
}

func (c *client) request(ctx context.Context, method, path string, params url.Values, payload interface{}, target interface{}) error {
	if c.cfg.APIKey == "" || c.cfg.SecretKey == "" || c.cfg.Passphrase == "" {
		return errors.ErrUnauthorized
	}

	var body io.Reader
	var bodyStr string
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		bodyStr = string(raw)
		body = strings.NewReader(bodyStr)
	}

	requestPath := path
	if len(params) > 0 {
		requestPath += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+requestPath, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	prehash := timestamp + strings.ToUpper(method) + path
	if len(params) > 0 {
		prehash += "?" + params.Encode()
	}
	prehash += bodyStr

	signature := sign(prehash, c.cfg.SecretKey)

	req.Header.Set("OK-ACCESS-KEY", c.cfg.APIKey)
	req.Header.Set("OK-ACCESS-SIGN", signature)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", c.cfg.Passphrase)
	if c.cfg.Testnet {
		req.Header.Set("x-simulated-trading", "1")
	}

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return errors.Wrapf(errors.ErrInternal, "okx http %d: %s", resp.StatusCode, string(respBody))
	}
	return decodeEnvelope(respBody, target)
}

type envelope struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func decodeEnvelope(body []byte, target interface{}) error {
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return err
	}
	if env.Code != "0" {
		return errors.Wrapf(errors.ErrInternal, "okx error %s: %s", env.Code, env.Msg)
	}
	if target == nil {
		return nil
	}
	if len(env.Data) == 0 {
		return nil
	}
	return json.Unmarshal(env.Data, target)
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (c *client) baseURL() string {
	if c.cfg.Testnet {
		return testBaseURL
	}
	return productionBaseURL
}

func (c *client) instType() string {
	switch c.cfg.Market {
	case exchanges.MarketTypeSpot:
		return "SPOT"
	case exchanges.MarketTypeInversePerp, exchanges.MarketTypeDeliveryFut:
		return "SWAP"
	default:
		return "SWAP"
	}
}

func (c *client) marketType() exchanges.MarketType {
	return c.cfg.Market
}

func tdMode(mode interface{}) string {
	switch v := mode.(type) {
	case exchanges.MarginMode:
		if v == exchanges.MarginIsolated {
			return "isolated"
		}
	case string:
		if v == string(exchanges.MarginIsolated) {
			return "isolated"
		}
	}
	return "cross"
}

func percentage(current, open decimal.Decimal) decimal.Decimal {
	if open.IsZero() {
		return decimal.Zero
	}
	return current.Sub(open).Div(open).Mul(decimal.NewFromInt(100))
}

func sumAvailable(details []exchanges.BalanceDetail) decimal.Decimal {
	total := decimal.Zero
	for _, d := range details {
		total = total.Add(d.Available)
	}
	return total
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

func parseInt64(v string) int64 {
	i, _ := strconv.ParseInt(v, 10, 64)
	return i
}

func intervalDurationMs(tf string) int64 {
	switch tf {
	case "1m":
		return int64(time.Minute / time.Millisecond)
	case "3m":
		return int64(3 * time.Minute / time.Millisecond)
	case "5m":
		return int64(5 * time.Minute / time.Millisecond)
	case "15m":
		return int64(15 * time.Minute / time.Millisecond)
	case "30m":
		return int64(30 * time.Minute / time.Millisecond)
	case "1H", "1h":
		return int64(time.Hour / time.Millisecond)
	case "4H", "4h":
		return int64(4 * time.Hour / time.Millisecond)
	case "1D", "1d":
		return int64(24 * time.Hour / time.Millisecond)
	default:
		return int64(time.Minute / time.Millisecond)
	}
}

func sideFromString(value string) exchanges.OrderSide {
	if strings.EqualFold(value, "sell") {
		return exchanges.OrderSideSell
	}
	return exchanges.OrderSideBuy
}

func positionSideFromString(value string) exchanges.PositionSide {
	switch strings.ToLower(value) {
	case "short":
		return exchanges.PositionSideShort
	case "long":
		return exchanges.PositionSideLong
	default:
		return exchanges.PositionSideBoth
	}
}

func orderTypeFromString(value string) exchanges.OrderType {
	switch strings.ToUpper(value) {
	case "MARKET":
		return exchanges.OrderTypeMarket
	case "POST_ONLY", "LIMIT":
		return exchanges.OrderTypeLimit
	case "IOC":
		return exchanges.OrderTypeMarket
	default:
		return exchanges.OrderTypeLimit
	}
}

func orderTypeToAPI(value exchanges.OrderType) string {
	switch value {
	case exchanges.OrderTypeMarket:
		return "market"
	default:
		return "limit"
	}
}

func stateToOrderStatus(state string) exchanges.OrderStatus {
	switch state {
	case "live":
		return exchanges.OrderStatusOpen
	case "partially_filled":
		return exchanges.OrderStatusPartial
	case "filled":
		return exchanges.OrderStatusFilled
	case "canceled":
		return exchanges.OrderStatusCanceled
	default:
		return exchanges.OrderStatusUnknown
	}
}

func marginModeFromString(value string) exchanges.MarginMode {
	if strings.EqualFold(value, "isolated") {
		return exchanges.MarginIsolated
	}
	return exchanges.MarginCross
}

func normalizeSymbol(symbol string) string {
	s := strings.ToUpper(symbol)
	s = strings.ReplaceAll(s, "/", "-")
	return s
}

func tifFromOKX(value string) exchanges.TimeInForce {
	switch strings.ToUpper(value) {
	case "IOC":
		return exchanges.TimeInForceIOC
	case "FOK":
		return exchanges.TimeInForceFOK
	default:
		return exchanges.TimeInForceGTC
	}
}
