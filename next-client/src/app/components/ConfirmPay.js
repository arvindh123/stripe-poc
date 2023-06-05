

export const ConfirmPay = async (orgId, stripe, setMessage) => {
    const reUrl = orgId ? "organization/" + orgId : ""
    const pis = await stripe?.confirmPayment({
        elements: elements,
        confirmParams: {
            return_url: `${window.location.origin}/${reUrl}`
        },
        redirect: "always",
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
