'use client';
import { Card, Button } from 'flowbite-react';
import { useRouter } from 'next/navigation';
import { Toast } from 'flowbite-react';
import { RxCross1 } from 'react-icons/rx'
import { useState } from 'react';
import { TOrg, TPrice } from '../../interface/all'
import { getSub } from '../../backendAPI/getAllOrg'
import { MdOutlineKeyboardBackspace } from 'react-icons/md'

export default function Plan({ org, plans }: { org: TOrg, plans: TPrice[] }) {

    const [errMsg, setErrMsg] = useState("")
    const [isProcessing, setIsProcessing] = useState(false)
    const router = useRouter();

    const handleCompletePayment = async (org: TOrg) => {
        const res = await getSub(org.id)
        if ((res?.clientSecret) && (res?.clientSecret !== "")) {
            setIsProcessing(true)
            router.push(`/checkout?payment=${res?.clientSecret}&id=${org.id}`)
            return
        }
    }

    const getCurrentPlan = ( org: TOrg, plans: TPrice[]) => {
        return plans.find( (price) => {
            return ((org.plans) && (org.plans.some(plan => plan.id === price.id)))
        })
    }
    const handleSubmit = async (plan: string): Promise<void> => {
        setIsProcessing(true)
        const method = getCurrentPlan(org, plans) ? "put" : "post"
        const apiURL = process.env.APIURL ? process.env.APIURL : process.env.NEXT_PUBLIC_APIURL ? process.env.NEXT_PUBLIC_APIURL : `http://localhost:8080`
        try {
            const result = await fetch(`${apiURL}/organization/${org.id}/sub`, {
                body: JSON.stringify({ plan: `${plan}` }),
                method: method,
                headers: {
                    "content-type": "application/json",
                },
            })
            if (result.status == 200) {
                const resp = await result.json()
                if ((resp?.clientSecret) && (resp?.clientSecret != "")) {
                    router.push(`/checkout?payment=${resp?.clientSecret}&id=${org.id}`)
                    return
                } else {
                    router.push(`/organization/${org.id}`)
                    return
                }
            } else {
                const body = await result.text()
                setErrMsg(body)
            }

        } catch (err) {
            setErrMsg((err as Error).message)
        }
        setIsProcessing(false)

    };
    return (
        <>

            <div className="relative h-16 w-1/3 flex align-middle items-center justify-center ">
                <button onClick={() => { router.push(`/organization/${org.id}`) }} className='absolute left-0 top-0 h-1 w-16'>
                    <MdOutlineKeyboardBackspace className="text-4xl  text-gray-100 dark:text-gray-900" title='Back' />
                    <p className="absolute left-10 top-1 h-1 w-16 font-normal text-gray-100 dark:text-gray-900"> Back </p>
                </button>
            </div>

            <div className='grid grid-cols-2 gap-32'>

                <>
                    {
                        plans.sort((a, b) => a.unit_amount - b.unit_amount).map((price, index) => {
                            return (
                                <Card key={index} className='max-w-xl'>
                                    <h5 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
                                        <p>
                                            {price.product.name}
                                        </p>
                                    </h5>
                                    <p className="font-normal text-gray-900 dark:text-gray-100">
                                        Users Limit : {price.metadata.users_limit}
                                    </p>
                                    <p className="font-normal text-gray-900 dark:text-gray-100">
                                        Price : {price.unit_amount / 100}/{price.recurring.interval}
                                    </p>
                                    {
                                        ((org.plans) && (org.plans.some(plan => plan.id === price.id))) ?
                                            <>
                                                {
                                                    org.sub_status === "incomplete" ?
                                                        <Button gradientDuoTone="greenToBlue" disabled={isProcessing} onClick={() => handleCompletePayment(org)}>
                                                            <p>
                                                                Complete Subscription Payment
                                                            </p>
                                                        </Button>
                                                        :
                                                        <Button gradientDuoTone="purpleToBlue" disabled={true}>
                                                            <p>
                                                                Current Plan
                                                            </p>
                                                        </Button>
                                                }

                                            </>

                                            :
                                            <Button gradientDuoTone="purpleToBlue" onClick={() => handleSubmit(price.nickname)} disabled={isProcessing}>
                                                <p>
                                                    Subscribe
                                                </p>
                                            </Button>
                                    }

                                </Card>
                            )

                        })
                    }
                </>

            </div>
            {
                errMsg !== "" ?
                    <Toast>
                        <div className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-red-100 text-red-500 dark:bg-red-800 dark:text-red-200">
                            <RxCross1 className="h-5 w-5" />
                        </div>
                        <div className="ml-3 text-sm font-normal">
                            Could not create subscription for Organization , Error : {errMsg}
                        </div>
                        <Toast.Toggle onClick={() => setErrMsg("")} />
                    </Toast>
                    : null
            }
        </>



    )

}