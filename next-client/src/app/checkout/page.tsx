"use client"
import { Elements, PaymentElement } from '@stripe/react-stripe-js';
import { Stripe, loadStripe} from '@stripe/stripe-js';
import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import CheckoutForm from "../components/CheckoutForm"
import Link from 'next/link';


export default function  Checkout() {
    const [stripePromise, setStripePromise] = useState< Promise<Stripe | null> | null>(null)
    const searchParams = useSearchParams();
    const payment = searchParams.get("payment")
    const orgId = searchParams.get("id")
    useEffect(() => {
      fetch("http://localhost:8080/config").then( async(r) => {
        const { publishableKey } = await r.json()
        setStripePromise(loadStripe(publishableKey))
      })

    }, [])
    const appearance = {
      theme: 'night',
      labels: 'floating'
    };


  return (
    <>
      {
        payment ?
          <Elements stripe={stripePromise} options={ { clientSecret: (payment as string)}}  appearance={appearance} la>
            <CheckoutForm orgId={(orgId as string)} payment={(payment as string)}/>
          </Elements>
          :
          null
      }
      <div className='align-left'>

        <p>
          Test Card with 3d Secure
        </p>
        <p>
          Card Number : 4000002760003184
        </p>
        <p>
          Expiry Date: Any future date 9/33
        </p>
        <p>
          CVC: Any number 123
        </p>
        <Link href="https://stripe.com/docs/testing" target="_blank" className='font-extrabold text-transparent text-xl bg-clip-text bg-gradient-to-r from-purple-400 to-pink-600'>
            Click here to know about test card numbers
        </Link>
        <Link href="https://stripe.com/docs/india-recurring-payments?" target="_blank" className='font-extrabold text-transparent text-xl bg-clip-text bg-gradient-to-r from-purple-400 to-pink-600'>
            Click here to know about e-Mandate required for India
        </Link>
      </div>

    </>
)

};