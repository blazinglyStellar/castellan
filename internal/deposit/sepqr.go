package deposit

import (
	"encoding/base64"
	"fmt"
	"net/url"

	qrcode "github.com/skip2/go-qrcode"
)

const defaultQRSize = 256

func BuildSEP7URI(destination, memo, amount, assetCode string) string {
	q := url.Values{}
	q.Set("destination", destination)
	q.Set("memo", memo)
	q.Set("memo_type", "text")
	q.Set("amount", amount)
	q.Set("asset_code", assetCode)
	return "web+stellar:pay?" + q.Encode()
}

func GenerateQRCode(uri string) (string, error) {
	png, err := qrcode.Encode(uri, qrcode.Medium, defaultQRSize)
	if err != nil {
		return "", fmt.Errorf("encode qr: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
