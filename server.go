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
	"strings"

	"github.com/go-zoo/bone"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/invoice"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/paymentmethod"
	"github.com/stripe/stripe-go/v74/price"
	"github.com/stripe/stripe-go/v74/product"
	sub "github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/webhook"
	_ "modernc.org/sqlite"
)

const ctxOrgKey = "Organization"

var subPlans = map[string]string{
	"planA": "price_1NEDyqSAVJByQTEdNkeEdf7P",
	"planB": "price_1NEDyNSAVJByQTEdrH97Z3B6",
}

var db *sqlx.DB

type Product struct {
	ID          string `json:"id"`
	Active      bool   `json:"active"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
type Plan struct {
	ID       string  `json:"id"`
	SiID     string  `json:"si_id"`
	SubID    string  `json:"sub_id"`
	Active   bool    `json:"active"`
	Quantity int64   `json:"quantity"`
	Amount   int64   `json:"Amount"`
	Product  Product `json:"product"`
}

type Organization struct {
	ID          int    `json:"id"          db:"id"`
	Name        string `json:"name"        db:"name"`
	Email       string `json:"email"       db:"email"`
	StripeID    string `json:"stripe_id"   db:"stripe_id"`
	StripeSubID string `json:"stripe_sub"  db:"stripe_sub"`
	SubStatus   string `json:"sub_status"  db:"sub_status"`
	Plans       []Plan `json:"plans"  db:"-"`
	PlansByte   []byte `json:"-"  db:"plans"`
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

	// cors.Default() setup the middleware with default options being
	// all origins accepted with simple methods (GET, POST). See
	// documentation below for more options.
	mux := bone.New()

	mux.Get("/config", http.HandlerFunc(getConfig))
	mux.Post("/organization/create", http.HandlerFunc(handleCreateOrg))
	mux.Get("/organization", http.HandlerFunc(getAllOrg))
	mux.Get("/plans", http.HandlerFunc(getPlans))
	mux.Get("/organization/:id", middlewareGetID(http.HandlerFunc(getOrgById)))
	mux.Get("/organization/:id/sub", middlewareGetID(http.HandlerFunc(getSubscriptionInfo)))
	mux.Post("/organization/:id/sub", middlewareGetID(http.HandlerFunc(createSubscription)))
	mux.Delete("/organization/:id/sub", middlewareGetID(http.HandlerFunc(cancelSubscription)))
	mux.Patch("/organization/:id/sub", middlewareGetID(http.HandlerFunc(updateSubscription)))
	mux.Get("/organization/:id/payment-method", middlewareGetID(http.HandlerFunc(retrievePaymentMethod)))
	mux.Get("/organization/:id/add-payment-method", middlewareGetID(http.HandlerFunc(retrievePaymentMethod)))
	mux.Post("/stripe/webhook", http.HandlerFunc(handleWebhook))

	c := cors.New(cors.Options{
		AllowedMethods:   []string{"HEAD", "GET", "POST", "PATCH", "UPDATE", "DELETE", "PUT", "OPTIONS"},
		AllowCredentials: true,
	})
	handler := c.Handler(mux)

	fmt.Println("Starting Server")
	log.Fatalln(http.ListenAndServe("127.0.0.1:8080", handler))

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

func middlelwareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Max-Age", "15")
		next.ServeHTTP(w, r)
	})
}

func getPlans(w http.ResponseWriter, r *http.Request) {
	var plansPrice []*stripe.Price
	for planKeyName, planPriceID := range subPlans {
		pr, err := getPrice(planPriceID)
		if err != nil {
			http.Error(w, "Failed to get plan "+err.Error(), http.StatusInternalServerError)
			return
		}
		pr.Nickname = planKeyName
		plansPrice = append(plansPrice, pr)
	}

	writeJSON(w, plansPrice)
}

func getConfig(w http.ResponseWriter, r *http.Request) {
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

func getAllOrg(w http.ResponseWriter, r *http.Request) {
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

func getOrgById(w http.ResponseWriter, r *http.Request) {
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
		if err != nil {
			http.Error(w, "Organization already exists or any other error : "+err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "Organization already exists", http.StatusForbidden)
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

func retrievePaymentMethod(w http.ResponseWriter, r *http.Request) {
	org := r.Context().Value(ctxOrgKey)
	organization, ok := org.(Organization)
	if !ok {
		http.Error(w, "invalid organization context", http.StatusInternalServerError)
		return
	}

	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(organization.StripeID),
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

func handleCreatePaymentMethod(w http.ResponseWriter, r *http.Request) {

}

func getSubscriptionInfo(w http.ResponseWriter, r *http.Request) {
	org := r.Context().Value(ctxOrgKey)
	organization, ok := org.(Organization)
	if !ok {
		http.Error(w, "invalid organization context", http.StatusInternalServerError)
		return
	}
	params := &stripe.SubscriptionParams{}
	s, err := sub.Get(organization.StripeSubID, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		log.Printf("sub.New: %v", err)
		return
	}
	s.LatestInvoice, err = invoice.Get(s.LatestInvoice.ID, nil)
	if err != nil {
		http.Error(w, "Can't able to get invoice error:"+err.Error(), http.StatusInternalServerError)
		return
	}
	s.LatestInvoice.PaymentIntent, err = paymentintent.Get(s.LatestInvoice.PaymentIntent.ID, nil)
	if err != nil {
		http.Error(w, "Can't able to get payment intent error:"+err.Error(), http.StatusInternalServerError)
		return
	}

	var clientSecret string = ""
	if s.LatestInvoice.PaymentIntent.ClientSecret != "" {
		clientSecret = s.LatestInvoice.PaymentIntent.ClientSecret
	}

	writeJSON(w, struct {
		SubscriptionID     string `json:"subscriptionId"`
		SubscriptionStatus string `json:"subscriptionStatus"`
		ClientSecret       string `json:"clientSecret"`
	}{
		SubscriptionID:     s.ID,
		SubscriptionStatus: string(s.Status),
		ClientSecret:       clientSecret,
	})
}

func createSubscription(w http.ResponseWriter, r *http.Request) {
	org := r.Context().Value(ctxOrgKey)
	organization, ok := org.(Organization)
	if !ok {
		http.Error(w, "invalid organization context", http.StatusInternalServerError)
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body "+err.Error(), http.StatusUnprocessableEntity)
		return
	}

	items := getSubItemsPrice(req.Plan)

	if items == nil {
		http.Error(w, "Invalid plan :"+req.Plan, http.StatusUnprocessableEntity)
		return
	}

	// Automatically save the payment method to the subscription
	// when the first payment is successful.
	paymentSettings := &stripe.SubscriptionPaymentSettingsParams{
		SaveDefaultPaymentMethod: stripe.String("on_subscription"),
	}

	subscriptionParams := &stripe.SubscriptionParams{
		Customer:        stripe.String(organization.StripeID),
		Items:           items,
		PaymentSettings: paymentSettings,
		PaymentBehavior: stripe.String("default_incomplete"),
	}
	subscriptionParams.AddExpand("latest_invoice.payment_intent")

	if subID := strings.TrimSpace(organization.StripeSubID); subID != "" {
		if _, err := sub.Cancel(subID, nil); err != nil {
			http.Error(w, "Failed to cancel present subscription "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	s, err := sub.New(subscriptionParams)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		log.Printf("sub.New: %v", err)
		return
	}

	if err := createSubForOrg(*s, organization.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var clientSecret string
	if (s.LatestInvoice.PaymentIntent != nil) && (s.LatestInvoice.PaymentIntent.ClientSecret != "") {
		clientSecret = s.LatestInvoice.PaymentIntent.ClientSecret
	}
	writeJSON(w, struct {
		SubscriptionID     string `json:"subscriptionId"`
		SubscriptionStatus string `json:"subscriptionStatus"`
		ClientSecret       string `json:"clientSecret"`
	}{
		SubscriptionID:     s.ID,
		SubscriptionStatus: string(s.Status),
		ClientSecret:       clientSecret,
	})
}

func cancelSubscription(w http.ResponseWriter, r *http.Request) {
	org := r.Context().Value(ctxOrgKey)
	organization, ok := org.(Organization)
	if !ok {
		http.Error(w, "invalid organization context", http.StatusInternalServerError)
		return
	}

	s, err := sub.Cancel(organization.StripeSubID, nil)

	if err != nil {
		if strings.Contains(err.Error(), "resource_missing") {
			deleteSubByOrgId(organization.ID)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := deleteSub(*s); err != nil {
		http.Error(w, "Unsubscribe but failed to remove recode "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, "")
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

func updateSubscription(w http.ResponseWriter, r *http.Request) {
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

	switch event.Type {
	case "checkout.session.completed":
		cust, err := customer.Get(event.GetObjectValue("customer"), nil)
		if err != nil {
			log.Printf("customer.Get: %v", err)
			return
		}

		if event.GetObjectValue("display_items", "0", "custom") != "" &&
			event.GetObjectValue("display_items", "0", "custom", "name") == "Pasha e-book" {
			fmt.Printf("🔔 Customer is subscribed and bought an e-book! Send the e-book to %s\n\n", cust.Email)
		} else {
			fmt.Printf("🔔 Customer is subscribed but did not buy an e-book.\n\n")
		}
	case "customer.subscription.updated",
		"customer.subscription.created",
		"customer.subscription.deleted",
		"customer.subscription.paused",
		"customer.subscription.pending_update_applied",
		"customer.subscription.pending_update_expired",
		"customer.subscription.resumed",
		"customer.subscription.trial_will_end":

		subBytes, err := json.Marshal(event.Data.Object)
		if err != nil {
			errS := fmt.Sprintf(`{ "message" : "Failed to marshal event.Data.Object",  "error" : "%s", "inputdata" :"%+v" }`, err.Error(), event.Data.Object)
			fmt.Println(errS)
			// http.Error(w, errS, http.StatusInternalServerError)
			return
		}
		var sub stripe.Subscription

		if err := json.Unmarshal(subBytes, &sub); err != nil {
			errS := fmt.Sprintf(`{ "message" : "Failed to unmarshal subBytes",  "error" : "%s", "inputdata" : "%v" `, err.Error(), subBytes)
			fmt.Println(errS)
			// http.Error(w, errS, http.StatusInternalServerError)
			return
		}
		fmt.Println(event.Type)
		fmt.Println(sub)
		switch event.Type {
		case
			"customer.subscription.created":
			if err := createSub(sub); err != nil {
				fmt.Println(err.Error())
				return
			}
		case
			"customer.subscription.updated",
			"customer.subscription.paused",
			"customer.subscription.pending_update_applied",
			"customer.subscription.pending_update_expired",
			"customer.subscription.resumed",
			"customer.subscription.trial_will_end":
			switch sub.Status {
			// case "incomplete_expired":
			// 	if err := deleteSub(sub); err != nil {
			// 		fmt.Println(err.Error())
			// 		return
			// 	}
			default:
				if err := updateSub(sub); err != nil {
					fmt.Println(err.Error())
					return
				}
			}
		case "customer.subscription.deleted":
			if err := deleteSub(sub); err != nil {
				fmt.Println(err.Error())
				return
			}
		}
	default:
		// fmt.Printf("Got Unhandled event event type : %s , \n request body %+v\n\n========\n\n", event.Type, event)
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
	_ = json.Unmarshal(org.PlansByte, &org.Plans)
	return org, err
}

func checkOrganization(name, email string) (Organization, error) {
	var org Organization
	err := db.Get(&org, "SELECT * FROM  organization WHERE name=$1 and email=$2 LIMIT 1 ", name, email)
	return org, err
}

func getSubItemsPrice(planName string) []*stripe.SubscriptionItemsParams {
	if priceId, ok := subPlans[planName]; ok {
		var sip []*stripe.SubscriptionItemsParams
		sip = append(sip, &stripe.SubscriptionItemsParams{Price: stripe.String(priceId)})
		return sip
	}
	return nil
}

func createSub(sub stripe.Subscription) error {
	query := `
	UPDATE organization
	SET
		stripe_sub = ?,
		sub_status = ?,
		plans = ?
	WHERE
		stripe_id = ? ;
	`
	var plans []Plan
	for _, item := range sub.Items.Data {
		prod := Product{
			ID: item.Plan.Product.ID,
		}
		params := &stripe.ProductParams{}
		result, err := product.Get(item.Plan.Product.ID, params)
		if err == nil {
			prod.Name = result.Name
			prod.Active = result.Active
			prod.Description = result.Description
		}

		plan := Plan{
			ID:       item.Plan.ID,
			SiID:     item.ID,
			SubID:    item.Subscription,
			Active:   item.Plan.Active,
			Quantity: item.Quantity,
			Product:  prod,
		}

		plans = append(plans, plan)
	}
	plansByte, err := json.Marshal(plans)
	if err != nil {
		errS := fmt.Errorf(`{ "message" : "Failed to marshal plans",  "error" : "%s", "inputdata" : "%v" `, err.Error(), plans)
		fmt.Println(errS)
		return errS
	}

	if _, err := db.ExecContext(context.Background(), query, sub.ID, sub.Status, plansByte, sub.Customer.ID); err != nil {
		return err
	}
	return nil
}

func updateSub(sub stripe.Subscription) error {
	query := `
	UPDATE organization
	SET
		stripe_sub = ?,
		sub_status = ?,
		plans = ?
	WHERE
		stripe_id = ? AND stripe_sub = ?;
	`
	var plans []Plan
	for _, item := range sub.Items.Data {
		prod := Product{
			ID: item.Plan.Product.ID,
		}
		params := &stripe.ProductParams{}
		result, err := product.Get(item.Plan.Product.ID, params)
		if err == nil {
			prod.Name = result.Name
			prod.Active = result.Active
			prod.Description = result.Description
		}

		plan := Plan{
			ID:       item.Plan.ID,
			SiID:     item.ID,
			SubID:    item.Subscription,
			Active:   item.Plan.Active,
			Quantity: item.Quantity,
			Product:  prod,
		}

		plans = append(plans, plan)
	}
	plansByte, err := json.Marshal(plans)
	if err != nil {
		errS := fmt.Errorf(`{ "message" : "Failed to marshal plans",  "error" : "%s", "inputdata" : "%v" `, err.Error(), plans)
		fmt.Println(errS)
		return errS
	}

	if _, err := db.ExecContext(context.Background(), query, sub.ID, sub.Status, plansByte, sub.Customer.ID, sub.ID); err != nil {
		return err
	}
	return nil
}

func createSubForOrg(sub stripe.Subscription, orgID int) error {
	query := `
	UPDATE organization
	SET
		stripe_sub = ?,
		sub_status = ?,
		plans = ?
	WHERE
		id = ? ;
	`

	plans := getSubPlans(sub)
	plansByte, err := json.Marshal(plans)
	if err != nil {
		errS := fmt.Errorf(`{ "message" : "Failed to marshal plans",  "error" : "%s", "inputdata" : "%v" `, err.Error(), plans)
		fmt.Println(errS)
		return errS
	}
	if _, err := db.ExecContext(context.Background(), query, sub.ID, sub.Status, plansByte, orgID); err != nil {
		return err
	}
	return nil
}

func deleteSub(sub stripe.Subscription) error {
	query := `
	UPDATE organization
	SET
		stripe_sub = '',
		sub_status = '',
		plans = NULL
	WHERE
		stripe_sub = ? ;
	`
	if _, err := db.ExecContext(context.Background(), query, sub.ID); err != nil {
		return err
	}
	return nil
}

func deleteSubByOrgId(orgId int) error {
	query := `
	UPDATE organization
	SET
		stripe_sub = '',
		sub_status = '',
		plans = NULL
	WHERE
		id = ? ;
	`
	if _, err := db.ExecContext(context.Background(), query, orgId); err != nil {
		return err
	}
	return nil
}

func getSubPlans(sub stripe.Subscription) []Plan {
	var plans []Plan
	for _, item := range sub.Items.Data {
		prod := Product{
			ID: item.Plan.Product.ID,
		}
		params := &stripe.ProductParams{}
		result, err := product.Get(item.Plan.Product.ID, params)
		if err == nil {
			prod.Name = result.Name
			prod.Active = result.Active
			prod.Description = result.Description
		}

		plan := Plan{
			ID:       item.Plan.ID,
			SiID:     item.ID,
			SubID:    item.Subscription,
			Active:   item.Plan.Active,
			Quantity: item.Quantity,
			Amount:   item.Plan.Amount,
			Product:  prod,
		}

		plans = append(plans, plan)
	}
	return plans
}

func getPrice(id string) (*stripe.Price, error) {
	pr, err := price.Get(id, nil)
	if err != nil {
		return pr, err
	}
	pr.Product, err = product.Get(pr.Product.ID, nil)
	return pr, err
}
