"use client"
import { useEffect, useState } from "react";
import { StripeElements } from "@stripe/stripe-js";
import { Button } from "flowbite-react"
import { useStripe, useElements, PaymentElement, CardElement } from "@stripe/react-stripe-js"

export default function CheckoutForm({ orgId, payment }: {orgId: string, payment: string}) {
    const stripe = useStripe()
    const elements = useElements()

    const [message, setMessage] = useState<string | undefined>("")
    const [processed, setProcessed] = useState(false)
    const [isProcessing, setIsProcessing] = useState(false)
    const [checking, setChecking] = useState(true)

    const checkPay = async (): Promise<Boolean> => {
        const pi = await stripe?.retrievePaymentIntent(payment)
        if (pi?.error) {
            setMessage("Failed to get payment : " + pi.error.message)
            return true
        }
        if (pi?.paymentIntent) {
            switch(pi.paymentIntent.status) {
                case "canceled":
                    setMessage("Payment Failed")
                    setProcessed(true)
                    return true
                case "succeeded":
                    setMessage("Payment succeeded")
                    setProcessed(true)
                    return true
                case "processing":
                    setMessage("Still processing payment, Please check after few minutes")
                    setProcessed(true)
                    return true
                case "requires_action":
                    if ((pi?.paymentIntent?.next_action)) {
                        const pi2 = await stripe?.handleNextAction({ clientSecret: payment })
                         console.log("pi2", pi2)
                        if (pi2?.error) {
                            setMessage("Payment Failed : " + pi2.error.message)
                            return true
                        }
                        if (pi2?.paymentIntent?.status) {
                            switch(pi2.paymentIntent.status) {
                                case "canceled":
                                    setMessage("Payment Failed")
                                    setProcessed(true)
                                    return true
                                case "succeeded":
                                    setMessage("Payment succeeded")
                                    setProcessed(true)
                                    return true
                                case "processing":
                                    setMessage("Still processing payment, Please check after few minutes")
                                    setProcessed(true)
                                    return true
                                default:
                                    return true
                            }
                        }
                    }
                case "requires_payment_method":
                    return true
                case "requires_capture":
                    return true
                case "requires_confirmation":
                    return true
                default:
                    return true
            }
        }
        return false
    }


    const confirmPay = async () => {
        const reUrl = orgId ? "organization/" + orgId : ""
        const pis = await stripe?.confirmPayment({
            elements: elements,
            confirmParams: {
                return_url: `${window.location.origin}/${reUrl}`
            },
            // redirect: "if_required",
        })
        if (pis?.error) {
            if ((pis.error.payment_intent) && (pis.error.payment_intent.status)) {
                switch (pis.error.payment_intent.status) {
                    case "succeeded":
                        setMessage("Paid already")
                        break;
                    case "canceled":
                        setMessage("Payment Indent Canceled")
                        break;
                    default:
                        setMessage(pis.error.message + "\n\npayment indent status : " + pis.error.payment_intent.status)
                }
            } else {
                setMessage(pis.error.message)
            }
        }
    }

    useEffect(() => {
        (async () => {
            const done = await checkPay()
            setChecking(!done)
        })();
        return () => { }

    },[stripe])

    const handleSubmit = async (e: React.FormEvent<HTMLFormElement>): Promise<void> => {
        e.preventDefault();
        if (!stripe || !elements) {
            return
        }
        setIsProcessing(true)
        await confirmPay()
        setIsProcessing(false)
    }

    return (
        <>
            <form className="flex max-w-md flex-col gap-4 items-center content-center	" onSubmit={handleSubmit} >
                <h1 className="font-extrabold text-transparent text-6xl bg-clip-text bg-gradient-to-r from-purple-400 to-pink-600" >
                    Checkout
                </h1>
                {
                    (checking ||  !elements) ?
                        <div role="status" className="absolute content-center items-center	 -translate-x-1/2 -translate-y-1/2 top-1/4 left-1/2">
                            <svg aria-hidden="true" className="w-8 h-8 mr-2 text-gray-200 animate-spin dark:text-gray-600 fill-blue-600" viewBox="0 0 100 101" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M100 50.5908C100 78.2051 77.6142 100.591 50 100.591C22.3858 100.591 0 78.2051 0 50.5908C0 22.9766 22.3858 0.59082 50 0.59082C77.6142 0.59082 100 22.9766 100 50.5908ZM9.08144 50.5908C9.08144 73.1895 27.4013 91.5094 50 91.5094C72.5987 91.5094 90.9186 73.1895 90.9186 50.5908C90.9186 27.9921 72.5987 9.67226 50 9.67226C27.4013 9.67226 9.08144 27.9921 9.08144 50.5908Z" fill="currentColor" /><path d="M93.9676 39.0409C96.393 38.4038 97.8624 35.9116 97.0079 33.5539C95.2932 28.8227 92.871 24.3692 89.8167 20.348C85.8452 15.1192 80.8826 10.7238 75.2124 7.41289C69.5422 4.10194 63.2754 1.94025 56.7698 1.05124C51.7666 0.367541 46.6976 0.446843 41.7345 1.27873C39.2613 1.69328 37.813 4.19778 38.4501 6.62326C39.0873 9.04874 41.5694 10.4717 44.0505 10.1071C47.8511 9.54855 51.7191 9.52689 55.5402 10.0491C60.8642 10.7766 65.9928 12.5457 70.6331 15.2552C75.2735 17.9648 79.3347 21.5619 82.5849 25.841C84.9175 28.9121 86.7997 32.2913 88.1811 35.8758C89.083 38.2158 91.5421 39.6781 93.9676 39.0409Z" fill="currentFill" /></svg>
                            <h1 className="font-extrabold text-transparent text-1xl bg-clip-text bg-white">Checking Status...</h1>
                        </div>
                        :

                        <>
                        {
                            processed ?
                            null :
                            <>
                                <PaymentElement />
                                <Button gradientDuoTone="purpleToBlue" disabled={isProcessing} isProcessing={isProcessing} type="submit">
                                    <p>
                                        {isProcessing ? "Processing..." : "Pay Now"}
                                    </p>
                                </Button>
                            </>
                        }
                        </>
                }

                <p className=" text-transparent text-2xl bg-clip-text bg-white">
                    {message}
                </p>
            </form>

        </>

    )
}