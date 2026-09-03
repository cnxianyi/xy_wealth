package asset

// Balance keeps monetary values as decimal strings to avoid floating-point
// precision loss at API boundaries.
type Balance struct {
	Symbol string `json:"symbol"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
	Total  string `json:"total"`
}
