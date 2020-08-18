package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"payment/layer"
	"strconv"
	"strings"
	"time"
)

var tplIndex = template.Must(template.ParseFiles("index.html"))
var tplResponse = template.Must(template.ParseFiles("response.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	paymentData := setupPayment()
	tplIndex.Execute(w, paymentData)
}

func responseHandler(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	paymentResponse := verifyPayment(r)
	tplResponse.Execute(w, paymentResponse)
}

func setupPayment() layer.PaymentData {
	config := layer.LoadConfiguration("config.json")

	var paymentData layer.PaymentData
	paymentData.Name = config.SampleDataName
	paymentData.Email = config.SampleDataEmail
	paymentData.Amount = config.SampleDataAmount
	paymentData.Currency = config.SampleDataCurrency
	paymentData.ContactNumber = config.SampleDataContactNumber
	paymentData.AccessKey = config.AccessKey
	paymentData.Udf = ""

	t := time.Now()
	rndSource := rand.NewSource(t.UnixNano())
	rnd := rand.New(rndSource)
	paymentData.Mtx = t.Format("20060102") + strconv.Itoa((rnd.Intn(9000) + 1000))

	layer.PaymentSetup(config)

	var amount float64
	paymentID := ""
	objJsonData := make(map[string]interface{})

	resp, err := layer.CreatePaymentToken(paymentData)

	if err == nil {
		if err := json.Unmarshal([]byte(resp), &objJsonData); err != nil {
			paymentData.ErrMessage = "Error creating payment token " + fmt.Sprint(err)
		} else {
			_, foundErr := objJsonData["error"]
			if foundErr {
				paymentData.ErrMessage = "E55 Payment error. " + objJsonData["error"].(string)
				if errData, foundErr := objJsonData["error_data"]; foundErr {
					for _, v := range errData.(map[string]interface{}) {
						s := fmt.Sprint(v)
						s = strings.Replace(s, "[", "", -1)
						s = strings.Replace(s, "]", "", -1)
						paymentData.ErrMessage += " " + s
					}
				}
			}

			if paymentData.ErrMessage == "" {
				if objJsonData["id"].(string) == "" {
					paymentData.ErrMessage = "Payment error. Layer token ID cannot be empty."
				} else {
					paymentID = objJsonData["id"].(string)
					amount, _ = strconv.ParseFloat(objJsonData["amount"].(string), 32)
				}
			}

		}
	} else {
		paymentData.ErrMessage = "Error creating payment token. API retuned an error."
	}

	objJsonDataToken := make(map[string]interface{})
	if paymentData.ErrMessage == "" {

		//check that the payment is setup correctly and has not been paid
		respToken, err := layer.GetPaymentToken(paymentID)

		if err == nil {
			if err := json.Unmarshal([]byte(respToken), &objJsonDataToken); err != nil {
				paymentData.ErrMessage = "Error creating payment token " + fmt.Sprint(err)
			} else {

				_, foundErr := objJsonDataToken["error"]
				if foundErr {
					paymentData.ErrMessage = "E56 Payment error. " + objJsonDataToken["error"].(string)
				}

				if paymentData.ErrMessage == "" {
					if objJsonDataToken["status"].(string) == "paid" {
						paymentData.ErrMessage = "Layer: this order has already been paid."
					}
				}

				if paymentData.ErrMessage == "" {
					amount1, _ := strconv.ParseFloat(objJsonDataToken["amount"].(string), 32)
					if amount != amount1 {
						paymentData.ErrMessage = "Layer: an amount mismatch occurred"
					}
				}

			}
		}
	}

	if paymentData.ErrMessage == "" {
		paymentData.PaymentTokenID = objJsonDataToken["id"].(string)
		paymentData.AmountFormatted = fmt.Sprintf("%.2f", amount)

		//create hash
		paymentData.Hash = layer.CreateHash(paymentData.PaymentTokenID, paymentData.AmountFormatted, paymentData.Mtx)
	}

	return paymentData
}

func verifyPayment(r *http.Request) layer.PaymentResponse {

	status := ""
	id := ""

	objJsonData := make(map[string]interface{})

	paymentID := r.Form.Get("layer_payment_id")
	if paymentID == "" {
		status = "Invalid response."
	}

	var amount float64
	if status == "" {
		if amt, err := strconv.ParseFloat(r.Form.Get("layer_order_amount"), 32); err != nil {
			status = "Invalid response."
		} else {
			amount = amt
		}
	}

	transactionID := r.Form.Get("tranid")
	if status == "" && transactionID == "" {
		status = "Invalid response."
	}

	paymentTokenID := r.Form.Get("layer_pay_token_id")
	if status == "" && paymentTokenID == "" {
		status = "Invalid response."
	}

	//calculate and compare hash
	if status == "" {
		amountFormatted := fmt.Sprintf("%.2f", amount)
		hashValue := layer.CreateHash(paymentTokenID, amountFormatted, transactionID)
		if hashValue != r.Form.Get("hash") {
			status = "Invalid hash"
		}
	}

	if status == "" {

		resp, err := layer.GetPaymentDetails(paymentID)

		if err == nil {
			if err := json.Unmarshal([]byte(resp), &objJsonData); err != nil {
				status = "Error fetching payment details " + fmt.Sprint(err)
			} else {
				if len(objJsonData) == 0 {
					status = "Invalid payment data"
				}
				_, foundErr := objJsonData["error"]
				if foundErr {
					status = "Layer: an error occurred E14" + objJsonData["error"].(string)
				}

				if status == "" && objJsonData["id"].(string) == "" {
					status = "Invalid payment data received E98"
				}

				if status == "" {
					//testing
					pToken := objJsonData["payment_token"].(map[string]interface{}) //objJsonData["payment_token"]["id"].(string)
					fetchedID := pToken["id"].(string)
					if fetchedID != paymentTokenID {
						status = "Layer: received layer_pay_token_id and collected layer_pay_token_id doesnt match"
					}
				}

				if status == "" {
					amount1, _ := strconv.ParseFloat(objJsonData["amount"].(string), 32)
					if amount != amount1 {
						status = "Layer: received amount and collected amount doesnt match"
					}
				}

				if status == "" {
					id = objJsonData["id"].(string)

					switch objJsonData["status"].(string) {
					case "authorized", "captured":
						status = "Payment captured: Payment ID " + id
					case "failed", "cancelled":
						status = "Payment cancelled/failed: Payment ID " + id
					default:
						status = "Payment pending: Payment ID " + id
					}
				}
			}
		} else {
			status = "Error fetching payment details " + fmt.Sprint(err)
		}
	}

	paymentResponse := layer.PaymentResponse{status, id}
	return paymentResponse
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/response", responseHandler)
	http.ListenAndServe(":"+port, mux)

}
