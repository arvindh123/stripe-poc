package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-zoo/bone"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/invoice"
	"github.com/stripe/stripe-go/v74/paymentmethod"
	sub "github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/webhook"
	_ "modernc.org/sqlite"
)

const ctxOrgKey = "Organization"

var db *sqlx.DB

type Organization struct {
	ID          int    `json:"id"          db:"id"`
	Name        string `json:"name"        db:"name"`
	Email       string `json:"email"       db:"email"`
	StripeID    string `json:"stripe_id"   db:"stripe_id"`
	StripeSubID string `json:"stripe_sub"  db:"stripe_sub"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("godotenv.Load: %v", err)
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	var err error
	db, err = sqlx.Open("sqlite", "local.db")

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	mux := bone.New()
	mux.Get("/config", http.HandlerFunc(handleConfig))
	mux.Post("/organization/create", http.HandlerFunc(handleCreateOrg))
	mux.Get("/organization", http.HandlerFunc(handleGetAllOrg))
	mux.Get("/organization/:id", middlewareGetID(http.HandlerFunc(handleGetOrgById)))
	mux.Post("/organization/:id/subscription/create", http.HandlerFunc(handleCreateSubscription))
	mux.Get("/subscribe/:id/subscription/cancel", http.HandlerFunc(handleCancelSubscription))
	mux.Get("/subscribe/:id/subscription/update", http.HandlerFunc(handleUpdateSubscription))
	mux.Get("/subscribe/:id/subscription/pay", http.HandlerFunc(handleUpdateSubscription))
	mux.Get("/organization/:id/payment-method", middlewareGetID(http.HandlerFunc(handleRetrievePaymentMethod)))
	mux.Get("/organization/:id/add-payment-method", middlewareGetID(http.HandlerFunc(handleRetrievePaymentMethod)))

	fmt.Println("Starting Server")
	log.Fatalln(http.ListenAndServe(":8080", mux))

}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, struct {
		PublishableKey string `json:"publishableKey"`
	}{
		PublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
	})
}

func handleGetAllOrg(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Queryx("SELECT * FROM  organization")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		err := rows.StructScan(&o)
		if err != nil {
			log.Fatal(err)
		}
		orgs = append(orgs, o)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orgs)
}

func middlewareGetID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := bone.GetValue(r, "id")
		if id == "" {
			http.Error(w, "id is missing path", http.StatusBadRequest)
			return
		}
		org, err := getOrganization(id)
		if err != nil {
			switch {
			case err == sql.ErrNoRows:
				http.Error(w, "", http.StatusNotFound)
				return
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		ctx := context.WithValue(r.Context(), ctxOrgKey, org)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handleGetOrgById(w http.ResponseWriter, r *http.Request) {
	org := r.Context().Value(ctxOrgKey)
	writeJSON(w, org)
}

func handleCreateOrg(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err := checkOrganization(req.Name, req.Email)
	if err != sql.ErrNoRows {
		http.Error(w, "Organization already exists or any other error : "+err.Error(), http.StatusForbidden)
		return
	}

	params := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
		Name:  stripe.String(req.Name),
	}

	c, err := customer.New(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("customer.New: %v", err)
		return
	}

	query := "INSERT INTO `organization` (`name`, `email`, `stripe_id`) VALUES (?, ?, ?)"
	if _, err := db.ExecContext(context.Background(), query, req.Name, req.Email, c.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	writeJSON(w, "")
}

func handleRetrievePaymentMethod(w http.ResponseWriter, r *http.Request) {
	org := r.Context().Value(ctxOrgKey)
	organization, ok := org.(Organization)
	if !ok {
		http.Error(w, "invalid organization context", http.StatusInternalServerError)
		return
	}

	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(organization.StripeID),
		// Type:     stripe.String("card"),
	}
	i := paymentmethod.List(params)
	if i.Err() != nil {
		http.Error(w, i.Err().Error(), http.StatusInternalServerError)
		return
	}
	if !i.Next() {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	for i.Next() {
		pm := i.PaymentMethod()
		fmt.Println(pm)
		writeJSON(w, pm)
	}

}

func handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PaymentMethodID string `json:"paymentMethodId"`
		CustomerID      string `json:"customerId,omitempty"`
		PriceID         string `json:"priceId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}
	req.CustomerID = bone.GetValue(r, "id")
	// Attach PaymentMethod
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(req.CustomerID),
	}
	pm, err := paymentmethod.Attach(
		req.PaymentMethodID,
		params,
	)
	if err != nil {
		writeJSON(w, struct {
			Error error `json:"error"`
		}{err})
		return
	}

	// Update invoice settings default
	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	c, err := customer.Update(
		req.CustomerID,
		customerParams,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("customer.Update: %v %s", err, c.ID)
		return
	}

	// Create subscription
	subscriptionParams := &stripe.SubscriptionParams{
		Customer: stripe.String(req.CustomerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Plan: stripe.String(os.Getenv(req.PriceID)),
			},
		},
	}
	subscriptionParams.AddExpand("latest_invoice.payment_intent")
	subscriptionParams.AddExpand("pending_setup_intent")

	s, err := sub.New(subscriptionParams)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("sub.New: %v", err)
		return
	}

	writeJSON(w, s)
}

func handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubscriptionID string `json:"subscriptionId"`
	}
	req.SubscriptionID = bone.GetValue(r, "id")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	s, err := sub.Cancel(req.SubscriptionID, nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("sub.Cancel: %v", err)
		return
	}

	writeJSON(w, s)
}

func handleRetrieveUpcomingInvoice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubscriptionID string `json:"subscriptionId,omitempty"`
		CustomerID     string `json:"customerId,omitempty"`
		NewPriceID     string `json:"newPriceId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	s, err := sub.Get(req.SubscriptionID, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("sub.Get: %v", err)
		return
	}

	params := &stripe.InvoiceUpcomingParams{
		Customer:     stripe.String(req.CustomerID),
		Subscription: stripe.String(req.SubscriptionID),
		SubscriptionItems: []*stripe.SubscriptionItemsParams{{
			ID:         stripe.String(s.Items.Data[0].ID),
			Deleted:    stripe.Bool(true),
			ClearUsage: stripe.Bool(true),
		}, {
			Price:   stripe.String(os.Getenv(req.NewPriceID)),
			Deleted: stripe.Bool(false),
		}},
	}
	in, err := invoice.Upcoming(params)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("invoice.GetNext: %v", err)
		return
	}

	writeJSON(w, in)
}

func handleUpdateSubscription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubscriptionID string `json:"subscriptionId"`
		NewPriceID     string `json:"newPriceId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	s, err := sub.Get(req.SubscriptionID, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("sub.Get: %v", err)
		return
	}

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
		Items: []*stripe.SubscriptionItemsParams{{
			ID:    stripe.String(s.Items.Data[0].ID),
			Price: stripe.String(os.Getenv(req.NewPriceID)),
		}},
	}

	updatedSubscription, err := sub.Update(req.SubscriptionID, params)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("sub.Update: %v", err)
		return
	}

	writeJSON(w, updatedSubscription)
}

func handleRetryInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CustomerID      string `json:"customerId"`
		PaymentMethodID string `json:"paymentMethodId"`
		InvoiceID       string `json:"invoiceId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	// Attach PaymentMethod
	params := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(req.CustomerID),
	}
	pm, err := paymentmethod.Attach(
		req.PaymentMethodID,
		params,
	)
	if err != nil {
		writeJSON(w, struct {
			Error error `json:"error"`
		}{err})
		return
	}

	// Update invoice settings default
	customerParams := &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm.ID),
		},
	}
	c, err := customer.Update(
		req.CustomerID,
		customerParams,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("customer.Update: %v %s", err, c.ID)
		return
	}

	// Retrieve Invoice
	invoiceParams := &stripe.InvoiceParams{}
	invoiceParams.AddExpand("payment_intent")
	in, err := invoice.Get(
		req.InvoiceID,
		invoiceParams,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("invoice.Get: %v", err)
		return
	}

	writeJSON(w, in)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("ioutil.ReadAll: %v", err)
		return
	}

	event, err := webhook.ConstructEvent(b, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("webhook.ConstructEvent: %v", err)
		return
	}

	if event.Type != "checkout.session.completed" {
		return
	}

	cust, err := customer.Get(event.GetObjectValue("customer"), nil)
	if err != nil {
		log.Printf("customer.Get: %v", err)
		return
	}

	if event.GetObjectValue("display_items", "0", "custom") != "" &&
		event.GetObjectValue("display_items", "0", "custom", "name") == "Pasha e-book" {
		log.Printf("🔔 Customer is subscribed and bought an e-book! Send the e-book to %s", cust.Email)
	} else {
		log.Printf("🔔 Customer is subscribed but did not buy an e-book.")
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewEncoder.Encode: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, &buf); err != nil {
		log.Printf("io.Copy: %v", err)
		return
	}
}

func getOrganization(id string) (Organization, error) {
	var org Organization
	err := db.Get(&org, fmt.Sprintf("SELECT * FROM  organization WHERE id=$1"), id)
	return org, err
}

func checkOrganization(name, email string) (Organization, error) {
	var org Organization
	err := db.Get(&org, "SELECT * FROM  organization WHERE name=$1 and email=$2 LIMIT 1 ", name, email)
	return org, err
}
