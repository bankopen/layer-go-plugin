package layer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/go-resty/resty"
)

var baseURIUAT string = "https://icp-api.bankopen.co/api"
var baseURISandbox string = "https://sandbox-icp-api.bankopen.co/api"

var createTokenURI, getpaymentURI string

//PaymentData - structure to data that will be passed to the gateway
type PaymentData struct {
	Name            string  `json:"name",omitempty`
	Email           string  `json:"email_id",omitempty`
	Amount          float32 `json:"amount",omitempty`
	Currency        string  `json:"currency",omitempty`
	ContactNumber   string  `json:"contact_number",omitempty`
	Mtx             string  `json:"mtx",omitempty`
	Udf             string  `json:"udf",omitempty`
	PaymentTokenID  string  `json:"-"`
	PaymentID       string  `json:"-"`
	AmountFormatted string  `json:"-"`
	FallbackURL     string  `json:"-"`
	Hash            string  `json:"-"`
	Key             string  `json:"-"`
	ErrMessage      string  `json:"-"`
	AccessKey       string  `json:"-"`
}

type PaymentResponse struct {
	Status   string
	PaymentI string
}

var config Config

//PaymentSetup - set up the payment parameters
func PaymentSetup(configuration Config) {
	config = configuration

	//url's
	if config.Environment == "test" {
		createTokenURI = baseURISandbox + "/payment_token"
		getpaymentURI = baseURISandbox + "/payment"
	} else {
		createTokenURI = baseURIUAT + "/payment_token"
		getpaymentURI = baseURIUAT + "/payment"
	}

}

//CreatePaymentToken - create payment token
func CreatePaymentToken(data PaymentData) (string, error) {

	client := resty.New()

	jsonData, err := json.Marshal(data)

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+config.AccessKey+":"+config.SecretKey).
		SetBody(jsonData).
		EnableTrace().
		Post(createTokenURI)

	return resp.String(), err
}

//GetPaymentToken - get payment token information based on id
func GetPaymentToken(paymentTokenID string) (string, error) {

	client := resty.New()

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+config.AccessKey+":"+config.SecretKey).
		EnableTrace().
		Get(createTokenURI + "/" + paymentTokenID)

	return resp.String(), err
}

//GetPaymentDetails - retrieve payment details for a specific payment
func GetPaymentDetails(paymentID string) (string, error) {

	client := resty.New()

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+config.AccessKey+":"+config.SecretKey).
		Get(getpaymentURI + "/" + paymentID)

	return resp.String(), err
}

//CreateHash - create a SHA 256 hash value
func CreateHash(id, amount, transactionID string) string {

	data := config.AccessKey + "|" + amount + "|" + id + "|" + transactionID

	h := hmac.New(sha256.New, []byte(config.SecretKey))
	h.Write([]byte(data))
	sha := hex.EncodeToString(h.Sum(nil))

	return sha
}
